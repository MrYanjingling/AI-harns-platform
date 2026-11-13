package v1

import (
	"encoding/json"
	"fmt"
	"time"

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
	Debug      *int                       `json:"debug" binding:"omitempty,oneof=0 1"`
}

// UpdateThingReq updates the mutable fields of a thing.
type UpdateThingReq struct {
	DeviceName   *string                     `json:"deviceName" binding:"omitempty,max=64"`
	DeviceDesc   *string                     `json:"deviceDesc" binding:"omitempty,max=255"`
	MQTTClientID *string                     `json:"mqttClientId" binding:"omitempty,max=64"`
	MQTTUsername *string                     `json:"mqttUsername" binding:"omitempty,max=64"`
	MQTTPassword *string                     `json:"mqttPassword" binding:"omitempty,max=64"`
	Enable       *int                        `json:"enable" binding:"omitempty,oneof=0 1"`
	Debug        *int                        `json:"debug" binding:"omitempty,oneof=0 1"`
	Features     *map[string]json.RawMessage `json:"features"`
	Image        *string                     `json:"image" binding:"omitempty,max=255"`
}

// ThingQuery is the paged query of things.
type ThingQuery struct {
	apis.PageQuery
	Keyword      string `json:"keyword" form:"keyword" binding:"omitempty,max=64"`
	ThingModelID string `json:"thingModelId" form:"thingModelId" binding:"omitempty,max=32"`
}

// ThingShadowQuery is the paged query of merged shadow properties.
type ThingShadowQuery struct {
	apis.PageQuery
	Keyword string `json:"keyword" form:"keyword" binding:"omitempty,max=64"`
}

// FlexibleTime accepts Unix milliseconds (number) or RFC3339 (string).
type FlexibleTime struct {
	time.Time
}

func (t *FlexibleTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return fmt.Errorf("invalid RFC3339 time %q: %w", value, err)
		}
		t.Time = parsed
		return nil
	}
	var millis int64
	if err := json.Unmarshal(data, &millis); err != nil {
		return fmt.Errorf("invalid Unix millisecond time: %w", err)
	}
	t.Time = time.UnixMilli(millis)
	return nil
}

// ThingTimeSeriesReq queries time series for multiple things.
type ThingTimeSeriesReq struct {
	Things    []ThingTimeSeriesItem `json:"things" binding:"required,min=1,max=50,dive"`
	StartTime FlexibleTime          `json:"startTime" binding:"required"`
	EndTime   FlexibleTime          `json:"endTime" binding:"required"`
	Aggregate string                `json:"aggregate" binding:"omitempty"`
	Interval  string                `json:"interval" binding:"omitempty,max=16"`
	Sort      string                `json:"sort" binding:"omitempty,oneof=asc desc"`
	Limit     int                   `json:"limit" binding:"omitempty,min=1,max=2000"`
}

// ThingTimeSeriesItem requests properties for one thing.
type ThingTimeSeriesItem struct {
	ThingID    string   `json:"thingId" binding:"required,max=32"`
	Properties []string `json:"properties" binding:"required,min=1,max=100"`
}
