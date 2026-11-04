package thingmodel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/logging"
	"harns-platform/pkg/platform/principal"
	"harns-platform/pkg/platform/thingmodel/tsl"
	"harns-platform/pkg/platform/thingmodel/v1"
)

const (
	propertiesCachePrefix = "iot:thingmodel:properties:"
	propertiesCacheTTL    = 24 * time.Hour
)

// Service is the business layer of the thing model module.
type Service struct {
	store *Store
	redis redis.UniversalClient
}

func NewService(store *Store, rdb redis.UniversalClient) *Service {
	return &Service{store: store, redis: rdb}
}

// Create creates a thing model with optional initial properties.
func (s *Service) Create(ctx context.Context, req *v1.CreateThingModelReq) (*v1.ThingModelDetailVo, error) {
	defer logging.Trace("thingmodel.Create", "name", req.Name, "thingModelKey", req.ThingModelKey)()

	doc := &tsl.ThingModel{
		ThingModelKey: strings.TrimSpace(req.ThingModelKey),
		Name:          strings.TrimSpace(req.Name),
		Description:   strings.TrimSpace(req.Description),
		Properties:    req.Properties,
		Features:      req.Features,
	}
	doc.Normalize()
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	if exists, err := s.store.ExistsThingModelName(ctx, doc.Name, ""); err != nil {
		return nil, response.ErrInternal(err)
	} else if exists {
		return nil, response.ErrThingModelNameExists(doc.Name)
	}

	key, err := s.resolveKey(ctx, doc.ThingModelKey)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	operator := principal.UserID(ctx)
	model := &ThingModel{
		Meta: apis.Meta{
			ID:          NewID(),
			Version:     0,
			CreatedUser: operator,
			UpdatedUser: operator,
			CreatedTime: now,
			UpdatedTime: now,
		},
		ThingModelKey: key,
		Name:          doc.Name,
		Description:   doc.Description,
		Features:      doc.Features,
	}
	properties := buildProperties(model.ID, doc.Properties, operator, now)
	if err := s.store.CreateThingModel(ctx, model, properties); err != nil {
		switch {
		case isDuplicateIndex(err, "uk_thing_models_name"):
			return nil, response.ErrThingModelNameExists(doc.Name)
		case isDuplicateIndex(err, "uk_thing_models_key"):
			return nil, response.ErrThingModelKeyExists(key)
		default:
			return nil, response.ErrInternal(err)
		}
	}
	s.invalidatePropertiesCache(ctx, model.ThingModelKey)
	return s.Get(ctx, model.ID)
}

// List returns a page of thing models.
func (s *Service) List(ctx context.Context, q v1.ThingModelQuery) (*apis.PageResp[*v1.ThingModelVo], error) {
	defer logging.Trace("thingmodel.List", "keyword", q.Keyword, "page", q.Page, "pageSize", q.PageSize)()

	q.Normalize()
	models, total, err := s.store.ListThingModels(ctx, q.PageQuery, q.Keyword)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	list := make([]*v1.ThingModelVo, 0, len(models))
	for _, model := range models {
		list = append(list, toThingModelVo(model))
	}
	return apis.NewPageResp(list, total, q.PageQuery), nil
}

// Get returns a thing model with its properties.
func (s *Service) Get(ctx context.Context, id string) (*v1.ThingModelDetailVo, error) {
	defer logging.Trace("thingmodel.Get", "id", id)()

	model, err := s.getEntity(ctx, id)
	if err != nil {
		return nil, err
	}
	return toThingModelDetailVo(model), nil
}

// GetPropertiesByKey returns the properties of a model indexed by identifier.
// Redis is used as a transparent cache; a miss falls back to MySQL.
func (s *Service) GetPropertiesByKey(ctx context.Context, modelKey string) (map[string]tsl.Property, error) {
	defer logging.Trace("thingmodel.GetPropertiesByKey", "thingModelKey", modelKey)()

	modelKey = strings.TrimSpace(modelKey)
	if modelKey == "" {
		return nil, fmt.Errorf("thing model key is required")
	}

	cacheKey := propertiesCachePrefix + modelKey
	if s.redis != nil {
		if data, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var properties map[string]tsl.Property
			if err := json.Unmarshal(data, &properties); err == nil {
				return properties, nil
			} else {
				logging.ErrorS(err, "Failed to decode cached thing model properties", "thingModelKey", modelKey)
			}
		} else if !errors.Is(err, redis.Nil) {
			logging.ErrorS(err, "Failed to read cached thing model properties", "thingModelKey", modelKey)
		}
	}

	entities, err := s.store.ListPropertiesByModelKey(ctx, modelKey)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		exists, err := s.store.ExistsThingModelKey(ctx, modelKey, "")
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("thing model %s not found", modelKey)
		}
	}
	properties := make(map[string]tsl.Property, len(entities))
	for i := range entities {
		properties[entities[i].Identifier] = entities[i].Property
	}

	if s.redis != nil {
		if data, err := json.Marshal(properties); err != nil {
			logging.ErrorS(err, "Failed to encode thing model properties", "thingModelKey", modelKey)
		} else if err := s.redis.Set(ctx, cacheKey, data, propertiesCacheTTL).Err(); err != nil {
			logging.ErrorS(err, "Failed to cache thing model properties", "thingModelKey", modelKey)
		}
	}
	return properties, nil
}

func (s *Service) invalidatePropertiesCache(ctx context.Context, modelKey string) {
	if s.redis == nil || modelKey == "" {
		return
	}
	if err := s.redis.Del(ctx, propertiesCachePrefix+modelKey).Err(); err != nil {
		logging.ErrorS(err, "Failed to invalidate thing model properties cache", "thingModelKey", modelKey)
	}
}

// Update changes the mutable header fields of a thing model.
func (s *Service) Update(ctx context.Context, id string, req *v1.UpdateThingModelReq) (*v1.ThingModelVo, error) {
	defer logging.Trace("thingmodel.Update", "id", id)()

	model, err := s.getEntity(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if issues := tsl.ValidateName(name); len(issues) > 0 {
			return nil, tsl.NewValidationError(issues)
		}
		if name != model.Name {
			if exists, err := s.store.ExistsThingModelName(ctx, name, id); err != nil {
				return nil, response.ErrInternal(err)
			} else if exists {
				return nil, response.ErrThingModelNameExists(name)
			}
			model.Name = name
		}
	}
	if req.Description != nil {
		model.Description = strings.TrimSpace(*req.Description)
	}
	model.UpdatedUser = principal.UserID(ctx)
	model.UpdatedTime = time.Now()

	n, err := s.store.UpdateThingModel(ctx, model)
	if err != nil {
		if isDuplicateIndex(err, "uk_thing_models_name") {
			return nil, response.ErrThingModelNameExists(model.Name)
		}
		return nil, response.ErrInternal(err)
	}
	if n == 0 {
		return nil, response.ErrThingModelNotFound(id)
	}
	model.Version++
	return toThingModelVo(model), nil
}

// Delete removes a thing model and all of its properties. cascade is reserved
// for cascading to related things and is not applied yet.
func (s *Service) Delete(ctx context.Context, id string, cascade bool) error {
	defer logging.Trace("thingmodel.Delete", "id", id, "cascade", cascade)()

	modelKey, err := s.store.GetThingModelKey(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return response.ErrThingModelNotFound(id)
	}
	if err != nil {
		return response.ErrInternal(err)
	}
	n, err := s.store.DeleteThingModel(ctx, id)
	if err != nil {
		return response.ErrInternal(err)
	}
	if n == 0 {
		return response.ErrThingModelNotFound(id)
	}
	s.invalidatePropertiesCache(ctx, modelKey)
	return nil
}

// AddProperties appends properties atomically.
func (s *Service) AddProperties(ctx context.Context, id string, properties []tsl.Property) ([]v1.PropertyVo, error) {
	defer logging.Trace("thingmodel.AddProperties", "id", id, "count", len(properties))()

	model, err := s.getEntity(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(properties) == 0 {
		return []v1.PropertyVo{}, nil
	}

	doc := &tsl.ThingModel{Name: model.Name, Properties: properties}
	doc.Normalize()
	if verr := doc.Validate(); verr != nil {
		return nil, verr
	}

	existing, err := s.store.ListProperties(ctx, id)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	identifierSet := make(map[string]struct{}, len(existing))
	nameSet := make(map[string]struct{}, len(existing))
	for _, p := range existing {
		identifierSet[strings.ToLower(p.Identifier)] = struct{}{}
		nameSet[strings.ToLower(p.Name)] = struct{}{}
	}

	issues := make([]tsl.FieldIssue, 0)
	for i, p := range doc.Properties {
		if _, ok := identifierSet[strings.ToLower(p.Identifier)]; ok {
			issues = append(issues, tsl.FieldIssue{
				Field: fmt.Sprintf("properties[%d].identifier", i),
				MsgID: "validate.thing_model_duplicate_identifier",
				Args:  []interface{}{p.Identifier},
			})
		}
		if _, ok := nameSet[strings.ToLower(p.Name)]; ok {
			issues = append(issues, tsl.FieldIssue{
				Field: fmt.Sprintf("properties[%d].name", i),
				MsgID: "validate.thing_model_duplicate_name",
				Args:  []interface{}{p.Name},
			})
		}
	}
	if verr := tsl.NewValidationError(issues); verr != nil {
		return nil, verr
	}

	now := time.Now()
	operator := principal.UserID(ctx)
	entities := buildProperties(model.ID, doc.Properties, operator, now)
	if err := s.store.AppendProperties(ctx, model.ID, entities, operator, now); err != nil {
		return nil, response.ErrInternal(err)
	}
	s.invalidatePropertiesCache(ctx, model.ThingModelKey)
	return toPropertyVos(entities), nil
}

// UpdateProperty replaces the mutable fields of one property.
func (s *Service) UpdateProperty(ctx context.Context, id, identifier string, property tsl.Property) (*v1.PropertyVo, error) {
	defer logging.Trace("thingmodel.UpdateProperty", "id", id, "identifier", identifier)()

	existing, err := s.store.GetProperty(ctx, id, identifier)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrThingModelPropertyNotFound(identifier)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	modelKey, err := s.store.GetThingModelKey(ctx, existing.ThingModelID)
	if err != nil {
		return nil, response.ErrInternal(err)
	}

	if property.Identifier != "" && property.Identifier != identifier {
		return nil, tsl.NewValidationError([]tsl.FieldIssue{{
			Field: "identifier",
			MsgID: "validate.thing_model_identifier_immutable",
		}})
	}
	property.Identifier = identifier
	property.Normalize()
	if issues := property.Validate("properties[0]"); len(issues) > 0 {
		return nil, tsl.NewValidationError(issues)
	}

	properties, err := s.store.ListProperties(ctx, id)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	for _, p := range properties {
		if p.Identifier == identifier {
			continue
		}
		if strings.EqualFold(p.Name, property.Name) {
			return nil, tsl.NewValidationError([]tsl.FieldIssue{{
				Field: "properties[0].name",
				MsgID: "validate.thing_model_duplicate_name",
				Args:  []interface{}{property.Name},
			}})
		}
	}

	existing.Property = property
	now := time.Now()
	operator := principal.UserID(ctx)
	existing.UpdatedUser = operator
	existing.UpdatedTime = now

	n, err := s.store.UpdateProperty(ctx, existing, operator, now)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if n == 0 {
		return nil, response.ErrThingModelPropertyNotFound(identifier)
	}
	s.invalidatePropertiesCache(ctx, modelKey)
	existing.Version++
	vo := toPropertyVo(existing)
	return &vo, nil
}

// DeleteProperties removes properties idempotently.
func (s *Service) DeleteProperties(ctx context.Context, id string, identifiers []string) (int64, error) {
	defer logging.Trace("thingmodel.DeleteProperties", "id", id, "count", len(identifiers))()

	model, err := s.getEntity(ctx, id)
	if err != nil {
		return 0, err
	}
	cleaned := make([]string, 0, len(identifiers))
	for _, identifier := range identifiers {
		if v := strings.TrimSpace(identifier); v != "" {
			cleaned = append(cleaned, v)
		}
	}
	n, err := s.store.DeleteProperties(ctx, id, cleaned, principal.UserID(ctx), time.Now())
	if err != nil {
		return 0, response.ErrInternal(err)
	}
	s.invalidatePropertiesCache(ctx, model.ThingModelKey)
	return n, nil
}

// Export serializes a thing model as JSON.
func (s *Service) Export(ctx context.Context, id string) ([]byte, error) {
	defer logging.Trace("thingmodel.Export", "id", id)()

	model, err := s.getEntity(ctx, id)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(toTSLDocument(model), "", "  ")
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return data, nil
}

// Import replaces the imported thing model fields from a JSON document.
func (s *Service) Import(ctx context.Context, id string, content []byte) (*v1.ThingModelDetailVo, error) {
	defer logging.Trace("thingmodel.Import", "id", id, "size", len(content))()

	model, err := s.getEntity(ctx, id)
	if err != nil {
		return nil, err
	}

	var req v1.ImportThingModelReq
	if err := json.Unmarshal(content, &req); err != nil {
		return nil, response.ErrThingModelImportInvalid(err.Error())
	}

	issues := make([]tsl.FieldIssue, 0, 2)
	key := strings.TrimSpace(req.ThingModelKey)
	if key != "" && key != model.ThingModelKey {
		issues = append(issues, tsl.FieldIssue{
			Field: "thingModelKey",
			MsgID: "validate.thing_model_key_immutable",
			Args:  []interface{}{key},
		})
	}
	name := strings.TrimSpace(req.Name)
	if name != "" {
		issues = append(issues, tsl.ValidateName(name)...)
	}
	if req.Properties == nil {
		issues = append(issues, tsl.FieldIssue{Field: "properties", MsgID: "validate.thing_model_properties_required"})
	}
	if verr := tsl.NewValidationError(issues); verr != nil {
		return nil, verr
	}

	properties := make([]tsl.Property, 0)
	if req.Properties != nil {
		properties = *req.Properties
	}
	features := model.Features
	if req.Features != nil {
		features = *req.Features
	}
	effectiveName := model.Name
	if name != "" {
		effectiveName = name
	}
	doc := &tsl.ThingModel{Name: effectiveName, Properties: properties, Features: features}
	doc.Normalize()
	if verr := doc.Validate(); verr != nil {
		return nil, verr
	}

	if name != "" && name != model.Name {
		exists, err := s.store.ExistsThingModelName(ctx, name, id)
		if err != nil {
			return nil, response.ErrInternal(err)
		}
		if exists {
			return nil, response.ErrThingModelNameExists(name)
		}
		model.Name = name
	}
	if req.Description != nil {
		model.Description = strings.TrimSpace(*req.Description)
	}
	model.Features = doc.Features

	now := time.Now()
	operator := principal.UserID(ctx)
	model.UpdatedUser = operator
	model.UpdatedTime = now
	entities := buildProperties(model.ID, doc.Properties, operator, now)
	if err := s.store.ReplaceThingModel(ctx, model, entities); err != nil {
		if isDuplicateIndex(err, "uk_thing_models_name") {
			return nil, response.ErrThingModelNameExists(model.Name)
		}
		return nil, response.ErrInternal(err)
	}
	s.invalidatePropertiesCache(ctx, model.ThingModelKey)
	return s.Get(ctx, model.ID)
}

func (s *Service) resolveKey(ctx context.Context, key string) (string, error) {
	if key != "" {
		exists, err := s.store.ExistsThingModelKey(ctx, key, "")
		if err != nil {
			return "", response.ErrInternal(err)
		}
		if exists {
			return "", response.ErrThingModelKeyExists(key)
		}
		return key, nil
	}
	for i := 0; i < 3; i++ {
		candidate := NewThingModelKey()
		exists, err := s.store.ExistsThingModelKey(ctx, candidate, "")
		if err != nil {
			return "", response.ErrInternal(err)
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", response.ErrInternal(errors.New("failed to generate a unique thing model key"))
}

func (s *Service) getEntity(ctx context.Context, id string) (*ThingModel, error) {
	model, err := s.store.GetThingModel(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrThingModelNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return model, nil
}

func buildProperties(modelID string, properties []tsl.Property, operator string, now time.Time) []*Property {
	out := make([]*Property, 0, len(properties))
	for _, property := range properties {
		out = append(out, &Property{
			Meta: apis.Meta{
				ID:          NewID(),
				Version:     0,
				CreatedUser: operator,
				UpdatedUser: operator,
				CreatedTime: now,
				UpdatedTime: now,
			},
			ThingModelID: modelID,
			Property:     property,
		})
	}
	return out
}

func toThingModelVo(model *ThingModel) *v1.ThingModelVo {
	return &v1.ThingModelVo{
		ID:            model.ID,
		Version:       model.Version,
		ThingModelKey: model.ThingModelKey,
		Name:          model.Name,
		Description:   model.Description,
		PropertyCount: model.PropertyCount,
		CreatedUser:   model.CreatedUser,
		UpdatedUser:   model.UpdatedUser,
		CreatedTime:   model.CreatedTime,
		UpdatedTime:   model.UpdatedTime,
	}
}

func toThingModelDetailVo(model *ThingModel) *v1.ThingModelDetailVo {
	properties := make([]v1.PropertyVo, 0, len(model.Properties))
	for i := range model.Properties {
		properties = append(properties, toPropertyVo(&model.Properties[i]))
	}
	return &v1.ThingModelDetailVo{
		ThingModelVo: *toThingModelVo(model),
		Properties:   properties,
		Features:     model.Features,
	}
}

func toPropertyVo(property *Property) v1.PropertyVo {
	return v1.PropertyVo{
		ID:          property.ID,
		Version:     property.Version,
		Property:    property.Property,
		CreatedUser: property.CreatedUser,
		UpdatedUser: property.UpdatedUser,
		CreatedTime: property.CreatedTime,
		UpdatedTime: property.UpdatedTime,
	}
}

func toPropertyVos(properties []*Property) []v1.PropertyVo {
	out := make([]v1.PropertyVo, 0, len(properties))
	for _, property := range properties {
		out = append(out, toPropertyVo(property))
	}
	return out
}

func toTSLDocument(model *ThingModel) *tsl.ThingModel {
	properties := make([]tsl.Property, 0, len(model.Properties))
	for i := range model.Properties {
		properties = append(properties, model.Properties[i].Property)
	}
	return &tsl.ThingModel{
		ThingModelKey: model.ThingModelKey,
		Name:          model.Name,
		Description:   model.Description,
		Properties:    properties,
		Features:      model.Features,
	}
}

func isDuplicateIndex(err error, index string) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return strings.Contains(mysqlErr.Message, index)
	}
	return false
}
