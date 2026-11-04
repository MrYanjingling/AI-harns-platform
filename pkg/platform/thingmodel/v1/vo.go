package v1

import (
	"time"

	"harns-platform/pkg/platform/thingmodel/tsl"
)

// ThingModelVo is the thing model summary returned by list/create/update.
type ThingModelVo struct {
	ID            string    `json:"id"`
	Version       int64     `json:"version"`
	ThingModelKey string    `json:"thingModelKey"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	PropertyCount int       `json:"propertyCount"`
	CreatedUser   string    `json:"createdUser"`
	UpdatedUser   string    `json:"updatedUser"`
	CreatedTime   time.Time `json:"createdTime"`
	UpdatedTime   time.Time `json:"updatedTime"`
}

// ThingModelDetailVo is the thing model detail including its properties.
type ThingModelDetailVo struct {
	ThingModelVo
	Properties []PropertyVo  `json:"properties"`
	Features   []tsl.Feature `json:"features"`
}

// PropertyVo is a thing model property returned to clients.
type PropertyVo struct {
	ID      string `json:"id"`
	Version int64  `json:"version"`
	tsl.Property
	CreatedUser string    `json:"createdUser"`
	UpdatedUser string    `json:"updatedUser"`
	CreatedTime time.Time `json:"createdTime"`
	UpdatedTime time.Time `json:"updatedTime"`
}
