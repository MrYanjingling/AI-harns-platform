package v1

import (
	"harns-platform/pkg/apis"
	"harns-platform/pkg/platform/thingmodel/tsl"
)

// CreateThingModelReq creates a thing model with optional initial properties.
type CreateThingModelReq struct {
	ThingModelKey string         `json:"thingModelKey" binding:"omitempty,max=64"`
	Name          string         `json:"name" binding:"required,max=64"`
	Description   string         `json:"description" binding:"omitempty,max=255"`
	Properties    []tsl.Property `json:"properties" binding:"omitempty"`
	Features      []tsl.Feature  `json:"features" binding:"omitempty"`
}

// UpdateThingModelReq updates the mutable thing model header fields.
type UpdateThingModelReq struct {
	Name        *string `json:"name" binding:"omitempty,max=64"`
	Description *string `json:"description" binding:"omitempty,max=255"`
}

// ImportThingModelReq is the JSON document accepted by the import endpoint.
// ThingModelKey is optional but must match the current key when present.
// Properties is required; an empty array clears all properties.
type ImportThingModelReq struct {
	ThingModelKey string          `json:"thingModelKey"`
	Name          string          `json:"name"`
	Description   *string         `json:"description"`
	Properties    *[]tsl.Property `json:"properties"`
	Features      *[]tsl.Feature  `json:"features"`
}

// ThingModelQuery is the paged query of thing models.
type ThingModelQuery struct {
	apis.PageQuery
	Keyword string `json:"keyword" form:"keyword" binding:"omitempty,max=64"`
}
