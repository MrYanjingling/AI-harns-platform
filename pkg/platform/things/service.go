package things

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/emqx"
	"harns-platform/pkg/logging"
	"harns-platform/pkg/platform/principal"
	"harns-platform/pkg/platform/things/v1"
)

const (
	tokenCachePrefix          = "iot:thing:token:"
	tokenCacheTTL             = 30 * time.Minute
	shadowKeyPrefix           = "iot:thing:shadow:"
	relationCleanupTimeout    = 30 * time.Second
	mqttKickTimeout           = 60 * time.Second
	thingCacheDeleteChunkSize = 500
)

// Service is the business layer of the things module.
type Service struct {
	store           *Store
	redis           redis.UniversalClient
	onlineTTL       time.Duration
	emqx            *emqx.Service
	models          PropertyDefinitionResolver
	timeSeries      ThingTimeSeriesQuerier
	relationCleaner ThingRelationCleaner
	denyCache       *authDenyCache
}

// NewService builds a things service. A non-positive TTL falls back to 60s.
func NewService(
	store *Store,
	rdb redis.UniversalClient,
	onlineTTL time.Duration,
	emqxSvc *emqx.Service,
	models PropertyDefinitionResolver,
	timeSeries ThingTimeSeriesQuerier,
	denyCacheCfg AuthDenyCacheConfig,
) *Service {
	if onlineTTL <= 0 {
		onlineTTL = 60 * time.Second
	}
	return &Service{
		store:      store,
		redis:      rdb,
		onlineTTL:  onlineTTL,
		emqx:       emqxSvc,
		models:     models,
		timeSeries: timeSeries,
		denyCache:  newAuthDenyCache(denyCacheCfg),
	}
}

// Create creates a batch of things.
func (s *Service) Create(ctx context.Context, req *v1.CreateThingsReq) ([]*v1.ThingDetailVo, error) {
	defer logging.Trace("things.Create", "thingModelId", req.ThingModelID, "count", len(req.Things))()

	modelID := strings.TrimSpace(req.ThingModelID)
	issues := make([]FieldIssue, 0, 2)
	if modelID == "" {
		issues = append(issues, FieldIssue{Field: "thingModelId", MsgID: "validate.thing_model_id_required"})
	}
	switch {
	case len(req.Things) == 0:
		issues = append(issues, FieldIssue{Field: "things", MsgID: "validate.thing_batch_empty"})
	case len(req.Things) > MaxBatchSize:
		issues = append(issues, FieldIssue{Field: "things", MsgID: "validate.thing_batch_too_large", Args: []interface{}{MaxBatchSize}})
	}
	if verr := NewValidationError(issues); verr != nil {
		return nil, verr
	}

	items, verr := validateCreateItems(req.Things)
	if verr != nil {
		return nil, verr
	}
	if exists, err := s.store.ThingModelExists(ctx, modelID); err != nil {
		return nil, response.ErrInternal(err)
	} else if !exists {
		return nil, response.ErrThingModelNotFound(modelID)
	}
	featureIDs, err := s.store.GetThingModelFeatureIdentifiers(ctx, modelID)
	if err != nil {
		return nil, response.ErrInternal(err)
	}

	names := make([]string, 0, len(items))
	userKeys := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.DeviceName)
		if item.DeviceKey != "" {
			userKeys = append(userKeys, item.DeviceKey)
		}
	}
	existingNames, err := s.store.ExistingDeviceNames(ctx, modelID, names)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	existingKeys, err := s.store.ExistingDeviceKeys(ctx, userKeys)
	if err != nil {
		return nil, response.ErrInternal(err)
	}

	now := time.Now()
	operator := principal.UserID(ctx)
	usedNames := make(map[string]struct{}, len(items))
	usedKeys := make(map[string]struct{}, len(items))
	result := make([]*Thing, 0, len(items))
	conflicts := make([]FieldIssue, 0)

	for i := range items {
		path := fmt.Sprintf("things[%d]", i)
		item := &items[i]

		nameKey := strings.ToLower(item.DeviceName)
		if _, ok := usedNames[nameKey]; ok {
			conflicts = append(conflicts, FieldIssue{Field: path + ".deviceName", MsgID: "validate.thing_device_name_duplicate", Args: []interface{}{item.DeviceName}})
		} else if _, ok := existingNames[nameKey]; ok {
			conflicts = append(conflicts, FieldIssue{Field: path + ".deviceName", MsgID: "validate.thing_device_name_duplicate", Args: []interface{}{item.DeviceName}})
		}
		usedNames[nameKey] = struct{}{}

		for featureKey := range item.Features {
			if _, ok := featureIDs[featureKey]; !ok {
				conflicts = append(conflicts, FieldIssue{
					Field: path + ".features." + featureKey,
					MsgID: "validate.thing_feature_unknown",
					Args:  []interface{}{featureKey},
				})
			}
		}

		key := item.DeviceKey
		if key == "" {
			key, err = generateUniqueKey(usedKeys, existingKeys)
			if err != nil {
				return nil, response.ErrInternal(err)
			}
		} else if _, ok := usedKeys[key]; ok {
			conflicts = append(conflicts, FieldIssue{Field: path + ".deviceKey", MsgID: "validate.thing_device_key_duplicate", Args: []interface{}{key}})
		} else if _, ok := existingKeys[key]; ok {
			conflicts = append(conflicts, FieldIssue{Field: path + ".deviceKey", MsgID: "validate.thing_device_key_duplicate", Args: []interface{}{key}})
		}
		usedKeys[key] = struct{}{}

		secret, err := NewDeviceSecret()
		if err != nil {
			return nil, response.ErrInternal(err)
		}
		token, err := NewToken()
		if err != nil {
			return nil, response.ErrInternal(err)
		}

		id := NewID()
		result = append(result, &Thing{
			Meta: apis.Meta{
				ID:          id,
				Version:     0,
				CreatedUser: operator,
				UpdatedUser: operator,
				CreatedTime: now,
				UpdatedTime: now,
			},
			ThingModelID: modelID,
			DeviceName:   item.DeviceName,
			DeviceKey:    key,
			DeviceSecret: secret,
			Token:        token,
			UpTopic:      UpTopicFor(key),
			DownTopic:    DownTopicFor(key),
			DeviceDesc:   item.DeviceDesc,
			Features:     item.Features,
			Image:        item.Image,
			MQTTClientID: id,
			MQTTUsername: key,
			MQTTPassword: secret,
			Enable:       1,
			Debug:        debugValue(item.Debug),
		})
	}
	if verr := NewValidationError(conflicts); verr != nil {
		return nil, verr
	}

	if err := s.store.CreateThings(ctx, result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, response.ErrThingModelNotFound(modelID)
		}
		return nil, response.ErrInternal(err)
	}

	out := make([]*v1.ThingDetailVo, 0, len(result))
	for _, item := range result {
		out = append(out, toThingDetailVo(item))
	}
	return out, nil
}

// List returns a page of things with online status from Redis.
func (s *Service) List(ctx context.Context, q v1.ThingQuery) (*apis.PageResp[*v1.ThingVo], error) {
	defer logging.Trace("things.List", "keyword", q.Keyword, "thingModelId", q.ThingModelID, "page", q.Page, "pageSize", q.PageSize)()

	q.Normalize()
	items, total, err := s.store.ListThings(ctx, q.PageQuery, q.Keyword, q.ThingModelID)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	scores := s.lastActiveScores(ctx, ids)
	now := time.Now().UnixMilli()

	list := make([]*v1.ThingVo, 0, len(items))
	for _, item := range items {
		vo := toThingVo(item)
		applyOnline(vo, item.Enable, scores[item.ID], now, s.onlineTTL)
		list = append(list, vo)
	}
	return apis.NewPageResp(list, total, q.PageQuery), nil
}

// GetBasic returns a thing with credentials, model, creator/updater and connection info.
func (s *Service) GetBasic(ctx context.Context, id string) (*v1.ThingDetailVo, error) {
	defer logging.Trace("things.GetBasic", "id", id)()

	item, err := s.store.GetThingBasic(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrThingNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}

	vo := toThingBasicVo(item)
	vo.MQTTBroker = s.emqx.Broker()

	properties, err := s.store.GetThingModelPropertySamples(ctx, item.ThingModelID, uploadFormatPropertyLimit)
	if err != nil {
		logging.ErrorS(err, "Failed to load thing model property samples", "thingId", id, "thingModelId", item.ThingModelID)
	} else if uploadFormat, err := buildUploadFormat(properties, time.Now().UnixMilli()); err != nil {
		logging.ErrorS(err, "Failed to build upload format", "thingId", id, "thingModelId", item.ThingModelID)
	} else {
		vo.UploadFormat = uploadFormat
	}

	scores := s.lastActiveScores(ctx, []string{item.ID})
	applyOnline(&vo.ThingVo, item.Enable, scores[item.ID], time.Now().UnixMilli(), s.onlineTTL)
	return vo, nil
}

// GetFeatureValues returns raw device feature values keyed by feature identifier.
func (s *Service) GetFeatureValues(ctx context.Context, thingID string) (map[string]json.RawMessage, error) {
	defer logging.Trace("things.GetFeatureValues", "thingId", thingID)()

	item, err := s.getEntity(ctx, thingID)
	if err != nil {
		return nil, err
	}
	featureIDs, err := s.store.GetThingModelFeatureIdentifiers(ctx, item.ThingModelID)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return mergeFeatureValues(featureIDs, item.Features), nil
}

func mergeFeatureValues(featureIDs map[string]struct{}, values map[string]json.RawMessage) map[string]json.RawMessage {
	merged := make(map[string]json.RawMessage, len(featureIDs)+len(values))
	for identifier := range featureIDs {
		merged[identifier] = nil
	}
	for identifier, value := range values {
		merged[identifier] = value
	}
	return merged
}

// Authenticate validates the MQTT credentials used by the EMQX HTTP authenticator.
func (s *Service) Authenticate(ctx context.Context, req *v1.DeviceAuthReq) (*v1.DeviceAuthResp, error) {
	var clientID, username string
	if req != nil {
		clientID, username = req.ClientID, req.Username
	}
	defer logging.Trace("things.Authenticate", "clientid", clientID, "username", username)()

	if req == nil || req.ClientID == "" || req.Username == "" || req.Password == "" {
		return denyAuth(), nil
	}
	denyKey := authDenyCacheKey(req.ClientID, req.Username, req.Password)
	if s.denyCache.contains(denyKey) {
		return denyAuth(), nil
	}
	item, err := s.store.AuthenticateThing(ctx, req.ClientID, req.Username, req.Password)
	if errors.Is(err, sql.ErrNoRows) {
		s.denyCache.add(denyKey)
		logging.V(2).InfoS("Denied MQTT authentication for unknown credentials",
			"clientid", req.ClientID, "username", req.Username)
		return denyAuth(), nil
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if item.Enable != 1 {
		logging.Warnf("Denied MQTT authentication for disabled thing: clientid=%s username=%s thingId=%s",
			req.ClientID, req.Username, item.ID)
		return denyAuth(), nil
	}
	return buildAuthResponse(item), nil
}

func denyAuth() *v1.DeviceAuthResp {
	return &v1.DeviceAuthResp{Result: v1.AuthResultDeny, Superuser: false}
}

func buildAuthResponse(item *AuthThing) *v1.DeviceAuthResp {
	return &v1.DeviceAuthResp{
		Result:    v1.AuthResultAllow,
		Superuser: false,
		ClientAttrs: map[string]string{
			ClientAttrsThingID:       item.ID,
			ClientAttrsThingModelID:  item.ThingModelID,
			ClientAttrsThingModelKey: item.ThingModelKey,
			ClientAttrsDeviceKey:     item.DeviceKey,
			ClientAttrsDeviceName:    item.DeviceName,
		},
		ACL: []v1.ACLRule{
			{
				Permission: v1.ACLPermissionAllow,
				Action:     v1.ACLActionAll,
				Topic:      item.UpTopic,
				QoS:        []int{0, 1, 2},
				Retain:     false,
			},
			{
				Permission: v1.ACLPermissionAllow,
				Action:     v1.ACLActionAll,
				Topic:      item.DownTopic,
				QoS:        []int{0, 1, 2},
				Retain:     false,
			},
		},
	}
}

// GetThingByToken resolves a thing identity by token, using Redis as a sliding
// 30-minute cache in front of MySQL.
func (s *Service) GetThingByToken(ctx context.Context, token string) (*ThingIdentity, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, response.ErrUnauthorized()
	}
	cacheKey := tokenCacheKey(token)
	if s.redis != nil {
		if raw, err := s.redis.GetEx(ctx, cacheKey, tokenCacheTTL).Bytes(); err == nil {
			var identity ThingIdentity
			if err := json.Unmarshal(raw, &identity); err == nil {
				return &identity, nil
			}
			logging.V(2).InfoS("Failed to decode thing token cache", "err", err)
		} else if !errors.Is(err, redis.Nil) {
			logging.V(2).InfoS("Failed to read thing token cache", "err", err)
		}
	}

	identity, err := s.store.GetThingByToken(ctx, token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrUnauthorized()
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if s.redis != nil {
		if data, err := json.Marshal(identity); err == nil {
			if err := s.redis.Set(ctx, cacheKey, data, tokenCacheTTL).Err(); err != nil {
				logging.V(2).InfoS("Failed to cache thing token", "err", err)
			}
		}
	}
	return identity, nil
}

// TouchThingOnline refreshes the online index. The shadow timestamp is
// maintained by property processing and uses the device data time instead.
func (s *Service) TouchThingOnline(ctx context.Context, thingID string, dataTime int64) {
	if s.redis == nil || thingID == "" {
		return
	}
	now := time.Now().UnixMilli()
	score := dataTime
	if score <= 0 || score > now {
		score = now
	}
	if err := s.redis.ZAddArgs(ctx, LastActiveKey, redis.ZAddArgs{
		GT:      true,
		Ch:      true,
		Members: []redis.Z{{Score: float64(score), Member: thingID}},
	}).Err(); err != nil {
		logging.V(2).InfoS("Failed to refresh thing online index", "err", err)
	}
}

func tokenCacheKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return tokenCachePrefix + hex.EncodeToString(sum[:])
}

func (s *Service) invalidateTokenCache(ctx context.Context, token string) {
	if s.redis == nil || token == "" {
		return
	}
	if err := s.redis.Del(ctx, tokenCacheKey(token)).Err(); err != nil {
		logging.V(2).InfoS("Failed to invalidate thing token cache", "err", err)
	}
}

// Update changes the mutable fields of a thing.
func (s *Service) Update(ctx context.Context, id string, req *v1.UpdateThingReq) (*v1.ThingDetailVo, error) {
	defer logging.Trace("things.Update", "id", id)()

	item, err := s.getEntity(ctx, id)
	if err != nil {
		return nil, err
	}

	issues := make([]FieldIssue, 0)
	if req.DeviceName != nil {
		name := strings.TrimSpace(*req.DeviceName)
		fieldIssues := validateDeviceName(name, "deviceName")
		issues = append(issues, fieldIssues...)
		if len(fieldIssues) == 0 && name != item.DeviceName {
			exists, err := s.store.ExistsDeviceName(ctx, item.ThingModelID, name, id)
			if err != nil {
				return nil, response.ErrInternal(err)
			}
			if exists {
				return nil, response.ErrThingDeviceNameExists(name)
			}
			item.DeviceName = name
		}
	}
	if req.DeviceDesc != nil {
		desc := strings.TrimSpace(*req.DeviceDesc)
		issues = append(issues, validateDeviceDesc(desc, "deviceDesc")...)
		item.DeviceDesc = desc
	}
	if req.MQTTClientID != nil {
		value := strings.TrimSpace(*req.MQTTClientID)
		fieldIssues := validateMQTTValue(value, "mqttClientId", "validate.thing_mqtt_client_id_invalid")
		issues = append(issues, fieldIssues...)
		if len(fieldIssues) == 0 && value != item.MQTTClientID {
			exists, err := s.store.ExistsMQTTClientID(ctx, value, id)
			if err != nil {
				return nil, response.ErrInternal(err)
			}
			if exists {
				return nil, response.ErrThingMQTTClientIDExists(value)
			}
			item.MQTTClientID = value
		}
	}
	if req.MQTTUsername != nil {
		value := strings.TrimSpace(*req.MQTTUsername)
		fieldIssues := validateMQTTValue(value, "mqttUsername", "validate.thing_mqtt_username_invalid")
		issues = append(issues, fieldIssues...)
		if len(fieldIssues) == 0 && value != item.MQTTUsername {
			exists, err := s.store.ExistsMQTTUsername(ctx, value, id)
			if err != nil {
				return nil, response.ErrInternal(err)
			}
			if exists {
				return nil, response.ErrThingMQTTUsernameExists(value)
			}
			item.MQTTUsername = value
		}
	}
	if req.MQTTPassword != nil {
		value := strings.TrimSpace(*req.MQTTPassword)
		issues = append(issues, validateMQTTValue(value, "mqttPassword", "validate.thing_mqtt_password_invalid")...)
		item.MQTTPassword = value
	}
	if req.Enable != nil {
		if *req.Enable != 0 && *req.Enable != 1 {
			issues = append(issues, FieldIssue{Field: "enable", MsgID: "validate.thing_enable_invalid"})
		}
		item.Enable = *req.Enable
	}
	if req.Debug != nil {
		item.Debug = *req.Debug
	}
	if req.Image != nil {
		image := strings.TrimSpace(*req.Image)
		issues = append(issues, validateImageKey(image, "image")...)
		item.Image = image
	}
	if req.Features != nil {
		featureIDs, err := s.store.GetThingModelFeatureIdentifiers(ctx, item.ThingModelID)
		if err != nil {
			return nil, response.ErrInternal(err)
		}
		fieldIssues := make([]FieldIssue, 0)
		for featureKey := range *req.Features {
			if _, ok := featureIDs[featureKey]; !ok {
				fieldIssues = append(fieldIssues, FieldIssue{
					Field: "features." + featureKey,
					MsgID: "validate.thing_feature_unknown",
					Args:  []interface{}{featureKey},
				})
			}
		}
		issues = append(issues, fieldIssues...)
		if len(fieldIssues) == 0 {
			item.Features = *req.Features
		}
	}
	if verr := NewValidationError(issues); verr != nil {
		return nil, verr
	}

	item.UpdatedUser = principal.UserID(ctx)
	item.UpdatedTime = time.Now()
	n, err := s.store.UpdateThing(ctx, item)
	if err != nil {
		switch {
		case isDuplicateIndex(err, "uk_things_model_name"):
			return nil, response.ErrThingDeviceNameExists(item.DeviceName)
		case isDuplicateIndex(err, "uk_things_mqtt_client_id"):
			return nil, response.ErrThingMQTTClientIDExists(item.MQTTClientID)
		case isDuplicateIndex(err, "uk_things_mqtt_username"):
			return nil, response.ErrThingMQTTUsernameExists(item.MQTTUsername)
		default:
			return nil, response.ErrInternal(err)
		}
	}
	if n == 0 {
		return nil, response.ErrThingNotFound(id)
	}
	item.Version++
	s.invalidateTokenCache(ctx, item.Token)
	s.invalidateSnapshotCache(ctx, item.ID)

	vo := toThingDetailVo(item)
	scores := s.lastActiveScores(ctx, []string{item.ID})
	applyOnline(&vo.ThingVo, item.Enable, scores[item.ID], time.Now().UnixMilli(), s.onlineTTL)
	return vo, nil
}

// SetThingRelationCleaner injects the optional asynchronous cleaner for rule
// and alarm configuration relations.
func (s *Service) SetThingRelationCleaner(cleaner ThingRelationCleaner) {
	s.relationCleaner = cleaner
}

// Delete removes things, their caches, and asynchronously cleans related rule
// and alarm configuration relations. cascade is reserved and not applied yet.
func (s *Service) Delete(ctx context.Context, ids []string, cascade bool) (int64, error) {
	defer logging.Trace("things.Delete", "count", len(ids), "cascade", cascade)()

	cleaned := dedupeIDs(ids)
	if len(cleaned) == 0 {
		return 0, nil
	}
	deleted, refs, err := s.store.DeleteThings(ctx, cleaned)
	if err != nil {
		return 0, response.ErrInternal(err)
	}
	s.dispatchMQTTKick(refs)
	s.cleanupDeletedThingCaches(ctx, refs)
	s.dispatchThingRelationCleanup(refs)
	return deleted, nil
}

// KickMQTTClients disconnects MQTT clients for things deleted by another module.
func (s *Service) KickMQTTClients(ctx context.Context, clientIDs []string) error {
	if s.emqx == nil {
		return nil
	}
	return s.emqx.KickClients(ctx, clientIDs)
}

func (s *Service) dispatchMQTTKick(refs []DeletedThingRef) {
	if s.emqx == nil || len(refs) == 0 {
		return
	}
	clientIDs := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.MQTTClientID != "" {
			clientIDs = append(clientIDs, ref.MQTTClientID)
		}
	}
	if len(clientIDs) == 0 {
		return
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logging.ErrorS(fmt.Errorf("panic: %v", recovered), "MQTT kick panicked", "count", len(clientIDs))
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), mqttKickTimeout)
		defer cancel()
		if err := s.emqx.KickClients(ctx, clientIDs); err != nil {
			logging.ErrorS(err, "Failed to kick deleted thing MQTT clients", "count", len(clientIDs))
		}
	}()
}

// CleanupDeletedThingCaches removes caches for things deleted by another module.
func (s *Service) CleanupDeletedThingCaches(ctx context.Context, thingIDs, tokens []string) {
	refs := make([]DeletedThingRef, 0, len(thingIDs)+len(tokens))
	for _, id := range dedupeIDs(thingIDs) {
		refs = append(refs, DeletedThingRef{ID: id})
	}
	for _, token := range dedupeIDs(tokens) {
		refs = append(refs, DeletedThingRef{Token: token})
	}
	s.cleanupDeletedThingCaches(ctx, refs)
}

func (s *Service) cleanupDeletedThingCaches(ctx context.Context, refs []DeletedThingRef) {
	if s.redis == nil || len(refs) == 0 {
		return
	}
	keys, ids := deletedThingCacheKeys(refs)
	for start := 0; start < len(keys); start += thingCacheDeleteChunkSize {
		end := start + thingCacheDeleteChunkSize
		if end > len(keys) {
			end = len(keys)
		}
		if err := s.redis.Del(ctx, keys[start:end]...).Err(); err != nil {
			logging.V(2).InfoS("Failed to delete thing cache keys", "count", end-start, "err", err)
		}
	}
	if len(ids) > 0 {
		if err := s.redis.ZRem(ctx, LastActiveKey, toAny(ids)...).Err(); err != nil {
			logging.V(2).InfoS("Failed to remove thing online index", "count", len(ids), "err", err)
		}
	}
}

func deletedThingCacheKeys(refs []DeletedThingRef) ([]string, []string) {
	keys := make([]string, 0, len(refs)*3)
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Token != "" {
			keys = append(keys, tokenCacheKey(ref.Token))
		}
		if ref.ID != "" {
			keys = append(keys, snapshotCachePrefix+ref.ID, shadowKeyPrefix+ref.ID)
			ids = append(ids, ref.ID)
		}
	}
	return keys, ids
}

func (s *Service) dispatchThingRelationCleanup(refs []DeletedThingRef) {
	cleaner := s.relationCleaner
	if cleaner == nil || len(refs) == 0 {
		return
	}
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.ID != "" {
			ids = append(ids, ref.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logging.ErrorS(fmt.Errorf("panic: %v", recovered), "Thing relation cleanup panicked", "count", len(ids))
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), relationCleanupTimeout)
		defer cancel()
		if err := cleaner.CleanupThingRelations(ctx, ids); err != nil {
			logging.ErrorS(err, "Failed to clean thing relations", "count", len(ids))
		}
	}()
}

func (s *Service) getEntity(ctx context.Context, id string) (*Thing, error) {
	item, err := s.store.GetThing(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrThingNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return item, nil
}

func (s *Service) lastActiveScores(ctx context.Context, ids []string) map[string]float64 {
	out := make(map[string]float64, len(ids))
	if s.redis == nil || len(ids) == 0 {
		return out
	}
	values, err := s.redis.ZMScore(ctx, LastActiveKey, ids...).Result()
	if err != nil {
		logging.V(2).InfoS("Failed to read thing online index", "err", err)
		return out
	}
	for i, value := range values {
		if i < len(ids) && value > 0 {
			out[ids[i]] = value
		}
	}
	return out
}

func validateCreateItems(items []v1.CreateThingItem) ([]v1.CreateThingItem, *ValidationError) {
	out := make([]v1.CreateThingItem, 0, len(items))
	issues := make([]FieldIssue, 0)
	for i, item := range items {
		path := fmt.Sprintf("things[%d]", i)
		name := strings.TrimSpace(item.DeviceName)
		issues = append(issues, validateDeviceName(name, path+".deviceName")...)
		key := strings.TrimSpace(item.DeviceKey)
		if key != "" && !ValidDeviceKey(key) {
			issues = append(issues, FieldIssue{Field: path + ".deviceKey", MsgID: "validate.thing_device_key_invalid", Args: []interface{}{key}})
		}
		desc := strings.TrimSpace(item.DeviceDesc)
		issues = append(issues, validateDeviceDesc(desc, path+".deviceDesc")...)
		image := strings.TrimSpace(item.Image)
		issues = append(issues, validateImageKey(image, path+".image")...)
		out = append(out, v1.CreateThingItem{DeviceName: name, DeviceKey: key, DeviceDesc: desc, Features: item.Features, Image: image})
	}
	return out, NewValidationError(issues)
}

func validateDeviceName(name, field string) []FieldIssue {
	switch {
	case name == "":
		return []FieldIssue{{Field: field, MsgID: "validate.thing_device_name_required"}}
	case utf8.RuneCountInString(name) > 64:
		return []FieldIssue{{Field: field, MsgID: "validate.thing_device_name_invalid", Args: []interface{}{name}}}
	}
	return nil
}

func validateImageKey(image, field string) []FieldIssue {
	switch {
	case len(image) > 255:
		return []FieldIssue{{Field: field, MsgID: "validate.thing_image_invalid", Args: []interface{}{image}}}
	case image != "" && !strings.HasPrefix(image, "images/"):
		return []FieldIssue{{Field: field, MsgID: "validate.thing_image_invalid", Args: []interface{}{image}}}
	}
	return nil
}

func validateDeviceDesc(desc, field string) []FieldIssue {
	if utf8.RuneCountInString(desc) > 255 {
		return []FieldIssue{{Field: field, MsgID: "validate.thing_device_desc_invalid"}}
	}
	return nil
}

func validateMQTTValue(value, field, msgID string) []FieldIssue {
	if value == "" || utf8.RuneCountInString(value) > 64 {
		return []FieldIssue{{Field: field, MsgID: msgID, Args: []interface{}{value}}}
	}
	return nil
}

func generateUniqueKey(used, existing map[string]struct{}) (string, error) {
	for i := 0; i < 5; i++ {
		key, err := NewDeviceKey()
		if err != nil {
			return "", err
		}
		if _, ok := used[key]; ok {
			continue
		}
		if _, ok := existing[key]; ok {
			continue
		}
		return key, nil
	}
	return "", errors.New("failed to generate a unique device key")
}

func toThingVo(item *Thing) *v1.ThingVo {
	return &v1.ThingVo{
		ID:           item.ID,
		Version:      item.Version,
		ThingModelID: item.ThingModelID,
		DeviceName:   item.DeviceName,
		DeviceKey:    item.DeviceKey,
		UpTopic:      item.UpTopic,
		DownTopic:    item.DownTopic,
		DeviceDesc:   item.DeviceDesc,
		MQTTClientID: item.MQTTClientID,
		MQTTUsername: item.MQTTUsername,
		Enable:       item.Enable,
		Debug:        item.Debug,
		CreatedUser:  item.CreatedUser,
		UpdatedUser:  item.UpdatedUser,
		CreatedTime:  item.CreatedTime,
		UpdatedTime:  item.UpdatedTime,
	}
}

func toThingBasicVo(item *ThingBasic) *v1.ThingDetailVo {
	vo := toThingDetailVo(&item.Thing)
	vo.ThingModelKey = item.ThingModelKey
	vo.ThingModelName = item.ThingModelName
	if item.CreatedUser != "" {
		vo.CreatedBy = &v1.UserBriefVo{ID: item.CreatedUser, UserName: item.CreatedUserName, NickName: item.CreatedUserNickName}
	}
	if item.UpdatedUser != "" {
		vo.UpdatedBy = &v1.UserBriefVo{ID: item.UpdatedUser, UserName: item.UpdatedUserName, NickName: item.UpdatedUserNickName}
	}
	return vo
}

func toThingDetailVo(item *Thing) *v1.ThingDetailVo {
	return &v1.ThingDetailVo{
		ThingVo:      *toThingVo(item),
		DeviceSecret: item.DeviceSecret,
		Token:        item.Token,
		MQTTPassword: item.MQTTPassword,
		Image:        item.Image,
		Features:     item.Features,
	}
}

func debugValue(value *int) int {
	if value != nil && *value == 1 {
		return 1
	}
	return 0
}

func applyOnline(vo *v1.ThingVo, enable int, score float64, now int64, ttl time.Duration) {
	if score <= 0 {
		vo.OnlineStatus = 0
		vo.LastActiveTime = nil
		return
	}
	last := int64(score)
	vo.LastActiveTime = &last
	if enable == 1 && now-last <= ttl.Milliseconds() {
		vo.OnlineStatus = 1
		return
	}
	vo.OnlineStatus = 0
}

func dedupeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func toAny(values []string) []interface{} {
	out := make([]interface{}, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func isDuplicateIndex(err error, index string) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return strings.Contains(mysqlErr.Message, index)
	}
	return false
}
