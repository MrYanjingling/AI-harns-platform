package v1

import (
	"encoding/json"
	"time"
)

// ThingVo is the thing summary returned by the list endpoint.
type ThingVo struct {
	ID             string    `json:"id"`
	Version        int64     `json:"version"`
	ThingModelID   string    `json:"thingModelId"`
	DeviceName     string    `json:"deviceName"`
	DeviceKey      string    `json:"deviceKey"`
	UpTopic        string    `json:"upTopic"`
	DownTopic      string    `json:"downTopic"`
	DeviceDesc     string    `json:"deviceDesc"`
	MQTTClientID   string    `json:"mqttClientId"`
	MQTTUsername   string    `json:"mqttUsername"`
	Enable         int       `json:"enable"`
	OnlineStatus   int       `json:"onlineStatus"`
	LastActiveTime *int64    `json:"lastActiveTime"`
	CreatedUser    string    `json:"createdUser"`
	UpdatedUser    string    `json:"updatedUser"`
	CreatedTime    time.Time `json:"createdTime"`
	UpdatedTime    time.Time `json:"updatedTime"`
}

// ThingDetailVo is the thing detail including credentials.
type ThingDetailVo struct {
	ThingVo
	DeviceSecret string                     `json:"deviceSecret"`
	Token        string                     `json:"token"`
	MQTTPassword string                     `json:"mqttPassword"`
	Image        string                     `json:"image"`
	Features     map[string]json.RawMessage `json:"features,omitempty"`
}
