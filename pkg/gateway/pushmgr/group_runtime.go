package pushmgr

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"harns-platform/pkg/logging"
)

type groupRuntime struct {
	group                 *PushGroup
	hub                   *payloadHub
	commands              *commandHub
	cancel                context.CancelFunc
	done                  chan struct{}
	outbox                *groupOutbox
	outboxInput           chan []propertyMessage
	outboxWake            chan struct{}
	outboxDone            chan struct{}
	mu                    sync.RWMutex
	subMu                 sync.Mutex
	running               bool
	pushError             string
	subscriptionError     string
	subscribedTopic       string
	subscribedConnectorID string
	subRetryCancel        context.CancelFunc
	subRetryDone          chan struct{}
	lastPushTime          time.Time
	pendingMsgs           atomic.Int64
	pendingRows           atomic.Int64
	oldestPending         atomic.Int64
	backlog               atomic.Bool
}

func (g *groupRuntime) isRunning() bool {
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.running
}

func (g *groupRuntime) setRunning() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.running = true
	g.mu.Unlock()
}

func (g *groupRuntime) setStopped() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.running = false
	g.pushError = ""
	g.subscriptionError = ""
	g.mu.Unlock()
}

func (g *groupRuntime) setPushError(err error) {
	if g == nil {
		return
	}
	message := "push failed"
	if err != nil {
		message = err.Error()
	}
	g.mu.Lock()
	g.pushError = message
	g.mu.Unlock()
}

func (g *groupRuntime) setSubscriptionError(err error) {
	if g == nil {
		return
	}
	message := "subscription failed"
	if err != nil {
		message = err.Error()
	}
	g.mu.Lock()
	g.subscriptionError = message
	g.mu.Unlock()
}

func (g *groupRuntime) clearSubscriptionError() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.subscriptionError = ""
	g.mu.Unlock()
}

func (g *groupRuntime) setPushSuccess(at time.Time) {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.pushError = ""
	g.lastPushTime = at
	g.mu.Unlock()
}

func (g *groupRuntime) errorMessage() string {
	if g == nil {
		return ""
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.pushError != "" {
		return g.pushError
	}
	return g.subscriptionError
}

func (g *groupRuntime) pushTime() time.Time {
	if g == nil {
		return time.Time{}
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastPushTime
}

func (m *RuntimeManager) UpsertGroup(group *PushGroup) {
	m.upsertGroup(group)
}

func (m *RuntimeManager) upsertGroup(group *PushGroup) {
	if m == nil || group == nil {
		return
	}
	copied := cloneGroup(group)
	if copied.PushIntervalMs <= 0 {
		copied.PushIntervalMs = DefaultPushIntervalMs
	}
	m.mu.Lock()
	old := m.groups[copied.ID]
	m.mu.Unlock()

	shouldRun := copied.RunState == ConnectorRunStateRunning && (old == nil || old.isRunning())
	hub := newPayloadHub()
	commands := newCommandHub()
	if old != nil {
		m.unsubscribeGroup(old)
		stopGroupRuntime(old)
		hub = old.hub
		commands = old.commands
	}
	runtime := &groupRuntime{group: copied, hub: hub, commands: commands}
	runtime.outbox = m.newGroupOutbox(runtime)
	runtime.outbox.seed(context.Background())
	m.mu.Lock()
	m.groups[copied.ID] = runtime
	m.mu.Unlock()

	if shouldRun {
		if m.connectorRunning(copied.ConnectorID) {
			if err := m.startGroupRuntime(runtime); err == nil {
				_ = m.subscribeGroup(runtime, true)
			}
		}
	}
}

func (m *RuntimeManager) connectorRunning(connectorID string) bool {
	m.mu.Lock()
	connector := m.connectors[connectorID]
	m.mu.Unlock()
	return connector != nil && connector.getStatus() == ConnectorStatusRunning
}

func (m *RuntimeManager) StartGroup(id string) error {
	m.mu.Lock()
	runtime := m.groups[id]
	m.mu.Unlock()
	if runtime == nil {
		return ErrGroupNotFound
	}
	runtime.mu.Lock()
	runtime.group.RunState = ConnectorRunStateRunning
	runtime.mu.Unlock()
	connectorID := runtime.group.ConnectorID
	m.mu.Lock()
	connector := m.connectors[connectorID]
	m.mu.Unlock()
	if connector == nil || connector.getStatus() != ConnectorStatusRunning {
		return ErrGroupConnectorNotRunning
	}
	if runtime.isRunning() {
		runtime.setRunning()
		_ = m.subscribeGroup(runtime, true)
		return nil
	}
	if err := m.startGroupRuntime(runtime); err != nil {
		return err
	}
	_ = m.subscribeGroup(runtime, true)
	return nil
}

func (m *RuntimeManager) startGroupRuntime(runtime *groupRuntime) error {
	if runtime == nil {
		return ErrGroupNotFound
	}
	runtime.mu.Lock()
	if runtime.running {
		runtime.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	runtime.cancel = cancel
	runtime.done = done
	runtime.running = true
	input := make(chan []propertyMessage, 256)
	wake := make(chan struct{}, 1)
	outboxDone := make(chan struct{})
	runtime.outboxInput = input
	runtime.outboxWake = wake
	runtime.outboxDone = outboxDone
	runtime.mu.Unlock()
	go m.groupProducerLoop(ctx, runtime.group.ID, done)
	go runtime.outbox.run(ctx, input, wake, outboxDone)
	return nil
}

func (m *RuntimeManager) GroupRunning(id string) bool {
	m.mu.Lock()
	runtime := m.groups[id]
	m.mu.Unlock()
	return runtime != nil && runtime.isRunning()
}

func (m *RuntimeManager) RestartGroup(id string) error {
	m.mu.Lock()
	runtime := m.groups[id]
	m.mu.Unlock()
	if runtime == nil {
		return ErrGroupNotFound
	}
	if runtime.group.RunState != ConnectorRunStateRunning {
		return ErrGroupAlreadyStopped
	}
	connectorID := runtime.group.ConnectorID
	if !m.connectorRunning(connectorID) {
		return ErrGroupConnectorNotRunning
	}
	m.unsubscribeGroup(runtime)
	stopGroupRuntime(runtime)
	if err := m.startGroupRuntime(runtime); err != nil {
		return err
	}
	_ = m.subscribeGroup(runtime, true)
	return nil
}

func (m *RuntimeManager) StopGroup(id string) error {
	m.mu.Lock()
	runtime := m.groups[id]
	m.mu.Unlock()
	if runtime == nil {
		return ErrGroupNotFound
	}
	if !runtime.isRunning() {
		return ErrGroupAlreadyStopped
	}
	m.unsubscribeGroup(runtime)
	stopGroupRuntime(runtime)
	if err := runtime.outbox.flush(context.Background()); err != nil {
		logging.ErrorS(err, "Failed to flush push group memory before stop", "groupId", id)
	}
	return nil
}

func (m *RuntimeManager) GroupStatus(id string) string {
	runtime, connector := m.groupSnapshot(id)
	if runtime == nil || connector == nil {
		return ConnectorStatusStopped
	}
	if runtime.group.RunState != ConnectorRunStateRunning || !runtime.isRunning() {
		return ConnectorStatusStopped
	}
	if connector.getStatus() == ConnectorStatusError {
		return ConnectorStatusError
	}
	if connector.getStatus() != ConnectorStatusRunning {
		return ConnectorStatusStopped
	}
	if runtime.errorMessage() != "" {
		return ConnectorStatusError
	}
	return ConnectorStatusRunning
}

func (m *RuntimeManager) GroupErrors(id string) (string, string) {
	runtime, _ := m.groupSnapshot(id)
	if runtime == nil || runtime.group.RunState != ConnectorRunStateRunning || !runtime.isRunning() {
		return "", ""
	}
	runtime.mu.RLock()
	pushErr := runtime.pushError
	subscriptionErr := runtime.subscriptionError
	runtime.mu.RUnlock()
	return pushErr, subscriptionErr
}

func (m *RuntimeManager) GroupLastError(id string) string {
	runtime, connector := m.groupSnapshot(id)
	if runtime == nil || runtime.group.RunState != ConnectorRunStateRunning || !runtime.isRunning() {
		return ""
	}
	pushErr, subscriptionErr := m.GroupErrors(id)
	if pushErr != "" {
		return pushErr
	}
	if subscriptionErr != "" {
		return subscriptionErr
	}
	if connector != nil {
		return connector.getLastError()
	}
	return ""
}

func (m *RuntimeManager) GroupLastPushTime(id string) time.Time {
	m.mu.Lock()
	runtime := m.groups[id]
	m.mu.Unlock()
	if runtime == nil {
		return time.Time{}
	}
	return runtime.pushTime()
}

// GroupPending returns in-memory plus persisted ready_send counters.
func (m *RuntimeManager) GroupPending(id string) (int64, int64, time.Time) {
	if m == nil {
		return 0, 0, time.Time{}
	}
	m.mu.Lock()
	runtime := m.groups[id]
	m.mu.Unlock()
	if runtime == nil {
		return 0, 0, time.Time{}
	}
	oldest := runtime.oldestPending.Load()
	var oldestTime time.Time
	if oldest > 0 {
		oldestTime = time.Unix(0, oldest).UTC()
	}
	return runtime.pendingMsgs.Load(), runtime.pendingRows.Load(), oldestTime
}

func (m *RuntimeManager) groupSnapshot(id string) (*groupRuntime, *connectorRuntime) {
	m.mu.Lock()
	runtime := m.groups[id]
	var connector *connectorRuntime
	if runtime != nil {
		connector = m.connectors[runtime.group.ConnectorID]
	}
	m.mu.Unlock()
	return runtime, connector
}

func (m *RuntimeManager) SubscribePayloads(groupID string) (<-chan PushPayloadEvent, func(), error) {
	m.mu.Lock()
	runtime := m.groups[groupID]
	m.mu.Unlock()
	if runtime == nil {
		return nil, nil, ErrGroupNotFound
	}
	ch, cancel := runtime.hub.Subscribe()
	return ch, cancel, nil
}

func (m *RuntimeManager) SubscribeCommands(groupID string) (<-chan PushCommandEvent, func(), error) {
	m.mu.Lock()
	runtime := m.groups[groupID]
	m.mu.Unlock()
	if runtime == nil {
		return nil, nil, ErrGroupNotFound
	}
	ch, cancel := runtime.commands.Subscribe()
	return ch, cancel, nil
}

func stopGroupRuntime(group *groupRuntime) {
	if group == nil {
		return
	}
	group.cancelSubscriptionRetry()
	group.mu.Lock()
	cancel := group.cancel
	done := group.done
	outboxDone := group.outboxDone
	group.cancel = nil
	group.done = nil
	group.outboxDone = nil
	group.running = false
	group.pushError = ""
	group.subscriptionError = ""
	group.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	if outboxDone != nil {
		<-outboxDone
	}
}

func (m *RuntimeManager) RemoveGroup(id string) {
	m.mu.Lock()
	current := m.groups[id]
	delete(m.groups, id)
	m.mu.Unlock()
	if current != nil {
		m.unsubscribeGroup(current)
		stopGroupRuntime(current)
		current.hub.Close()
		current.commands.Close()
	}
}

func (m *RuntimeManager) groupProducerLoop(ctx context.Context, groupID string, done chan struct{}) {
	defer close(done)
	for {
		m.mu.Lock()
		current := m.groups[groupID]
		m.mu.Unlock()
		if current == nil {
			return
		}
		timer := time.NewTimer(pushInterval(current))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			m.enqueueGroupSnapshot(ctx, groupID)
		}
	}
}

func pushInterval(runtime *groupRuntime) time.Duration {
	interval := time.Duration(DefaultPushIntervalMs) * time.Millisecond
	if runtime != nil && runtime.group != nil && runtime.group.PushIntervalMs > 0 {
		interval = time.Duration(runtime.group.PushIntervalMs) * time.Millisecond
	}
	return interval
}

func (m *RuntimeManager) enqueueGroupSnapshot(ctx context.Context, groupID string) {
	runtime, _ := m.groupSnapshot(groupID)
	if runtime == nil || !runtime.isRunning() {
		return
	}
	group := runtime.group
	snapshots, err := m.source.Snapshots(ctx, group.AllCollections, group.CollectionIDs)
	if err != nil {
		runtime.setPushError(err)
		logging.ErrorS(err, "Failed to snapshot collections for push", "groupId", group.ID)
		return
	}
	messages := buildPropertyMessages(snapshots, group.MaxCollectionsPerPushMessage)
	if len(messages) == 0 {
		return
	}
	runtime.mu.RLock()
	input := runtime.outboxInput
	runtime.mu.RUnlock()
	if input == nil {
		return
	}
	select {
	case input <- messages:
	case <-ctx.Done():
	}
}

func cloneGroup(group *PushGroup) *PushGroup {
	copied := *group
	copied.CollectionIDs = append([]string(nil), group.CollectionIDs...)
	return &copied
}
