package pushmgr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/gateway/pushmgr/v1"
	"harns-platform/pkg/gateway/validation"
)

type Service struct {
	store   *Store
	source  CollectionSource
	runtime *RuntimeManager
}

func NewService(store *Store, source CollectionSource, runtime *RuntimeManager) *Service {
	return &Service{store: store, source: source, runtime: runtime}
}

func (s *Service) StartRuntime(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	connectors, err := s.store.ListAllConnectors(ctx)
	if err != nil {
		return err
	}
	groups, err := s.store.ListAllGroups(ctx)
	if err != nil {
		return err
	}
	return s.runtime.Start(ctx, connectors, groups)
}

func (s *Service) Close(ctx context.Context) {
	if s != nil && s.runtime != nil {
		s.runtime.Close(ctx)
	}
}

func (s *Service) ListConnectors(ctx context.Context, q apis.PageQuery, keyword string) (*apis.PageResp[*v1.ConnectorVo], error) {
	q.Normalize()
	items, total, err := s.store.ListConnectors(ctx, q, keyword)
	if err != nil {
		return nil, err
	}
	list := make([]*v1.ConnectorVo, 0, len(items))
	for _, item := range items {
		list = append(list, s.connectorVO(item))
	}
	return apis.NewPageResp(list, total, q), nil
}

func (s *Service) GetConnector(ctx context.Context, id string) (*v1.ConnectorVo, error) {
	item, err := s.store.GetConnector(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConnectorNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.connectorVO(item), nil
}

func (s *Service) CreateConnector(ctx context.Context, req *v1.CreateConnectorReq) (*v1.ConnectorVo, error) {
	if err := validateConnectorRequest(req.Type, req.Name, req.PushConfig); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	connectorType := strings.ToLower(strings.TrimSpace(req.Type))
	if connectorType == "" {
		connectorType = ConnectorTypeMQTT
	}
	item := &ConnectorRecord{
		ID:          ulid.Make().String(),
		Type:        connectorType,
		Name:        strings.TrimSpace(req.Name),
		PushConfig:  req.PushConfig,
		Version:     1,
		CreatedTime: now,
		UpdatedTime: now,
	}
	if err := s.store.CreateConnector(ctx, item); err != nil {
		return nil, err
	}
	if s.runtime != nil {
		s.runtime.UpsertConnector(ctx, item)
	}
	return s.connectorVO(item), nil
}

func (s *Service) UpdateConnector(ctx context.Context, id string, req *v1.UpdateConnectorReq) (*v1.ConnectorVo, error) {
	item, err := s.store.GetConnector(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConnectorNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := validateConnectorRequest(req.Type, req.Name, req.PushConfig); err != nil {
		return nil, err
	}
	item.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if item.Type == "" {
		item.Type = ConnectorTypeMQTT
	}
	item.Name = strings.TrimSpace(req.Name)
	item.PushConfig = req.PushConfig
	item.Version++
	item.UpdatedTime = time.Now().UTC()
	wasRunning := s.runtime != nil && s.runtime.ConnectorStatus(id) == ConnectorStatusRunning
	if err := s.store.UpdateConnector(ctx, item); err != nil {
		return nil, err
	}
	if s.runtime != nil {
		if wasRunning {
			_ = s.runtime.ReplaceConnector(ctx, item, true)
		} else {
			_ = s.runtime.ReplaceConnector(ctx, item, false)
		}
	}
	return s.connectorVO(item), nil
}

func (s *Service) DeleteConnector(ctx context.Context, id string, cascade bool) error {
	groupIDs, err := s.store.DeleteConnector(ctx, id, cascade)
	if err != nil {
		return err
	}
	if s.runtime != nil {
		s.runtime.RemoveConnector(ctx, id)
		for _, groupID := range groupIDs {
			s.runtime.RemoveGroup(groupID)
		}
	}
	return nil
}

func (s *Service) StartConnector(ctx context.Context, id string) (string, error) {
	item, err := s.store.GetConnector(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrConnectorNotFound
	}
	if err != nil {
		return "", err
	}
	if s.runtime == nil {
		return "", &ConnectorStartError{Reason: "push runtime is not initialized"}
	}
	if s.runtime.ConnectorStatus(id) == ConnectorStatusRunning {
		return ConnectorStatusRunning, ErrConnectorAlreadyRunning
	}
	if err := s.runtime.ReplaceConnector(ctx, item, true); err != nil {
		return ConnectorStatusStopped, err
	}
	return ConnectorStatusRunning, nil
}

func (s *Service) StopConnector(ctx context.Context, id string) (string, error) {
	if _, err := s.store.GetConnector(ctx, id); errors.Is(err, sql.ErrNoRows) {
		return "", ErrConnectorNotFound
	} else if err != nil {
		return "", err
	}
	if s.runtime == nil {
		return ConnectorStatusStopped, nil
	}
	if s.runtime.ConnectorStatus(id) == ConnectorStatusStopped {
		return ConnectorStatusStopped, ErrConnectorAlreadyStopped
	}
	if err := s.runtime.StopConnector(ctx, id); err != nil {
		return ConnectorStatusStopped, err
	}
	return ConnectorStatusStopped, nil
}

func (s *Service) ListGroups(ctx context.Context, q apis.PageQuery, keyword string) (*apis.PageResp[*v1.PushGroupVo], error) {
	q.Normalize()
	items, total, err := s.store.ListGroups(ctx, q, keyword)
	if err != nil {
		return nil, err
	}
	list := make([]*v1.PushGroupVo, 0, len(items))
	for _, item := range items {
		list = append(list, groupVO(item))
	}
	return apis.NewPageResp(list, total, q), nil
}

func (s *Service) GetGroup(ctx context.Context, id string) (*v1.PushGroupVo, error) {
	item, err := s.store.GetGroup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	return groupVO(item), nil
}

func (s *Service) CreateGroup(ctx context.Context, req *v1.CreatePushGroupReq) (*v1.PushGroupVo, error) {
	item, err := s.buildGroup(ctx, req.Name, req.ConnectorID, req.PushIntervalMs, req.AllCollections, req.CollectionIDs, req.MaxCollectionsPerPushMessage)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	item.ID = ulid.Make().String()
	item.Version = 1
	item.CreatedTime = now
	item.UpdatedTime = now
	if err := s.store.CreateGroup(ctx, item); err != nil {
		return nil, err
	}
	if s.runtime != nil {
		s.runtime.UpsertGroup(item)
	}
	return groupVO(item), nil
}

func (s *Service) UpdateGroup(ctx context.Context, id string, req *v1.UpdatePushGroupReq) (*v1.PushGroupVo, error) {
	current, err := s.store.GetGroup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	updated, err := s.buildGroup(ctx, req.Name, req.ConnectorID, req.PushIntervalMs, req.AllCollections, req.CollectionIDs, req.MaxCollectionsPerPushMessage)
	if err != nil {
		return nil, err
	}
	current.Name = updated.Name
	current.ConnectorID = updated.ConnectorID
	current.PushIntervalMs = updated.PushIntervalMs
	current.AllCollections = updated.AllCollections
	current.CollectionIDs = updated.CollectionIDs
	current.MaxCollectionsPerPushMessage = updated.MaxCollectionsPerPushMessage
	current.Version++
	current.UpdatedTime = time.Now().UTC()
	if err := s.store.UpdateGroup(ctx, current); err != nil {
		return nil, err
	}
	if s.runtime != nil {
		s.runtime.UpsertGroup(current)
	}
	return groupVO(current), nil
}

func (s *Service) DeleteGroup(ctx context.Context, id string) error {
	if err := s.store.DeleteGroup(ctx, id); err != nil {
		return err
	}
	if s.runtime != nil {
		s.runtime.RemoveGroup(id)
	}
	return nil
}

func (s *Service) buildGroup(ctx context.Context, name, connectorID string, intervalMs int64, all bool, collectionIDs []string, maxPerMessage *int) (*PushGroup, error) {
	name = strings.TrimSpace(name)
	connectorID = strings.TrimSpace(connectorID)
	issues := make([]validation.FieldIssue, 0)
	if name == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.push_group_name_required"})
	}
	if connectorID == "" {
		issues = append(issues, validation.FieldIssue{Field: "connectorId", MsgID: "validate.push_group_connector_required"})
	} else if _, err := s.store.GetConnector(ctx, connectorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			issues = append(issues, validation.FieldIssue{Field: "connectorId", MsgID: "validate.push_group_connector_not_found"})
		} else {
			return nil, err
		}
	}
	if intervalMs == 0 {
		intervalMs = DefaultPushIntervalMs
	}
	if intervalMs < 0 {
		issues = append(issues, validation.FieldIssue{Field: "pushIntervalMs", MsgID: "validate.push_group_interval_invalid"})
	}
	maxCollections := DefaultMaxCollectionsPerMessage
	if maxPerMessage != nil {
		maxCollections = *maxPerMessage
		if maxCollections < 0 {
			issues = append(issues, validation.FieldIssue{Field: "maxCollectionsPerPushMessage", MsgID: "validate.push_group_max_collections_invalid"})
		}
	}
	if all && len(collectionIDs) > 0 {
		issues = append(issues, validation.FieldIssue{Field: "collectionIds", MsgID: "validate.push_group_collection_mode_invalid"})
	}
	if !all && len(collectionIDs) == 0 {
		issues = append(issues, validation.FieldIssue{Field: "collectionIds", MsgID: "validate.push_group_collection_required"})
	}
	if all {
		collectionIDs = []string{}
	}
	if !all {
		seen := make(map[string]struct{}, len(collectionIDs))
		cleaned := make([]string, 0, len(collectionIDs))
		for _, id := range collectionIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			if exists, err := s.source.Exists(ctx, id); err != nil {
				return nil, err
			} else if !exists {
				issues = append(issues, validation.FieldIssue{Field: "collectionIds", MsgID: "validate.push_group_collection_not_found", Args: []interface{}{id}})
				continue
			}
			cleaned = append(cleaned, id)
		}
		collectionIDs = cleaned
	}
	if len(issues) > 0 {
		return nil, &validation.ValidationError{Issues: issues}
	}
	return &PushGroup{
		Name:                         name,
		ConnectorID:                  connectorID,
		PushIntervalMs:               intervalMs,
		AllCollections:               all,
		CollectionIDs:                collectionIDs,
		MaxCollectionsPerPushMessage: maxCollections,
	}, nil
}

func validateConnectorRequest(typ, name string, raw json.RawMessage) error {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		typ = ConnectorTypeMQTT
	}
	name = strings.TrimSpace(name)
	issues := make([]validation.FieldIssue, 0)
	if name == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.push_connector_name_required"})
	}
	switch typ {
	case ConnectorTypeMQTT:
		var config MQTTConfig
		if len(raw) == 0 || json.Unmarshal(raw, &config) != nil {
			issues = append(issues, validation.FieldIssue{Field: "pushConfig", MsgID: "validate.push_connector_config_invalid"})
		} else {
			config.Normalize("")
			if err := config.Validate(); err != nil {
				issues = append(issues, validation.FieldIssue{Field: "pushConfig", MsgID: "validate.push_connector_config_invalid", Args: []interface{}{err.Error()}})
			}
		}
	default:
		issues = append(issues, validation.FieldIssue{Field: "type", MsgID: "validate.push_connector_type_invalid"})
	}
	if len(issues) > 0 {
		return &validation.ValidationError{Issues: issues}
	}
	return nil
}

func (s *Service) connectorVO(item *ConnectorRecord) *v1.ConnectorVo {
	if item == nil {
		return nil
	}
	status := ConnectorStatusStopped
	if s != nil && s.runtime != nil {
		status = s.runtime.ConnectorStatus(item.ID)
	}
	return &v1.ConnectorVo{
		ID:          item.ID,
		Type:        item.Type,
		Name:        item.Name,
		PushConfig:  item.PushConfig,
		Status:      status,
		Version:     item.Version,
		CreatedTime: item.CreatedTime,
		UpdatedTime: item.UpdatedTime,
	}
}

func groupVO(item *PushGroup) *v1.PushGroupVo {
	if item == nil {
		return nil
	}
	return &v1.PushGroupVo{
		ID:                           item.ID,
		Name:                         item.Name,
		ConnectorID:                  item.ConnectorID,
		PushIntervalMs:               item.PushIntervalMs,
		AllCollections:               item.AllCollections,
		CollectionIDs:                item.CollectionIDs,
		MaxCollectionsPerPushMessage: item.MaxCollectionsPerPushMessage,
		Version:                      item.Version,
		CreatedTime:                  item.CreatedTime,
		UpdatedTime:                  item.UpdatedTime,
	}
}
