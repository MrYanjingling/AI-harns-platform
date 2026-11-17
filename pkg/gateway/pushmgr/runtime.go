package pushmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/logging"
)

type connectorRuntime struct {
	record    *ConnectorRecord
	connector Connector
	mu        sync.RWMutex
	status    string
	lastError string
}

func (c *connectorRuntime) setStatus(status string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.status = status
	c.mu.Unlock()
}

func (c *connectorRuntime) getStatus() string {
	if c == nil {
		return ConnectorStatusStopped
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.status == "" {
		return ConnectorStatusStopped
	}
	return c.status
}

func (c *connectorRuntime) setRunning() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.status = ConnectorStatusRunning
	c.lastError = ""
	c.mu.Unlock()
}

func (c *connectorRuntime) setError(err error) {
	if c == nil {
		return
	}
	message := "connector failed"
	if err != nil {
		message = err.Error()
	}
	c.mu.Lock()
	c.status = ConnectorStatusError
	c.lastError = message
	c.mu.Unlock()
}

func (c *connectorRuntime) setStopped() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.status = ConnectorStatusStopped
	c.lastError = ""
	c.mu.Unlock()
}

func (c *connectorRuntime) getLastError() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}

type groupRuntime struct {
	group        *PushGroup
	hub          *payloadHub
	commands     *commandHub
	cancel       context.CancelFunc
	done         chan struct{}
	mu           sync.RWMutex
	running      bool
	lastError    string
	lastPushTime time.Time
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
	g.lastError = ""
	g.mu.Unlock()
}

func (g *groupRuntime) setStopped() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.running = false
	g.lastError = ""
	g.mu.Unlock()
}

func (g *groupRuntime) setError(err error) {
	if g == nil {
		return
	}
	message := "push failed"
	if err != nil {
		message = err.Error()
	}
	g.mu.Lock()
	g.lastError = message
	g.mu.Unlock()
}

func (g *groupRuntime) setPushSuccess(at time.Time) {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.lastError = ""
	g.lastPushTime = at
	g.mu.Unlock()
}

func (g *groupRuntime) errorMessage() string {
	if g == nil {
		return ""
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastError
}

func (g *groupRuntime) pushTime() time.Time {
	if g == nil {
		return time.Time{}
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastPushTime
}

type RuntimeManager struct {
	mu            sync.Mutex
	source        CollectionSource
	gatewayID     string
	factories     map[string]ConnectorFactory
	connectors    map[string]*connectorRuntime
	groups        map[string]*groupRuntime
	subscriptions map[string]map[string]struct{}
}

func NewRuntimeManager(source CollectionSource, gatewayID string, factories ...ConnectorFactory) (*RuntimeManager, error) {
	m := &RuntimeManager{
		source:        source,
		gatewayID:     gatewayID,
		factories:     make(map[string]ConnectorFactory, len(factories)),
		connectors:    make(map[string]*connectorRuntime),
		groups:        make(map[string]*groupRuntime),
		subscriptions: make(map[string]map[string]struct{}),
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
		handler.SetErrorHandler(runtime.setError)
	}
	m.mu.Lock()
	old := m.connectors[record.ID]
	m.connectors[record.ID] = runtime
	m.mu.Unlock()
	if old != nil {
		_ = old.connector.Close(ctx)
	}
	m.clearSubscriptions(record.ID)
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
	runtime.setRunning()
	m.startGroupsForConnector(runtime.record.ID)
	m.syncSubscriptions(runtime.record.ID)
	return nil
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
	m.clearSubscriptions(id)
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
		stopGroupRuntime(old)
		hub = old.hub
		commands = old.commands
	}
	runtime := &groupRuntime{group: copied, hub: hub, commands: commands}
	m.mu.Lock()
	m.groups[copied.ID] = runtime
	m.mu.Unlock()

	connectorID := copied.ConnectorID
	if old != nil && old.group.ConnectorID != connectorID {
		m.syncSubscriptions(old.group.ConnectorID)
	}
	if shouldRun {
		if m.connectorRunning(connectorID) {
			_ = m.startGroupRuntime(runtime)
		}
	}
	m.syncSubscriptions(connectorID)
}

func (m *RuntimeManager) startGroupsForConnector(connectorID string) {
	m.mu.Lock()
	runtimes := make([]*groupRuntime, 0)
	for _, runtime := range m.groups {
		if runtime.group.ConnectorID == connectorID &&
			runtime.group.RunState == ConnectorRunStateRunning &&
			!runtime.isRunning() {
			runtimes = append(runtimes, runtime)
		}
	}
	m.mu.Unlock()
	for _, runtime := range runtimes {
		_ = m.startGroupRuntime(runtime)
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
	if runtime.group.RunState != ConnectorRunStateRunning {
		return ErrGroupDisabled
	}
	connectorID := runtime.group.ConnectorID
	m.mu.Lock()
	connector := m.connectors[connectorID]
	m.mu.Unlock()
	if connector == nil || connector.getStatus() != ConnectorStatusRunning {
		return ErrGroupConnectorNotRunning
	}
	if runtime.isRunning() {
		runtime.setRunning()
		m.syncSubscriptions(connectorID)
		return nil
	}
	return m.startGroupRuntime(runtime)
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
	runtime.lastError = ""
	runtime.mu.Unlock()
	go m.loop(ctx, runtime.group.ID, done)
	m.syncSubscriptions(runtime.group.ConnectorID)
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
	stopGroupRuntime(runtime)
	return m.startGroupRuntime(runtime)
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
	connectorID := runtime.group.ConnectorID
	stopGroupRuntime(runtime)
	m.syncSubscriptions(connectorID)
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

func (m *RuntimeManager) GroupLastError(id string) string {
	runtime, connector := m.groupSnapshot(id)
	if runtime == nil || runtime.group.RunState != ConnectorRunStateRunning || !runtime.isRunning() {
		return ""
	}
	if runtime.errorMessage() != "" {
		return runtime.errorMessage()
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
	group.mu.Lock()
	cancel := group.cancel
	done := group.done
	group.cancel = nil
	group.done = nil
	group.running = false
	group.lastError = ""
	group.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
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
	delete(m.subscriptions, id)
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

func (m *RuntimeManager) RemoveGroup(id string) {
	m.mu.Lock()
	current := m.groups[id]
	delete(m.groups, id)
	m.mu.Unlock()
	if current != nil {
		connectorID := current.group.ConnectorID
		stopGroupRuntime(current)
		current.hub.Close()
		current.commands.Close()
		m.syncSubscriptions(connectorID)
	}
}

func (m *RuntimeManager) loop(ctx context.Context, groupID string, done chan struct{}) {
	defer close(done)
	for {
		m.mu.Lock()
		current := m.groups[groupID]
		m.mu.Unlock()
		if current == nil {
			return
		}
		interval := time.Duration(current.group.PushIntervalMs) * time.Millisecond
		if interval <= 0 {
			interval = time.Duration(DefaultPushIntervalMs) * time.Millisecond
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			m.pushGroup(ctx, groupID)
		}
	}
}

func (m *RuntimeManager) pushGroup(ctx context.Context, groupID string) {
	runtime, connector := m.groupSnapshot(groupID)
	if runtime == nil || connector == nil || !runtime.isRunning() {
		return
	}
	if connector.getStatus() != ConnectorStatusRunning {
		return
	}
	group := runtime.group
	snapshots, err := m.source.Snapshots(ctx, group.AllCollections, group.CollectionIDs)
	if err != nil {
		runtime.setError(err)
		logging.ErrorS(err, "Failed to snapshot collections for push", "groupId", group.ID)
		return
	}
	messages := buildPropertyMessages(snapshots, group.MaxCollectionsPerPushMessage)
	if len(messages) == 0 {
		return
	}
	lastSuccess := time.Time{}
	for index, message := range messages {
		data, err := json.Marshal(message)
		if err != nil {
			runtime.setError(err)
			logging.ErrorS(err, "Failed to marshal push payload", "groupId", group.ID, "batch", index)
			return
		}
		started := time.Now()
		publishErr := connector.connector.Publish(ctx, Message{
			Topic:   group.DataTopic,
			Key:     group.ID,
			Payload: data,
		})
		event := PushPayloadEvent{
			GroupID:       group.ID,
			ConnectorID:   group.ConnectorID,
			ConnectorType: connector.record.Type,
			Topic:         group.DataTopic,
			Key:           group.ID,
			Time:          started.UTC(),
			DurationMs:    time.Since(started).Milliseconds(),
			Payload:       json.RawMessage(data),
		}
		if publishErr != nil {
			event.Error = publishErr.Error()
			runtime.hub.Publish(event)
			runtime.setError(publishErr)
			connector.setError(publishErr)
			logging.ErrorS(publishErr, "Failed to publish push payload", "groupId", group.ID, "connectorId", group.ConnectorID, "batch", index)
			return
		}
		runtime.hub.Publish(event)
		lastSuccess = time.Now().UTC()
	}
	if !lastSuccess.IsZero() {
		runtime.setPushSuccess(lastSuccess)
	}
}

func (m *RuntimeManager) syncSubscriptions(connectorID string) {
	if connectorID == "" {
		return
	}
	m.mu.Lock()
	connector := m.connectors[connectorID]
	desired := make(map[string]struct{})
	if connector != nil && connector.getStatus() == ConnectorStatusRunning {
		for _, runtime := range m.groups {
			if runtime.group.ConnectorID != connectorID || !runtime.isRunning() {
				continue
			}
			topic := runtime.group.CommandTopic
			if topic != "" {
				desired[topic] = struct{}{}
			}
		}
	}
	current := m.subscriptions[connectorID]
	if current == nil {
		current = make(map[string]struct{})
	}
	m.mu.Unlock()
	if connector == nil {
		return
	}
	for topic := range desired {
		if _, ok := current[topic]; ok {
			continue
		}
		if err := connector.connector.Subscribe(context.Background(), topic, m.commandHandler(connectorID, topic)); err != nil {
			connector.setError(err)
			logging.ErrorS(err, "Failed to subscribe push command topic", "connectorId", connectorID, "topic", topic)
			continue
		}
		m.mu.Lock()
		if m.subscriptions[connectorID] == nil {
			m.subscriptions[connectorID] = make(map[string]struct{})
		}
		m.subscriptions[connectorID][topic] = struct{}{}
		m.mu.Unlock()
	}
	for topic := range current {
		if _, ok := desired[topic]; ok {
			continue
		}
		_ = connector.connector.Unsubscribe(context.Background(), topic)
		m.mu.Lock()
		delete(m.subscriptions[connectorID], topic)
		m.mu.Unlock()
	}
}

func (m *RuntimeManager) clearSubscriptions(connectorID string) {
	m.mu.Lock()
	delete(m.subscriptions, connectorID)
	m.mu.Unlock()
}

func (m *RuntimeManager) commandHandler(connectorID, topic string) CommandHandler {
	return func(ctx context.Context, payload []byte) {
		m.publishCommand(connectorID, topic, payload)
		m.handleCommand(ctx, payload)
	}
}

func (m *RuntimeManager) publishCommand(connectorID, topic string, payload []byte) {
	now := time.Now().UTC()
	m.mu.Lock()
	connectorType := ""
	if connector := m.connectors[connectorID]; connector != nil {
		connectorType = connector.connector.Type()
	}
	groups := make([]*groupRuntime, 0)
	for _, runtime := range m.groups {
		if runtime.group.ConnectorID == connectorID && runtime.group.CommandTopic == topic {
			groups = append(groups, runtime)
		}
	}
	m.mu.Unlock()

	raw := normalizeCommandPayload(payload)
	for _, runtime := range groups {
		runtime.commands.Publish(PushCommandEvent{
			GroupID:       runtime.group.ID,
			ConnectorID:   connectorID,
			ConnectorType: connectorType,
			Topic:         topic,
			Time:          now,
			Payload:       append(json.RawMessage(nil), raw...),
		})
	}
}

func normalizeCommandPayload(payload []byte) json.RawMessage {
	if json.Valid(payload) {
		return append(json.RawMessage(nil), payload...)
	}
	encoded, err := json.Marshal(string(payload))
	if err != nil {
		return json.RawMessage(`null`)
	}
	return encoded
}

func (m *RuntimeManager) handleCommand(ctx context.Context, payload []byte) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var message commandMessage
	if err := decoder.Decode(&message); err != nil {
		logging.ErrorS(err, "Failed to decode push command")
		return
	}
	if message.Type != "command" {
		return
	}
	for _, item := range message.Payload {
		actions := make([]Action, 0, len(item.Property))
		for name, value := range item.Property {
			converted, ok := commandValue(value)
			if !ok {
				continue
			}
			actions = append(actions, Action{Name: name, Value: converted})
		}
		if len(actions) == 0 {
			continue
		}
		logging.InfoS("Push command received", "collectionId", item.CollectionID, "messageId", item.MessageID, "actions", len(actions))
		if err := m.source.DeliverActions(ctx, item.CollectionID, actions); err != nil {
			logging.ErrorS(err, "Failed to deliver push command", "collectionId", item.CollectionID, "messageId", item.MessageID)
		}
	}
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
	m.subscriptions = make(map[string]map[string]struct{})
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

type propertyMessage struct {
	Type    string         `json:"type"`
	Payload []propertyItem `json:"payload"`
}

type propertyItem struct {
	CollectionID string            `json:"collectionId"`
	Time         int64             `json:"time"`
	Property     map[string]string `json:"property"`
}

func buildPropertyMessages(snapshots []CollectionSnapshot, maxPerMessage int) []propertyMessage {
	now := time.Now().UnixMilli()
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].CollectionID < snapshots[j].CollectionID })
	items := make([]propertyItem, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Status != collection.CollectStatusCollecting {
			continue
		}
		values := make(map[string]string, len(snapshot.Values))
		names := make([]string, 0, len(snapshot.Values))
		for name := range snapshot.Values {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			value, ok := propertyValue(snapshot.Values[name])
			if ok {
				values[name] = value
			}
		}
		timestamp := snapshot.Time.UnixMilli()
		if snapshot.Time.IsZero() {
			timestamp = now
		}
		items = append(items, propertyItem{
			CollectionID: snapshot.CollectionID,
			Time:         timestamp,
			Property:     values,
		})
	}
	if len(items) == 0 {
		return nil
	}
	if maxPerMessage <= 0 {
		return []propertyMessage{{Type: "property", Payload: items}}
	}
	out := make([]propertyMessage, 0, (len(items)+maxPerMessage-1)/maxPerMessage)
	for start := 0; start < len(items); start += maxPerMessage {
		end := start + maxPerMessage
		if end > len(items) {
			end = len(items)
		}
		out = append(out, propertyMessage{Type: "property", Payload: items[start:end]})
	}
	return out
}

func propertyValue(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), true
	case int:
		return strconv.FormatInt(int64(v), 10), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32), true
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), true
	case json.Number:
		return v.String(), true
	case []int:
		data, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(data), true
	case []byte:
		data, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(data), true
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(data), true
	}
}

func commandValue(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), true
	case json.Number:
		return v.String(), true
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), true
	default:
		return "", false
	}
}

type commandMessage struct {
	Type    string        `json:"type"`
	Payload []commandItem `json:"payload"`
}

type commandItem struct {
	CollectionID string         `json:"collectionId"`
	MessageID    string         `json:"messageId"`
	Property     map[string]any `json:"property"`
}

func cloneGroup(group *PushGroup) *PushGroup {
	copied := *group
	copied.CollectionIDs = append([]string(nil), group.CollectionIDs...)
	return &copied
}
