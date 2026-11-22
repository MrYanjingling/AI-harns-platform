package pushmgr

import (
	"context"
	"fmt"
	"sync"
)

type RuntimeManager struct {
	mu         sync.Mutex
	source     CollectionSource
	gatewayID  string
	readySend  ReadySendStore
	factories  map[string]ConnectorFactory
	connectors map[string]*connectorRuntime
	groups     map[string]*groupRuntime
}

func NewRuntimeManager(source CollectionSource, gatewayID string, factories ...ConnectorFactory) (*RuntimeManager, error) {
	m := &RuntimeManager{
		source:     source,
		gatewayID:  gatewayID,
		factories:  make(map[string]ConnectorFactory, len(factories)),
		connectors: make(map[string]*connectorRuntime),
		groups:     make(map[string]*groupRuntime),
	}
	for _, factory := range factories {
		if factory == nil {
			return nil, fmt.Errorf("push connector factory is nil")
		}
		if _, exists := m.factories[factory.Type()]; exists {
			return nil, fmt.Errorf("duplicate push connector factory %s", factory.Type())
		}
		m.factories[factory.Type()] = factory
	}
	return m, nil
}

// SetReadySendStore injects the ready_send persistence store.
func (m *RuntimeManager) SetReadySendStore(store ReadySendStore) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.readySend = store
	m.mu.Unlock()
}

func (m *RuntimeManager) Start(ctx context.Context, connectors []*ConnectorRecord, groups []*PushGroup) error {
	for _, connector := range connectors {
		if connector.RunState == ConnectorRunStateRunning {
			m.UpsertConnector(ctx, connector)
		} else {
			_ = m.ReplaceConnector(ctx, connector, false)
		}
	}
	for _, group := range groups {
		m.UpsertGroup(group)
	}
	return nil
}

func (m *RuntimeManager) Close(ctx context.Context) {
	m.mu.Lock()
	groups := make([]*groupRuntime, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group)
	}
	connectors := make([]*connectorRuntime, 0, len(m.connectors))
	for _, connector := range m.connectors {
		connectors = append(connectors, connector)
	}
	m.groups = make(map[string]*groupRuntime)
	m.connectors = make(map[string]*connectorRuntime)
	m.mu.Unlock()
	for _, group := range groups {
		stopGroupRuntime(group)
		group.hub.Close()
		group.commands.Close()
	}
	for _, connector := range connectors {
		_ = connector.connector.Close(ctx)
	}
}
