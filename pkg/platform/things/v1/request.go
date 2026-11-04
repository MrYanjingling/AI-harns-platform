package v1

import (
	"encoding/json"

	"harns-platform/pkg/apis"
)

// CreateThingsReq creates up to MaxBatchSize things under one thing model.
type CreateThingsReq struct {
	ThingModelID string            `json:"thingModelId" binding:"required,max=32"`
	Things       []CreateThingItem `json:"things" binding:"required,max=100"`
}

// CreateThingItem is a single thing in a batch create request.
type CreateThingItem struct {
	DeviceName string                     `json:"deviceName" binding:"required,max=64"`
	DeviceKey  string                     `json:"deviceKey" binding:"omitempty,max=64"`
	DeviceDesc string                     `json:"deviceDesc" binding:"omitempty,max=255"`
	Features   map[string]json.RawMessage `json:"features" binding:"omitempty"`
	Image      string                     `json:"image" binding:"omitempty,max=255"`
}

// UpdateThingReq updates the mutable fields of a thing.
type UpdateThingReq struct {
	DeviceName   *string                     `json:"deviceName" binding:"omitempty,max=64"`
	DeviceDesc   *string                     `json:"deviceDesc" binding:"omitempty,max=255"`
	MQTTClientID *string                     `json:"mqttClientId" binding:"omitempty,max=64"`
	MQTTUsername *string                     `json:"mqttUsername" binding:"omitempty,max=64"`
	MQTTPassword *string                     `json:"mqttPassword" binding:"omitempty,max=64"`
	Enable       *int                        `json:"enable" binding:"omitempty,oneof=0 1"`
	Features     *map[string]json.RawMessage `json:"features"`
	Image        *string                     `json:"image" binding:"omitempty,max=255"`
}

// ThingQuery is the paged query of things.
type ThingQuery struct {
	apis.PageQuery
	Keyword string `json:"keyword" form:"keyword" binding:"omitempty,max=64"`
}
