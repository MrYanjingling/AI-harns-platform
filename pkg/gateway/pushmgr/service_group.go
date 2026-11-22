package pushmgr

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/gateway/pushmgr/v1"
	"harns-platform/pkg/gateway/validation"
)

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
	if current.RunState != ConnectorRunStateStopped || (s.runtime != nil && s.runtime.GroupRunning(id)) {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "runState", MsgID: "validate.push_group_must_stopped",
		}})
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
