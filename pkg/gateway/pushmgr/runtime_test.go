package pushmgr

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

type fakeSource struct {
	snapshots  []CollectionSnapshot
	actions    []Action
	collection string
}

func (s *fakeSource) Snapshots(context.Context, bool, []string) ([]CollectionSnapshot, error) {
	return s.snapshots, nil
}
func (s *fakeSource) Exists(context.Context, string) (bool, error) { return true, nil }
func (s *fakeSource) DeliverActions(_ context.Context, collectionID string, actions []Action) error {
	s.collection = collectionID
	s.actions = append(s.actions, actions...)
	return nil
}

type fakeConnector struct {
	connected      bool
	connectErr     error
	publishErr     error
	subscribeErr   error
	payloads       [][]byte
	commands       CommandHandler
	subscriptions  map[string]int
	errorHandler   func(error)
	connectHandler func()
}

func (c *fakeConnector) Type() string { return ConnectorTypeMQTT }
func (c *fakeConnector) Connect(context.Context) error {
	if c.connectErr != nil {
		return c.connectErr
	}
	c.connected = true
	return nil
}
func (c *fakeConnector) Close(context.Context) error { return nil }
func (c *fakeConnector) Publish(_ context.Context, message Message) error {
	if c.publishErr != nil {
		return c.publishErr
	}
	c.payloads = append(c.payloads, append([]byte(nil), message.Payload...))
	return nil
}
func (c *fakeConnector) SetErrorHandler(handler func(error)) {
	c.errorHandler = handler
}
func (c *fakeConnector) SetConnectedHandler(handler func()) {
	c.connectHandler = handler
}
func (c *fakeConnector) Subscribe(_ context.Context, topic string, handler CommandHandler) error {
	if c.subscribeErr != nil {
		return c.subscribeErr
	}
	c.commands = handler
	if c.subscriptions == nil {
		c.subscriptions = make(map[string]int)
	}
	c.subscriptions[topic]++
	return nil
}
func (c *fakeConnector) Unsubscribe(_ context.Context, topic string) error {
	if c.subscriptions != nil {
		delete(c.subscriptions, topic)
	}
	return nil
}

type fakeFactory struct{ connector *fakeConnector }

func (f *fakeFactory) Type() string                   { return ConnectorTypeMQTT }
func (f *fakeFactory) Validate(json.RawMessage) error { return nil }
func (f *fakeFactory) New(string, json.RawMessage) (Connector, error) {
	return f.connector, nil
}

func TestBuildPropertyMessage(t *testing.T) {
	snapshotTime := time.Unix(1773827321, 0)
	messages := buildPropertyMessages([]CollectionSnapshot{
		{
			CollectionID: "c2",
			Status:       collection.CollectStatusCollecting,
			Time:         snapshotTime,
			Values: map[string]any{
				"a":     float32(25.6),
				"b":     true,
				"skip":  nil,
				"int64": int64(9007199254740993),
				"bytes": []int{1, 2, 3},
			},
		},
		{
			CollectionID: "stopped",
			Status:       collection.CollectStatusStopped,
			Time:         snapshotTime,
			Values:       map[string]any{"a": 1},
		},
	}, 0)
	require.Len(t, messages, 1)
	payload := messages[0]
	require.Equal(t, "property", payload.Type)
	require.Len(t, payload.Payload, 1)
	assert.Equal(t, "c2", payload.Payload[0].CollectionID)
	assert.Equal(t, snapshotTime.UnixMilli(), payload.Payload[0].Time)
	assert.Equal(t, map[string]string{
		"a":     "25.6",
		"b":     "true",
		"int64": "9007199254740993",
		"bytes": "[1,2,3]",
	}, payload.Payload[0].Property)
}

func TestBuildPropertyMessagesBatches(t *testing.T) {
	snapshots := make([]CollectionSnapshot, 0, 12)
	for i := 1; i <= 11; i++ {
		snapshots = append(snapshots, CollectionSnapshot{
			CollectionID: "c" + strconv.Itoa(i),
			Status:       collection.CollectStatusCollecting,
			Values:       map[string]any{"a": i},
		})
	}
	snapshots = append(snapshots, CollectionSnapshot{CollectionID: "stopped", Status: collection.CollectStatusStopped})

	messages := buildPropertyMessages(snapshots, 10)
	require.Len(t, messages, 2)
	assert.Len(t, messages[0].Payload, 10)
	assert.Len(t, messages[1].Payload, 1)

	messages = buildPropertyMessages(snapshots, 0)
	require.Len(t, messages, 1)
	assert.Len(t, messages[0].Payload, 11)
}

func TestRuntimeCommandCallsCollectionAction(t *testing.T) {
	source := &fakeSource{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: &fakeConnector{}})
	require.NoError(t, err)
	runtime.handleCommand(context.Background(), []byte(`{
		"type":"command",
		"payload":[{"collectionId":"c1","messageId":"m1","property":{"a":1,"b":true,"c":"auto","d":null}}]
	}`))
	assert.Equal(t, "c1", source.collection)
	assert.ElementsMatch(t, []Action{{Name: "a", Value: "1"}, {Name: "b", Value: "true"}, {Name: "c", Value: "auto"}}, source.actions)
}

func TestMQTTConnectorDefaults(t *testing.T) {
	connector, err := NewMQTTConnector("connector-1", json.RawMessage(`{
		"brokerUrl":"tcp://127.0.0.1:1883",
		"dataTopic":"data"
	}`))
	require.NoError(t, err)
	assert.Equal(t, "gateway-push-connector-1", connector.config.ClientID)
	assert.Equal(t, byte(1), connector.config.QoS)
	assert.Equal(t, 60, connector.config.KeepAlive)

	_, err = NewMQTTConnector("bad", json.RawMessage(`{"dataTopic":"data"}`))
	require.Error(t, err)
}

func TestRuntimePushesBatchPayload(t *testing.T) {
	source := &fakeSource{snapshots: []CollectionSnapshot{{
		CollectionID: "c1",
		Status:       collection.CollectStatusCollecting,
		Time:         time.Unix(1773827321, 0),
		Values:       map[string]any{"a": 25.6, "b": true, "skip": nil},
	}}}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	runtime.UpsertGroup(&PushGroup{ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group", RunState: ConnectorRunStateRunning, AllCollections: true, PushIntervalMs: 1000})
	runtime.enqueueGroupSnapshot(context.Background(), "group-1")
	require.Eventually(t, func() bool { return len(connector.payloads) == 1 }, time.Second, 10*time.Millisecond)

	var payload propertyMessage
	require.NoError(t, json.Unmarshal(connector.payloads[0], &payload))
	require.Len(t, payload.Payload, 1)
	assert.Equal(t, "property", payload.Type)
	assert.Equal(t, "c1", payload.Payload[0].CollectionID)
	assert.Equal(t, map[string]string{"a": "25.6", "b": "true"}, payload.Payload[0].Property)
	runtime.Close(context.Background())
}

func TestConnectorStatus(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	assert.Equal(t, ConnectorStatusStopped, runtime.ConnectorStatus("missing"))

	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	assert.Equal(t, ConnectorStatusRunning, runtime.ConnectorStatus("conn-1"))

	connector.connectErr = errors.New("broker unavailable")
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	assert.Equal(t, ConnectorStatusError, runtime.ConnectorStatus("conn-1"))
	assert.Equal(t, "broker unavailable", runtime.ConnectorLastError("conn-1"))
	runtime.Close(context.Background())
}

func TestRuntimeStartStopConnector(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	require.NoError(t, runtime.ReplaceConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)}, true))
	assert.Equal(t, ConnectorStatusRunning, runtime.ConnectorStatus("conn-1"))
	require.ErrorIs(t, runtime.StartConnector(context.Background(), "conn-1"), ErrConnectorAlreadyRunning)

	require.NoError(t, runtime.StopConnector(context.Background(), "conn-1"))
	assert.Equal(t, ConnectorStatusStopped, runtime.ConnectorStatus("conn-1"))
	require.ErrorIs(t, runtime.StopConnector(context.Background(), "conn-1"), ErrConnectorAlreadyStopped)

	require.NoError(t, runtime.StartConnector(context.Background(), "conn-1"))
	assert.Equal(t, ConnectorStatusRunning, runtime.ConnectorStatus("conn-1"))
	runtime.Close(context.Background())
}

func TestRuntimeConnectFailureStatus(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{connectErr: errors.New("broker unavailable")}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	err = runtime.ReplaceConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)}, true)
	require.Error(t, err)
	var startErr *ConnectorStartError
	require.ErrorAs(t, err, &startErr)
	assert.Equal(t, ConnectorStatusError, runtime.ConnectorStatus("conn-1"))
	assert.Equal(t, "broker unavailable", runtime.ConnectorLastError("conn-1"))
}

func TestConnectorErrorHandlerSetsErrorStatus(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	require.Equal(t, ConnectorStatusRunning, runtime.ConnectorStatus("conn-1"))

	connector.errorHandler(errors.New("mqtt connection lost"))
	assert.Equal(t, ConnectorStatusError, runtime.ConnectorStatus("conn-1"))
	assert.Equal(t, "mqtt connection lost", runtime.ConnectorLastError("conn-1"))
	runtime.Close(context.Background())
}

func TestConnectorReconnectRestoresRunningAndResubscribes(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	runtime.UpsertGroup(&PushGroup{
		ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group",
		CommandTopic: "command/group", RunState: ConnectorRunStateRunning, AllCollections: true,
		PushIntervalMs: 10000,
	})
	require.Equal(t, ConnectorStatusRunning, runtime.GroupStatus("group-1"))
	require.Equal(t, 1, connector.subscriptions["command/group"])

	connector.errorHandler(errors.New("mqtt connection lost: EOF"))
	require.Equal(t, ConnectorStatusError, runtime.ConnectorStatus("conn-1"))
	require.Equal(t, ConnectorStatusError, runtime.GroupStatus("group-1"))

	delete(connector.subscriptions, "command/group")

	connector.connectHandler()
	assert.Equal(t, ConnectorStatusRunning, runtime.ConnectorStatus("conn-1"))
	assert.Empty(t, runtime.ConnectorLastError("conn-1"))
	assert.Equal(t, ConnectorStatusRunning, runtime.GroupStatus("group-1"))
	require.Eventually(t, func() bool {
		return connector.subscriptions["command/group"] == 1
	}, time.Second, 10*time.Millisecond)
	runtime.Close(context.Background())
}

func TestRuntimePublishFailureSetsErrorStatus(t *testing.T) {
	source := &fakeSource{snapshots: []CollectionSnapshot{{
		CollectionID: "c1",
		Status:       collection.CollectStatusCollecting,
		Values:       map[string]any{"a": 1},
	}}}
	connector := &fakeConnector{publishErr: errors.New("publish failed")}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	runtime.UpsertGroup(&PushGroup{ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group", RunState: ConnectorRunStateRunning, AllCollections: true, PushIntervalMs: 1000})
	runtime.enqueueGroupSnapshot(context.Background(), "group-1")
	require.Eventually(t, func() bool {
		return runtime.GroupLastError("group-1") == "publish failed"
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, ConnectorStatusRunning, runtime.ConnectorStatus("conn-1"))
	assert.Empty(t, runtime.ConnectorLastError("conn-1"))
	assert.Equal(t, ConnectorStatusError, runtime.GroupStatus("group-1"))

	connector.publishErr = nil
	require.Eventually(t, func() bool {
		return runtime.GroupLastError("group-1") == ""
	}, 3*time.Second, 10*time.Millisecond)
	assert.Equal(t, ConnectorStatusRunning, runtime.GroupStatus("group-1"))
	require.Eventually(t, func() bool { return len(connector.payloads) > 0 }, time.Second, 10*time.Millisecond)
	runtime.Close(context.Background())
}

func TestSubscribeFailureSetsGroupError(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{subscribeErr: errors.New("subscribe failed")}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	runtime.UpsertGroup(&PushGroup{
		ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group", CommandTopic: "command/group",
		RunState: ConnectorRunStateRunning, AllCollections: true, PushIntervalMs: 10000,
	})
	assert.Equal(t, ConnectorStatusRunning, runtime.ConnectorStatus("conn-1"))
	assert.Equal(t, ConnectorStatusError, runtime.GroupStatus("group-1"))
	assert.Contains(t, runtime.GroupLastError("group-1"), "subscribe failed")

	connector.subscribeErr = nil
	require.Eventually(t, func() bool {
		return runtime.GroupStatus("group-1") == ConnectorStatusRunning &&
			connector.subscriptions["command/group"] == 1
	}, 3*time.Second, 50*time.Millisecond)
	runtime.Close(context.Background())
}

func TestGroupStartStopAndStatus(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	runtime.UpsertGroup(&PushGroup{
		ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group",
		CommandTopic: "command/group", RunState: ConnectorRunStateRunning, AllCollections: true,
		PushIntervalMs: 10000,
	})
	assert.Equal(t, ConnectorStatusRunning, runtime.GroupStatus("group-1"))
	assert.Equal(t, 1, connector.subscriptions["command/group"])

	require.NoError(t, runtime.StopGroup("group-1"))
	assert.Equal(t, ConnectorStatusStopped, runtime.GroupStatus("group-1"))
	assert.Equal(t, 0, connector.subscriptions["command/group"])
	require.ErrorIs(t, runtime.StopGroup("group-1"), ErrGroupAlreadyStopped)

	require.NoError(t, runtime.StartGroup("group-1"))
	assert.Equal(t, ConnectorStatusRunning, runtime.GroupStatus("group-1"))
	assert.Equal(t, 1, connector.subscriptions["command/group"])
	runtime.Close(context.Background())
}

func TestGroupOwnsCommandSubscription(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	runtime.UpsertGroup(&PushGroup{
		ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/1", CommandTopic: "command/one",
		RunState: ConnectorRunStateRunning, AllCollections: true, PushIntervalMs: 10000,
	})
	runtime.UpsertGroup(&PushGroup{
		ID: "group-2", ConnectorID: "conn-1", DataTopic: "data/2", CommandTopic: "command/two",
		RunState: ConnectorRunStateRunning, AllCollections: true, PushIntervalMs: 10000,
	})
	assert.Equal(t, 1, connector.subscriptions["command/one"])
	assert.Equal(t, 1, connector.subscriptions["command/two"])

	disabled := &PushGroup{
		ID: "group-2", ConnectorID: "conn-1", DataTopic: "data/2", CommandTopic: "command/two",
		RunState: ConnectorRunStateStopped, AllCollections: true, PushIntervalMs: 10000,
	}
	runtime.UpsertGroup(disabled)
	assert.Equal(t, ConnectorStatusStopped, runtime.GroupStatus("group-2"))
	assert.Equal(t, 1, connector.subscriptions["command/one"])
	assert.Equal(t, 0, connector.subscriptions["command/two"])

	runtime.RemoveGroup("group-1")
	assert.Equal(t, 0, connector.subscriptions["command/one"])
	runtime.Close(context.Background())
}

func TestGroupPayloadHubReceivesSuccess(t *testing.T) {
	source := &fakeSource{snapshots: []CollectionSnapshot{{
		CollectionID: "c1",
		Status:       collection.CollectStatusCollecting,
		Values:       map[string]any{"a": 1},
	}}}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	runtime.UpsertGroup(&PushGroup{
		ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group",
		RunState: ConnectorRunStateRunning, AllCollections: true, PushIntervalMs: 10000,
	})
	ch, cancel, err := runtime.SubscribePayloads("group-1")
	require.NoError(t, err)
	defer cancel()
	runtime.enqueueGroupSnapshot(context.Background(), "group-1")
	select {
	case event := <-ch:
		require.Equal(t, "data/group", event.Topic)
		require.NotEmpty(t, event.Payload)
		require.Empty(t, event.Error)
	case <-time.After(time.Second):
		t.Fatal("expected push payload event")
	}
	require.False(t, runtime.GroupLastPushTime("group-1").IsZero())
	runtime.Close(context.Background())
}

func TestGroupCommandHubReceivesInbound(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`)})
	runtime.UpsertGroup(&PushGroup{
		ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group", CommandTopic: "command/group",
		RunState: ConnectorRunStateRunning, AllCollections: true, PushIntervalMs: 10000,
	})
	ch, cancel, err := runtime.SubscribeCommands("group-1")
	require.NoError(t, err)
	defer cancel()

	connector.commands(context.Background(), []byte(`{
		"type":"command",
		"payload":[{"collectionId":"c1","messageId":"m1","property":{"a":"1"}}]
	}`))
	select {
	case event := <-ch:
		assert.Equal(t, "command/group", event.Topic)
		assert.Equal(t, "group-1", event.GroupID)
		assert.Contains(t, string(event.Payload), `"collectionId":"c1"`)
	case <-time.After(time.Second):
		t.Fatal("expected command event")
	}
	assert.Equal(t, "c1", source.collection)
	require.Len(t, source.actions, 1)
	runtime.Close(context.Background())
}

func TestReconnectStartsGroupCreatedWhileConnectorDown(t *testing.T) {
	source := &fakeSource{}
	connector := &fakeConnector{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{
		ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`),
	})
	connector.errorHandler(errors.New("mqtt connection lost"))
	require.Equal(t, ConnectorStatusError, runtime.ConnectorStatus("conn-1"))

	runtime.UpsertGroup(&PushGroup{
		ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group",
		CommandTopic: "command/group", RunState: ConnectorRunStateRunning,
		AllCollections: true, PushIntervalMs: 10000,
	})
	assert.False(t, runtime.GroupRunning("group-1"))

	connector.connectHandler()
	require.Eventually(t, func() bool {
		return runtime.GroupRunning("group-1") &&
			runtime.GroupStatus("group-1") == ConnectorStatusRunning &&
			connector.subscriptions["command/group"] == 1
	}, 3*time.Second, 20*time.Millisecond)
	runtime.Close(context.Background())
}

func TestGroupPushAndSubscriptionErrorsAreSeparate(t *testing.T) {
	group := &groupRuntime{}
	group.setSubscriptionError(errors.New("subscribe failed"))
	group.setPushSuccess(time.Now())
	assert.Equal(t, "subscribe failed", group.errorMessage())

	group.setPushError(errors.New("publish failed"))
	assert.Equal(t, "publish failed", group.errorMessage())

	group.clearSubscriptionError()
	assert.Equal(t, "publish failed", group.errorMessage())

	group.setPushSuccess(time.Now())
	assert.Empty(t, group.errorMessage())
}
