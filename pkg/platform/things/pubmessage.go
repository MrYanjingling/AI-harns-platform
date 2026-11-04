package things

import "strings"

// EMQX message.publish event name.
const EmqxEventMessagePublish = "message.publish"

// EmqxPubEventMessage mirrors the EMQX message.publish webhook payload.
type EmqxPubEventMessage struct {
	PublishReceivedAt int64                  `json:"publish_received_at"`
	PubProps          map[string]interface{} `json:"pub_props,omitempty"`
	QoS               int                    `json:"qos"`
	Topic             string                 `json:"topic"`
	ClientID          string                 `json:"clientid"`
	ClientAttrs       map[string]string      `json:"client_attrs"`
	PeerHost          string                 `json:"peerhost,omitempty"`
	Payload           string                 `json:"payload"`
	Username          string                 `json:"username,omitempty"`
	Event             string                 `json:"event"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	PeerName          string                 `json:"peername,omitempty"`
	Timestamp         int64                  `json:"timestamp"`
	Node              string                 `json:"node,omitempty"`
	ID                string                 `json:"id,omitempty"`
	Flags             map[string]interface{} `json:"flags,omitempty"`
}

// DeviceKey returns the device key carried in client_attrs.
func (e *EmqxPubEventMessage) DeviceKey() string {
	if e == nil || e.ClientAttrs == nil {
		return ""
	}
	return strings.TrimSpace(e.ClientAttrs[ClientAttrsDeviceKey])
}

// ReceivedAt returns the message receive time in Unix milliseconds.
func (e *EmqxPubEventMessage) ReceivedAt() int64 {
	if e == nil {
		return 0
	}
	if e.PublishReceivedAt > 0 {
		return e.PublishReceivedAt
	}
	return e.Timestamp
}
