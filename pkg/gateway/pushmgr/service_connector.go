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
	if item.RunState != ConnectorRunStateStopped ||
		(s.runtime != nil && s.runtime.ConnectorStatus(id) != ConnectorStatusStopped) {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "runState", MsgID: "validate.push_connector_must_stopped",
		}})
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
