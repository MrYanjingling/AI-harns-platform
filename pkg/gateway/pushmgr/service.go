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
	store     *Store
	source    CollectionSource
	runtime   *RuntimeManager
	factories map[string]ConnectorFactory
}

func NewService(store *Store, source CollectionSource, runtime *RuntimeManager) *Service {
	factories := map[string]ConnectorFactory{
		ConnectorTypeMQTT:  NewMQTTFactory(),
		ConnectorTypeKafka: NewKafkaFactory(),
	}
	if runtime != nil {
		for connectorType, factory := range runtime.factories {
			factories[connectorType] = factory
		}
	}
	return &Service{store: store, source: source, runtime: runtime, factories: factories}
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

// StopAll stops all push groups and connectors while keeping persisted records.
func (s *Service) StopAll(ctx context.Context) error {
	if s == nil {
		return nil
	}
	var errs []error
	groups, err := s.store.ListAllGroups(ctx)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group == nil || group.RunState != ConnectorRunStateRunning {
			continue
		}
		if _, err := s.StopGroup(ctx, group.ID); err != nil && !errors.Is(err, ErrGroupNotFound) {
			errs = append(errs, err)
		}
	}
	connectors, err := s.store.ListAllConnectors(ctx)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	for _, connector := range connectors {
		if connector == nil || connector.RunState != ConnectorRunStateRunning {
			continue
		}
		if _, err := s.StopConnector(ctx, connector.ID); err != nil && !errors.Is(err, ErrConnectorNotFound) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Service) ListConnectors(ctx context.Context, q apis.PageQuery, keyword, connectorType string) (*apis.PageResp[*v1.ConnectorVo], error) {
	q.Normalize()
	items, total, err := s.store.ListConnectors(ctx, q, keyword, connectorType)
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
	connectorType := strings.ToLower(strings.TrimSpace(req.Type))
	if connectorType == "" {
		connectorType = ConnectorTypeMQTT
	}
	pushConfig, err := s.normalizeConnectorConfig(connectorType, req.Name, req.PushConfig)
	if err != nil {
		return nil, &validation.ValidationError{Issues: []validation.FieldIssue{{
			Field: "pushConfig",
			MsgID: "validate.push_connector_config_invalid",
			Args:  []interface{}{err.Error()},
		}}}
	}
	if err := s.validateConnectorRequest(connectorType, req.Name, pushConfig); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	item := &ConnectorRecord{
		ID:          ulid.Make().String(),
		Type:        connectorType,
		Name:        strings.TrimSpace(req.Name),
		PushConfig:  pushConfig,
		RunState:    ConnectorRunStateRunning,
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
	connectorType := strings.ToLower(strings.TrimSpace(req.Type))
	if connectorType == "" {
		connectorType = ConnectorTypeMQTT
	}
	pushConfig, err := s.normalizeConnectorConfig(connectorType, req.Name, req.PushConfig)
	if err != nil {
		return nil, &validation.ValidationError{Issues: []validation.FieldIssue{{
			Field: "pushConfig",
			MsgID: "validate.push_connector_config_invalid",
			Args:  []interface{}{err.Error()},
		}}}
	}
	if err := s.validateConnectorRequest(connectorType, req.Name, pushConfig); err != nil {
		return nil, err
	}
	item.Type = connectorType
	item.Name = strings.TrimSpace(req.Name)
	item.PushConfig = pushConfig
	item.Version++
	item.UpdatedTime = time.Now().UTC()
	if err := s.store.UpdateConnector(ctx, item); err != nil {
		return nil, err
	}
	if s.runtime != nil {
		connect := item.RunState == ConnectorRunStateRunning
		_ = s.runtime.ReplaceConnector(ctx, item, connect)
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
	item.RunState = ConnectorRunStateRunning
	item.Version++
	item.UpdatedTime = time.Now().UTC()
	if err := s.store.UpdateConnector(ctx, item); err != nil {
		return ConnectorStatusError, err
	}
	if s.runtime.ConnectorStatus(id) == ConnectorStatusRunning {
		return ConnectorStatusRunning, ErrConnectorAlreadyRunning
	}
	if err := s.runtime.ReplaceConnector(ctx, item, true); err != nil {
		return ConnectorStatusError, err
	}
	return ConnectorStatusRunning, nil
}

func (s *Service) StopConnector(ctx context.Context, id string) (string, error) {
	item, err := s.store.GetConnector(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrConnectorNotFound
	}
	if err != nil {
		return "", err
	}
	item.RunState = ConnectorRunStateStopped
	item.Version++
	item.UpdatedTime = time.Now().UTC()
	if err := s.store.UpdateConnector(ctx, item); err != nil {
		return ConnectorStatusError, err
	}
	if s.runtime == nil || s.runtime.ConnectorStatus(id) == ConnectorStatusStopped {
		return ConnectorStatusStopped, nil
	}
	if err := s.runtime.StopConnector(ctx, id); err != nil {
		return ConnectorStatusStopped, err
	}
	return ConnectorStatusStopped, nil
}

func (s *Service) RestartConnector(ctx context.Context, id string) (string, error) {
	item, err := s.store.GetConnector(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrConnectorNotFound
	}
	if err != nil {
		return "", err
	}
	if item.RunState != ConnectorRunStateRunning {
		return ConnectorStatusStopped, ErrConnectorAlreadyStopped
	}
	if s.runtime == nil {
		return ConnectorStatusError, &ConnectorStartError{Reason: "push runtime is not initialized"}
	}
	if err := s.runtime.ReplaceConnector(ctx, item, true); err != nil {
		return ConnectorStatusError, err
	}
	return ConnectorStatusRunning, nil
}

func (s *Service) ListGroups(ctx context.Context, q apis.PageQuery, keyword, connectorID string) (*apis.PageResp[*v1.PushGroupVo], error) {
	q.Normalize()
	items, total, err := s.store.ListGroups(ctx, q, keyword, connectorID)
	if err != nil {
		return nil, err
	}
	list := make([]*v1.PushGroupVo, 0, len(items))
	for _, item := range items {
		list = append(list, s.groupVO(item))
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
	return s.groupVO(item), nil
}

func (s *Service) CreateGroup(ctx context.Context, req *v1.CreatePushGroupReq) (*v1.PushGroupVo, error) {
	item, err := s.buildGroup(ctx, req.Name, req.ConnectorID, req.DataTopic, req.CommandTopic, req.PushIntervalMs, req.AllCollections, req.CollectionIDs, req.MaxCollectionsPerPushMessage)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	item.ID = ulid.Make().String()
	item.RunState = ConnectorRunStateRunning
	item.Version = 1
	item.CreatedTime = now
	item.UpdatedTime = now
	if err := s.store.CreateGroup(ctx, item); err != nil {
		return nil, err
	}
	if s.runtime != nil {
		s.runtime.UpsertGroup(item)
	}
	return s.groupVO(item), nil
}

func (s *Service) UpdateGroup(ctx context.Context, id string, req *v1.UpdatePushGroupReq) (*v1.PushGroupVo, error) {
	current, err := s.store.GetGroup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	updated, err := s.buildGroup(ctx, req.Name, req.ConnectorID, req.DataTopic, req.CommandTopic, req.PushIntervalMs, req.AllCollections, req.CollectionIDs, req.MaxCollectionsPerPushMessage)
	if err != nil {
		return nil, err
	}
	current.Name = updated.Name
	current.ConnectorID = updated.ConnectorID
	current.DataTopic = updated.DataTopic
	current.CommandTopic = updated.CommandTopic
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
	return s.groupVO(current), nil
}

func (s *Service) SubscribePayloads(groupID string) (<-chan PushPayloadEvent, func(), error) {
	if s == nil || s.runtime == nil {
		return nil, nil, ErrGroupNotFound
	}
	return s.runtime.SubscribePayloads(groupID)
}

func (s *Service) SubscribeCommands(groupID string) (<-chan PushCommandEvent, func(), error) {
	if s == nil || s.runtime == nil {
		return nil, nil, ErrGroupNotFound
	}
	return s.runtime.SubscribeCommands(groupID)
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

func (s *Service) RemoveCollectionReferences(ctx context.Context, collectionID string) error {
	if s == nil || s.store == nil {
		return nil
	}
	groupIDs, err := s.store.RemoveCollectionReferences(ctx, collectionID)
	if err != nil {
		return err
	}
	if s.runtime == nil {
		return nil
	}
	for _, groupID := range groupIDs {
		group, err := s.store.GetGroup(ctx, groupID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		s.runtime.UpsertGroup(group)
	}
	return nil
}

func (s *Service) StartGroup(ctx context.Context, id string) (string, error) {
	item, err := s.store.GetGroup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ConnectorStatusStopped, ErrGroupNotFound
	}
	if err != nil {
		return ConnectorStatusStopped, err
	}
	if s.runtime == nil {
		return ConnectorStatusStopped, ErrGroupNotFound
	}
	if s.runtime.ConnectorStatus(item.ConnectorID) != ConnectorStatusRunning {
		return ConnectorStatusStopped, ErrGroupConnectorNotRunning
	}
	item.RunState = ConnectorRunStateRunning
	item.Version++
	item.UpdatedTime = time.Now().UTC()
	if err := s.store.UpdateGroup(ctx, item); err != nil {
		return ConnectorStatusError, err
	}
	s.runtime.UpsertGroup(item)
	if err := s.runtime.StartGroup(id); err != nil {
		return ConnectorStatusError, err
	}
	return ConnectorStatusRunning, nil
}

func (s *Service) StopGroup(ctx context.Context, id string) (string, error) {
	item, err := s.store.GetGroup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ConnectorStatusStopped, ErrGroupNotFound
	}
	if err != nil {
		return ConnectorStatusStopped, err
	}
	item.RunState = ConnectorRunStateStopped
	item.Version++
	item.UpdatedTime = time.Now().UTC()
	if err := s.store.UpdateGroup(ctx, item); err != nil {
		return ConnectorStatusError, err
	}
	if s.runtime == nil || !s.runtime.GroupRunning(id) {
		if s.runtime != nil {
			s.runtime.UpsertGroup(item)
		}
		return ConnectorStatusStopped, nil
	}
	if err := s.runtime.StopGroup(id); err != nil {
		return ConnectorStatusStopped, err
	}
	return ConnectorStatusStopped, nil
}

func (s *Service) RestartGroup(id string) (string, error) {
	item, err := s.store.GetGroup(context.Background(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return ConnectorStatusStopped, ErrGroupNotFound
	}
	if err != nil {
		return ConnectorStatusStopped, err
	}
	if item.RunState != ConnectorRunStateRunning {
		return ConnectorStatusStopped, ErrGroupAlreadyStopped
	}
	if s.runtime == nil {
		return ConnectorStatusStopped, ErrGroupNotFound
	}
	if s.runtime.ConnectorStatus(item.ConnectorID) != ConnectorStatusRunning {
		return ConnectorStatusStopped, ErrGroupConnectorNotRunning
	}
	if err := s.runtime.RestartGroup(id); err != nil {
		return ConnectorStatusError, err
	}
	return ConnectorStatusRunning, nil
}

func (s *Service) buildGroup(ctx context.Context, name, connectorID, dataTopic, commandTopic string, intervalMs int64, all bool, collectionIDs []string, maxPerMessage *int) (*PushGroup, error) {
	name = strings.TrimSpace(name)
	connectorID = strings.TrimSpace(connectorID)
	dataTopic = strings.TrimSpace(dataTopic)
	commandTopic = strings.TrimSpace(commandTopic)
	issues := make([]validation.FieldIssue, 0)
	if name == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.push_group_name_required"})
	}
	if dataTopic == "" {
		issues = append(issues, validation.FieldIssue{Field: "dataTopic", MsgID: "validate.push_group_data_topic_required"})
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
		DataTopic:                    dataTopic,
		CommandTopic:                 commandTopic,
		PushIntervalMs:               intervalMs,
		AllCollections:               all,
		CollectionIDs:                collectionIDs,
		MaxCollectionsPerPushMessage: maxCollections,
	}, nil
}

func (s *Service) normalizeConnectorConfig(typ, name string, raw json.RawMessage) (json.RawMessage, error) {
	factory, ok := s.factories[typ]
	if !ok {
		return raw, nil
	}
	if normalizer, ok := factory.(ConnectorConfigNormalizer); ok {
		return normalizer.Normalize(name, raw)
	}
	return raw, nil
}

func (s *Service) validateConnectorRequest(typ, name string, raw json.RawMessage) error {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		typ = ConnectorTypeMQTT
	}
	name = strings.TrimSpace(name)
	issues := make([]validation.FieldIssue, 0)
	if name == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.push_connector_name_required"})
	}
	factory, ok := s.factories[typ]
	if !ok {
		issues = append(issues, validation.FieldIssue{Field: "type", MsgID: "validate.push_connector_type_invalid"})
	} else if err := factory.Validate(raw); err != nil {
		issues = append(issues, validation.FieldIssue{Field: "pushConfig", MsgID: "validate.push_connector_config_invalid", Args: []interface{}{err.Error()}})
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
	lastError := ""
	if s != nil && s.runtime != nil {
		status = s.runtime.ConnectorStatus(item.ID)
		lastError = s.runtime.ConnectorLastError(item.ID)
	}
	return &v1.ConnectorVo{
		ID:          item.ID,
		Type:        item.Type,
		Name:        item.Name,
		PushConfig:  item.PushConfig,
		RunState:    item.RunState,
		GroupCount:  item.GroupCount,
		Status:      status,
		LastError:   lastError,
		Version:     item.Version,
		CreatedTime: item.CreatedTime,
		UpdatedTime: item.UpdatedTime,
	}
}

func (s *Service) groupVO(item *PushGroup) *v1.PushGroupVo {
	if item == nil {
		return nil
	}
	status := ConnectorStatusStopped
	lastError := ""
	var lastPushTime *time.Time
	if s != nil && s.runtime != nil {
		status = s.runtime.GroupStatus(item.ID)
		lastError = s.runtime.GroupLastError(item.ID)
		if value := s.runtime.GroupLastPushTime(item.ID); !value.IsZero() {
			lastPushTime = &value
		}
	}
	return &v1.PushGroupVo{
		ID:                           item.ID,
		Name:                         item.Name,
		ConnectorID:                  item.ConnectorID,
		DataTopic:                    item.DataTopic,
		CommandTopic:                 item.CommandTopic,
		RunState:                     item.RunState,
		Status:                       status,
		LastError:                    lastError,
		LastPushTime:                 lastPushTime,
		PushIntervalMs:               item.PushIntervalMs,
		AllCollections:               item.AllCollections,
		CollectionIDs:                item.CollectionIDs,
		MaxCollectionsPerPushMessage: item.MaxCollectionsPerPushMessage,
		Version:                      item.Version,
		CreatedTime:                  item.CreatedTime,
		UpdatedTime:                  item.UpdatedTime,
	}
}
