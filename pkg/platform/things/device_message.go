package things

import (
	"encoding/json"
	"strings"
)

// Client attribute keys set by the device authentication endpoint.
const (
	ClientAttrsDeviceKey     = "device_key"
	ClientAttrsThingID       = "thing_id"
	ClientAttrsThingModelID  = "thing_model_id"
	ClientAttrsThingModelKey = "thing_model_key"
	ClientAttrsDeviceName    = "device_name"
	ClientAttrsGatewayType   = "gateway_type"
	ClientAttrsGatewayID     = "gateway_id"
)

// MessageKind is the business category carried in the payload type field.
type MessageKind string

const (
	MessageKindProperty     MessageKind = "property"
	MessageKindEvent        MessageKind = "event"
	MessageKindCommandReply MessageKind = "command_reply"
	MessageKindCommand      MessageKind = "command"
)

// DeviceMessage is the platform-level message passed to processors and Kafka.
// It is independent of transport-specific structures such as raw client_attrs.
type DeviceMessage struct {
	Kind          MessageKind `json:"kind"`
	Topic         string      `json:"topic"`
	DeviceKey     string      `json:"deviceKey"`
	ThingID       string      `json:"thingId"`
	ThingModelID  string      `json:"thingModelId"`
	ThingModelKey string      `json:"thingModelKey"`
	DeviceName    string      `json:"deviceName"`
	GatewayType   string      `json:"gatewayType"`
	GatewayID     string      `json:"gatewayId"`
	Payload       string      `json:"payload"`
	Timestamp     int64       `json:"timestamp"`
	MessageID     string      `json:"messageId"`
	Source        string      `json:"source"`
	QoS           int         `json:"qos"`
}

// ResolveInboundKind reads the payload type field. The type is required;
// command is outbound and therefore rejected for inbound messages.
func ResolveInboundKind(payload string) (MessageKind, bool) {
	var meta struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(payload), &meta); err != nil {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(meta.Type)) {
	case string(MessageKindProperty):
		return MessageKindProperty, true
	case string(MessageKindEvent):
		return MessageKindEvent, true
	case string(MessageKindCommandReply):
		return MessageKindCommandReply, true
	case string(MessageKindCommand):
		return MessageKindCommand, false
	default:
		return "", false
	}
}

// BuildDeviceMessageFromEvent converts an EMQX event into a platform message.
func BuildDeviceMessageFromEvent(event *EmqxPubEventMessage) (*DeviceMessage, bool) {
	if event == nil || event.Event != EmqxEventMessagePublish {
		return nil, false
	}
	kind, ok := ResolveInboundKind(event.Payload)
	if !ok {
		return nil, false
	}
	attrs := event.ClientAttrs
	deviceKey := event.DeviceKey()
	gatewayID := strings.TrimSpace(attrs[ClientAttrsGatewayID])
	if deviceKey == "" && gatewayID == "" {
		return nil, false
	}
	return &DeviceMessage{
		Kind:          kind,
		Topic:         event.Topic,
		DeviceKey:     deviceKey,
		ThingID:       attrs[ClientAttrsThingID],
		ThingModelID:  attrs[ClientAttrsThingModelID],
		ThingModelKey: attrs[ClientAttrsThingModelKey],
		DeviceName:    attrs[ClientAttrsDeviceName],
		GatewayType:   strings.TrimSpace(attrs[ClientAttrsGatewayType]),
		GatewayID:     gatewayID,
		Payload:       event.Payload,
		Timestamp:     event.ReceivedAt(),
		MessageID:     event.ID,
		Source:        "mqtt",
		QoS:           event.QoS,
	}, true
}
