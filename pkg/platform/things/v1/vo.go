package v1

import (
	"encoding/json"
	"time"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/platform/thingmodel/tsl"
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
	Debug          int       `json:"debug"`
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

	ThingModelKey  string       `json:"thingModelKey,omitempty"`
	ThingModelName string       `json:"thingModelName,omitempty"`
	MQTTBroker     string       `json:"mqttBroker,omitempty"`
	UploadFormat   string       `json:"uploadFormat,omitempty"`
	CreatedBy      *UserBriefVo `json:"createdBy,omitempty"`
	UpdatedBy      *UserBriefVo `json:"updatedBy,omitempty"`
}

// UserBriefVo is a compact user reference embedded in thing detail.
type UserBriefVo struct {
	ID       string `json:"id"`
	UserName string `json:"userName"`
	NickName string `json:"nickName"`
}

// ThingShadowVo is the merged device shadow with model property definitions.
type ThingShadowVo struct {
	ThingID    string                               `json:"thingId"`
	Version    int64                                `json:"version"`
	Timestamp  int64                                `json:"timestamp"`
	Properties apis.PageResp[ThingShadowPropertyVo] `json:"properties"`
}

// ThingShadowPropertyVo is one model property merged with reported/desired values.
type ThingShadowPropertyVo struct {
	Identifier   string             `json:"identifier"`
	Name         string             `json:"name"`
	DataType     tsl.DataTypeDetail `json:"dataType"`
	RWFlag       string             `json:"rwFlag"`
	Required     bool               `json:"required"`
	Description  string             `json:"description"`
	Custom       map[string]any     `json:"custom,omitempty"`
	Sort         int                `json:"sort"`
	Value        json.RawMessage    `json:"value"`
	DesiredValue json.RawMessage    `json:"desiredValue"`
}

// ThingTimeSeriesVo is the multi-thing time series query result.
type ThingTimeSeriesVo struct {
	Aggregate string                    `json:"aggregate"`
	Interval  string                    `json:"interval,omitempty"`
	Series    []ThingTimeSeriesSeriesVo `json:"series"`
	Errors    []ThingTimeSeriesErrorVo  `json:"errors,omitempty"`
}

// ThingTimeSeriesSeriesVo is one thing's time series or aggregate values.
type ThingTimeSeriesSeriesVo struct {
	ThingID  string                   `json:"thingId"`
	ModelKey string                   `json:"modelKey"`
	Points   []ThingTimeSeriesPointVo `json:"points,omitempty"`
	Values   map[string]any           `json:"values,omitempty"`
}

// ThingTimeSeriesPointVo is one time series point.
type ThingTimeSeriesPointVo struct {
	Time   int64          `json:"time"`
	Values map[string]any `json:"values"`
}

// ThingTimeSeriesErrorVo is a partial query error.
type ThingTimeSeriesErrorVo struct {
	ThingID  string `json:"thingId"`
	Property string `json:"property,omitempty"`
	Message  string `json:"message"`
}
