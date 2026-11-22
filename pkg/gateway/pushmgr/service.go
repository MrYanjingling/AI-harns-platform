package pushmgr

import (
	"context"
	"errors"
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
