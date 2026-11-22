package pushmgr

import (
	"context"
	"strings"
	"time"

	"harns-platform/pkg/logging"
)

func (m *RuntimeManager) subscribeGroup(runtime *groupRuntime, force bool) error {
	if runtime == nil {
		return ErrGroupNotFound
	}
	topic := strings.TrimSpace(runtime.group.CommandTopic)
	if topic == "" {
		m.unsubscribeGroup(runtime)
		return nil
	}
	connectorID := runtime.group.ConnectorID
	connector := m.connectorRuntime(connectorID)
	if connector == nil || connector.getStatus() != ConnectorStatusRunning {
		return errPushConnectorUnavailable
	}

	runtime.subMu.Lock()
	if !force && runtime.subscribedTopic == topic && runtime.subscribedConnectorID == connectorID {
		runtime.subMu.Unlock()
		return nil
	}
	oldTopic := runtime.subscribedTopic
	oldConnectorID := runtime.subscribedConnectorID
	if oldTopic != "" && (oldTopic != topic || oldConnectorID != connectorID) {
		if oldConnector := m.connectorRuntime(oldConnectorID); oldConnector != nil {
			_ = oldConnector.connector.Unsubscribe(context.Background(), oldTopic)
		}
		runtime.subscribedTopic = ""
		runtime.subscribedConnectorID = ""
	}
	err := connector.connector.Subscribe(context.Background(), topic, m.groupCommandHandler(runtime, connectorID, topic))
	if err != nil {
		runtime.subMu.Unlock()
		runtime.setSubscriptionError(err)
		m.scheduleSubscriptionRetry(runtime)
		return err
	}
	runtime.subscribedTopic = topic
	runtime.subscribedConnectorID = connectorID
	runtime.subMu.Unlock()
	runtime.clearSubscriptionError()
	return nil
}

func (m *RuntimeManager) unsubscribeGroup(runtime *groupRuntime) {
	if runtime == nil {
		return
	}
	runtime.cancelSubscriptionRetry()
	runtime.subMu.Lock()
	topic := runtime.subscribedTopic
	connectorID := runtime.subscribedConnectorID
	runtime.subscribedTopic = ""
	runtime.subscribedConnectorID = ""
	runtime.subMu.Unlock()
	if topic == "" {
		return
	}
	if connector := m.connectorRuntime(connectorID); connector != nil {
		_ = connector.connector.Unsubscribe(context.Background(), topic)
	}
}

func (m *RuntimeManager) scheduleSubscriptionRetry(runtime *groupRuntime) {
	if runtime == nil {
		return
	}
	runtime.subMu.Lock()
	if runtime.subRetryCancel != nil {
		runtime.subMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	runtime.subRetryCancel = cancel
	runtime.subRetryDone = done
	runtime.subMu.Unlock()

	go func() {
		defer close(done)
		delay := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			if !runtime.isRunning() {
				return
			}
			if err := m.subscribeGroup(runtime, true); err == nil {
				runtime.subMu.Lock()
				if runtime.subRetryDone == done {
					runtime.subRetryCancel = nil
					runtime.subRetryDone = nil
				}
				runtime.subMu.Unlock()
				return
			}
			if delay < 30*time.Second {
				delay *= 2
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
			}
		}
	}()
}

func (g *groupRuntime) cancelSubscriptionRetry() {
	if g == nil {
		return
	}
	g.subMu.Lock()
	cancel := g.subRetryCancel
	done := g.subRetryDone
	g.subRetryCancel = nil
	g.subRetryDone = nil
	g.subMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (m *RuntimeManager) notifyConnectorGroups(connectorID string) {
	m.mu.Lock()
	groups := make([]*groupRuntime, 0)
	for _, runtime := range m.groups {
		if runtime.group.ConnectorID == connectorID {
			groups = append(groups, runtime)
		}
	}
	m.mu.Unlock()
	for _, runtime := range groups {
		go m.groupConnectorConnected(runtime)
	}
}

func (m *RuntimeManager) groupConnectorConnected(runtime *groupRuntime) {
	if runtime == nil || runtime.group.RunState != ConnectorRunStateRunning {
		return
	}
	if !runtime.isRunning() {
		if err := m.startGroupRuntime(runtime); err != nil {
			logging.ErrorS(err, "Failed to start push group after reconnect", "groupId", runtime.group.ID)
			return
		}
	}
	if err := m.subscribeGroup(runtime, true); err != nil {
		logging.ErrorS(err, "Failed to subscribe push group after reconnect",
			"groupId", runtime.group.ID, "topic", runtime.group.CommandTopic)
	}
	m.wakeGroup(runtime)
}

func (m *RuntimeManager) wakeGroup(runtime *groupRuntime) {
	if runtime == nil {
		return
	}
	runtime.mu.RLock()
	wake := runtime.outboxWake
	runtime.mu.RUnlock()
	if wake == nil {
		return
	}
	select {
	case wake <- struct{}{}:
	default:
	}
}
