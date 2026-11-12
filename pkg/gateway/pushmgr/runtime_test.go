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
	connected  bool
	connectErr error
	payloads   [][]byte
	commands   CommandHandler
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
func (c *fakeConnector) Publish(_ context.Context, payload []byte) error {
	c.payloads = append(c.payloads, append([]byte(nil), payload...))
	return nil
}
func (c *fakeConnector) SubscribeCommands(_ context.Context, handler CommandHandler) error {
	c.commands = handler
	return nil
}

type fakeFactory struct{ connector *fakeConnector }

func (f *fakeFactory) Type() string { return ConnectorTypeMQTT }
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
	runtime.UpsertGroup(&PushGroup{ID: "group-1", ConnectorID: "conn-1", AllCollections: true, PushIntervalMs: 1000})
	runtime.pushGroup(context.Background(), "group-1")
	require.Len(t, connector.payloads, 1)

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
	assert.Equal(t, ConnectorStatusStopped, runtime.ConnectorStatus("conn-1"))
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
	assert.Equal(t, ConnectorStatusStopped, runtime.ConnectorStatus("conn-1"))
}
