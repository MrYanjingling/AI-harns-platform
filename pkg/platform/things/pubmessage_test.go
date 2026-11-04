package things

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func publishEvent(payload string) *EmqxPubEventMessage {
	return &EmqxPubEventMessage{
		Event: EmqxEventMessagePublish,
		Topic: "iot/device_001/up",
		ClientAttrs: map[string]string{
			ClientAttrsDeviceKey:     "device_001",
			ClientAttrsThingID:       "thing-1",
			ClientAttrsThingModelID:  "model-1",
			ClientAttrsThingModelKey: "tm_test",
			ClientAttrsDeviceName:    "设备1",
		},
		Payload: payload,
		ID:      "msg-1",
		QoS:     1,
	}
}

func TestResolveInboundKind(t *testing.T) {
	cases := []struct {
		payload string
		kind    MessageKind
		ok      bool
	}{
		{`{"type":"property"}`, MessageKindProperty, true},
		{`{"type":"event"}`, MessageKindEvent, true},
		{`{"type":"command_reply"}`, MessageKindCommandReply, true},
		{`{"temperature":25.6}`, "", false},
		{`{"type":"command"}`, MessageKindCommand, false},
		{`{"type":"unknown"}`, "", false},
	}
	for _, tc := range cases {
		kind, ok := ResolveInboundKind(tc.payload)
		assert.Equal(t, tc.ok, ok, tc.payload)
		assert.Equal(t, tc.kind, kind, tc.payload)
	}
}

func TestBuildDeviceMessageFromEvent(t *testing.T) {
	msg, ok := BuildDeviceMessageFromEvent(publishEvent(`{"type":"property","temperature":25.6}`))
	require.True(t, ok)
	assert.Equal(t, MessageKindProperty, msg.Kind)
	assert.Equal(t, "iot/device_001/up", msg.Topic)
	assert.Equal(t, "device_001", msg.DeviceKey)
	assert.Equal(t, "thing-1", msg.ThingID)
	assert.Equal(t, "model-1", msg.ThingModelID)
	assert.Equal(t, "tm_test", msg.ThingModelKey)
	assert.Equal(t, "设备1", msg.DeviceName)
	assert.Equal(t, "mqtt", msg.Source)
	assert.Equal(t, "msg-1", msg.MessageID)
	assert.Equal(t, 1, msg.QoS)

	_, ok = BuildDeviceMessageFromEvent(publishEvent(`{"type":"command"}`))
	assert.False(t, ok)

	_, ok = BuildDeviceMessageFromEvent(publishEvent(`{"temperature":25.6}`))
	assert.False(t, ok)

	notPublish := publishEvent(`{"type":"property"}`)
	notPublish.Event = "client.connected"
	_, ok = BuildDeviceMessageFromEvent(notPublish)
	assert.False(t, ok)

	noDevice := publishEvent(`{"type":"property"}`)
	noDevice.ClientAttrs = nil
	_, ok = BuildDeviceMessageFromEvent(noDevice)
	assert.False(t, ok)
}

type stubHandler struct {
	called bool
	err    error
	last   *DeviceMessage
}

func (s *stubHandler) Handle(ctx context.Context, msg *DeviceMessage) error {
	s.called = true
	s.last = msg
	return s.err
}

func TestMessageRouter(t *testing.T) {
	property := &stubHandler{}
	event := &stubHandler{}
	reply := &stubHandler{}
	router := NewMessageRouter(map[MessageKind]MessageHandler{
		MessageKindProperty:     property,
		MessageKindEvent:        event,
		MessageKindCommandReply: reply,
	})

	require.NoError(t, router.Handle(context.Background(), &DeviceMessage{Kind: MessageKindProperty}))
	assert.True(t, property.called)
	assert.False(t, event.called || reply.called)

	property.called = false
	require.NoError(t, router.Handle(context.Background(), &DeviceMessage{Kind: MessageKindEvent}))
	assert.True(t, event.called)

	event.called = false
	require.NoError(t, router.Handle(context.Background(), &DeviceMessage{Kind: MessageKindCommandReply}))
	assert.True(t, reply.called)

	reply.called = false
	require.NoError(t, router.Handle(context.Background(), &DeviceMessage{Kind: MessageKindCommand}))
	assert.False(t, property.called || event.called || reply.called)

	require.NoError(t, router.Handle(context.Background(), nil))
}

func TestMessageRouterPropagatesError(t *testing.T) {
	boom := errors.New("boom")
	router := NewMessageRouter(map[MessageKind]MessageHandler{
		MessageKindProperty: &stubHandler{err: boom},
	})
	err := router.Handle(context.Background(), &DeviceMessage{Kind: MessageKindProperty})
	assert.ErrorIs(t, err, boom)
}

func TestPipelineDirectDispatch(t *testing.T) {
	handler := &stubHandler{}
	pipeline := NewMessagePipeline(MessagePipelineConfig{}, handler)
	require.NoError(t, pipeline.Handle(context.Background(), &DeviceMessage{Kind: MessageKindProperty}))
	assert.True(t, handler.called)
}

type stubSink struct {
	called bool
	err    error
	last   *DeviceMessage
}

func (s *stubSink) Handle(ctx context.Context, msg *DeviceMessage) error {
	s.called = true
	s.last = msg
	return s.err
}

func TestPubMessageHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	sink := &stubSink{}
	InstallEMQXHandler(engine.Group("/api/v1/iot"), &Service{}, sink, "")

	do := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/iot/thingsPubMessage", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec
	}

	rec := do(`{"event":"client.connected"}`)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.False(t, sink.called)

	rec = do(`{"event":"message.publish","topic":"iot/device_001/up","client_attrs":{}}`)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.False(t, sink.called)

	rec = do(`{"event":"message.publish","topic":"iot/device_001/up","client_attrs":{"device_key":"device_001","thing_id":"thing-1"},"payload":"{\"type\":\"property\"}"}`)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.True(t, sink.called)
	require.NotNil(t, sink.last)
	assert.Equal(t, MessageKindProperty, sink.last.Kind)
	assert.Equal(t, "mqtt", sink.last.Source)

	sink.called = false
	rec = do(`{"event":"message.publish","topic":"iot/device_001/up","client_attrs":{"device_key":"device_001"},"payload":"{\"type\":\"command\"}"}`)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.False(t, sink.called)

	sink.called = false
	sink.err = errors.New("boom")
	rec = do(`{"event":"message.publish","topic":"iot/device_001/up","client_attrs":{"device_key":"device_001"},"payload":"{\"type\":\"property\"}"}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBuildDeviceMessageFromEventGateway(t *testing.T) {
	event := &EmqxPubEventMessage{
		Event: EmqxEventMessagePublish,
		Topic: "data/gw-1/v1/meter-1",
		ClientAttrs: map[string]string{
			ClientAttrsGatewayType: "modbusTcp",
			ClientAttrsGatewayID:   "gw-1",
		},
		Payload: `{"type":"property","payload":[]}`,
	}

	msg, ok := BuildDeviceMessageFromEvent(event)
	require.True(t, ok)
	assert.Empty(t, msg.DeviceKey)
	assert.Empty(t, msg.ThingModelKey)
	assert.Equal(t, "modbusTcp", msg.GatewayType)
	assert.Equal(t, "gw-1", msg.GatewayID)
	assert.Equal(t, "data/gw-1/v1/meter-1", msg.Topic)
}
