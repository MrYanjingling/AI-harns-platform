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

type groupRuntime struct {
	group  *PushGroup
	cancel context.CancelFunc
	done   chan struct{}
}

// RuntimeManager owns connector connections and group push tickers.
type RuntimeManager struct {
	mu         sync.Mutex
	source     CollectionSource
	gatewayID  string
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

func (m *RuntimeManager) Start(ctx context.Context, connectors []*ConnectorRecord, groups []*PushGroup) error {
	for _, connector := range connectors {
		m.UpsertConnector(ctx, connector)
	}
	for _, group := range groups {
		m.upsertGroup(group)
	}
	return nil
}

func (m *RuntimeManager) UpsertConnector(ctx context.Context, record *ConnectorRecord) {
	_ = m.ReplaceConnector(ctx, record, true)
}

// ReplaceConnector installs a connector runtime and optionally connects it.
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
	if err := m.connectRuntime(ctx, runtime); err != nil {
		return err
	}
	return nil
}

func (m *RuntimeManager) connectRuntime(ctx context.Context, runtime *connectorRuntime) error {
	if runtime == nil || runtime.connector == nil {
		return &ConnectorStartError{Reason: "connector is nil"}
	}
	runtime.setStatus(ConnectorStatusStopped)
	if err := runtime.connector.Connect(ctx); err != nil {
		return &ConnectorStartError{Reason: err.Error()}
	}
	if err := runtime.connector.SubscribeCommands(ctx, m.handleCommand); err != nil {
		return &ConnectorStartError{Reason: err.Error()}
	}
	runtime.setStatus(ConnectorStatusRunning)
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
	runtime.setStatus(ConnectorStatusStopped)
	if err != nil {
		return &ConnectorStopError{Reason: err.Error()}
	}
	return nil
}

// ConnectorStatus returns the live connector status.
func (m *RuntimeManager) ConnectorStatus(id string) string {
	if m == nil {
		return ConnectorStatusStopped
	}
	m.mu.Lock()
	runtime := m.connectors[id]
	m.mu.Unlock()
	return runtime.getStatus()
}

func stopGroup(group *groupRuntime) {
	if group == nil {
		return
	}
	group.cancel()
	<-group.done
}

func (m *RuntimeManager) RemoveConnector(ctx context.Context, id string) {
	m.mu.Lock()
	current := m.connectors[id]
	delete(m.connectors, id)
	groups := make([]*groupRuntime, 0)
	for groupID, group := range m.groups {
		if group.group.ConnectorID == id {
			groups = append(groups, group)
			delete(m.groups, groupID)
		}
	}
	m.mu.Unlock()
	for _, group := range groups {
		stopGroup(group)
	}
	if current != nil {
		_ = current.connector.Close(ctx)
	}
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
	if old != nil {
		stopGroup(old)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	runtime := &groupRuntime{group: copied, cancel: cancel, done: done}
	m.mu.Lock()
	m.groups[copied.ID] = runtime
	m.mu.Unlock()
	go m.loop(ctx, copied.ID, done)
}

func (m *RuntimeManager) RemoveGroup(id string) {
	m.mu.Lock()
	current := m.groups[id]
	delete(m.groups, id)
	m.mu.Unlock()
	if current != nil {
		stopGroup(current)
	}
}

func (m *RuntimeManager) loop(ctx context.Context, groupID string, done chan struct{}) {
	defer close(done)
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
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pushGroup(ctx, groupID)
		}
	}
}

func (m *RuntimeManager) pushGroup(ctx context.Context, groupID string) {
	m.mu.Lock()
	groupRuntime := m.groups[groupID]
	m.mu.Unlock()
	if groupRuntime == nil {
		return
	}
	group := groupRuntime.group
	m.mu.Lock()
	connectorRuntime := m.connectors[group.ConnectorID]
	m.mu.Unlock()
	if connectorRuntime == nil || connectorRuntime.connector == nil || connectorRuntime.getStatus() != ConnectorStatusRunning {
		return
	}
	snapshots, err := m.source.Snapshots(ctx, group.AllCollections, group.CollectionIDs)
	if err != nil {
		logging.ErrorS(err, "Failed to snapshot collections for push", "groupId", group.ID)
		return
	}
	messages := buildPropertyMessages(snapshots, group.MaxCollectionsPerPushMessage)
	for index, message := range messages {
		data, err := json.Marshal(message)
		if err != nil {
			logging.ErrorS(err, "Failed to marshal push payload", "groupId", group.ID, "batch", index)
			continue
		}
		if err := connectorRuntime.connector.Publish(ctx, data); err != nil {
			logging.ErrorS(err, "Failed to publish push payload", "groupId", group.ID, "connectorId", group.ConnectorID, "batch", index)
		}
	}
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
	m.mu.Unlock()
	for _, group := range groups {
		stopGroup(group)
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
