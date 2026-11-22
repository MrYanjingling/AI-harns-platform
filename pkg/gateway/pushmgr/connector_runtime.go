package pushmgr

import (
	"context"
	"fmt"
)

func (m *RuntimeManager) UpsertConnector(ctx context.Context, record *ConnectorRecord) {
	_ = m.ReplaceConnector(ctx, record, true)
}

func (m *RuntimeManager) ReplaceConnector(ctx context.Context, record *ConnectorRecord, connect bool) error {
	if m == nil || record == nil {
		return fmt.Errorf("push connector is nil")
	}
	factory, ok := m.factories[record.Type]
	if !ok {
		return &ConnectorStartError{Reason: fmt.Sprintf("unsupported connector type %q", record.Type)}
	}
	connector, err := factory.New(record.ID, record.PushConfig)
	if err != nil {
		return &ConnectorStartError{Reason: err.Error()}
	}
	runtime := &connectorRuntime{record: record, connector: connector, status: ConnectorStatusStopped}
	if handler, ok := connector.(ConnectorErrorHandler); ok {
		handler.SetErrorHandler(func(err error) {
			m.handleConnectorError(runtime, err)
		})
	}
	if handler, ok := connector.(ConnectorConnectedHandler); ok {
		handler.SetConnectedHandler(func() {
			m.handleConnectorConnected(runtime)
		})
	}
	m.mu.Lock()
	old := m.connectors[record.ID]
	m.connectors[record.ID] = runtime
	m.mu.Unlock()
	if old != nil {
		_ = old.connector.Close(ctx)
	}
	if !connect {
		return nil
	}
	return m.connectRuntime(ctx, runtime)
}

func (m *RuntimeManager) connectRuntime(ctx context.Context, runtime *connectorRuntime) error {
	if runtime == nil || runtime.connector == nil {
		return &ConnectorStartError{Reason: "connector is nil"}
	}
	runtime.setStatus(ConnectorStatusStopped)
	if err := runtime.connector.Connect(ctx); err != nil {
		runtime.setError(err)
		return &ConnectorStartError{Reason: err.Error()}
	}
	m.handleConnectorConnected(runtime)
	return nil
}

func (m *RuntimeManager) handleConnectorConnected(runtime *connectorRuntime) {
	if runtime == nil {
		return
	}
	if m.connectorRuntime(runtime.record.ID) != runtime {
		return
	}
	if runtime.getStatus() == ConnectorStatusRunning {
		return
	}
	runtime.setRunning()
	m.notifyConnectorGroups(runtime.record.ID)
}

func (m *RuntimeManager) handleConnectorError(runtime *connectorRuntime, err error) {
	if runtime == nil || err == nil {
		return
	}
	if m.connectorRuntime(runtime.record.ID) != runtime {
		return
	}
	runtime.setError(err)
}

func (m *RuntimeManager) StartConnector(ctx context.Context, id string) error {
	if m == nil {
		return ErrConnectorNotFound
	}
	m.mu.Lock()
	runtime := m.connectors[id]
	m.mu.Unlock()
	if runtime == nil {
		return ErrConnectorNotFound
	}
	if runtime.getStatus() == ConnectorStatusRunning {
		return ErrConnectorAlreadyRunning
	}
	return m.connectRuntime(ctx, runtime)
}

func (m *RuntimeManager) StopConnector(ctx context.Context, id string) error {
	if m == nil {
		return ErrConnectorNotFound
	}
	m.mu.Lock()
	runtime := m.connectors[id]
	m.mu.Unlock()
	if runtime == nil {
		return ErrConnectorNotFound
	}
	if runtime.getStatus() == ConnectorStatusStopped {
		return ErrConnectorAlreadyStopped
	}
	err := runtime.connector.Close(ctx)
	runtime.setStopped()
	if err != nil {
		return &ConnectorStopError{Reason: err.Error()}
	}
	return nil
}

func (m *RuntimeManager) ConnectorStatus(id string) string {
	if m == nil {
		return ConnectorStatusStopped
	}
	m.mu.Lock()
	runtime := m.connectors[id]
	m.mu.Unlock()
	return runtime.getStatus()
}

func (m *RuntimeManager) ConnectorLastError(id string) string {
	if m == nil {
		return ""
	}
	m.mu.Lock()
	runtime := m.connectors[id]
	m.mu.Unlock()
	return runtime.getLastError()
}

func (m *RuntimeManager) RemoveConnector(ctx context.Context, id string) {
	m.mu.Lock()
	current := m.connectors[id]
	delete(m.connectors, id)
	groups := make([]*groupRuntime, 0)
	for groupID, runtime := range m.groups {
		if runtime.group.ConnectorID == id {
			groups = append(groups, runtime)
			delete(m.groups, groupID)
		}
	}
	m.mu.Unlock()
	for _, group := range groups {
		stopGroupRuntime(group)
		group.hub.Close()
		group.commands.Close()
	}
	if current != nil {
		_ = current.connector.Close(ctx)
	}
}
