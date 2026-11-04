// Package thingmodel implements thing model management: the model header and
// its properties and features, plus JSON import and export.
package thingmodel

import (
	"strings"

	"github.com/oklog/ulid/v2"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/platform/thingmodel/tsl"
)

const (
	// ThingModelKeyPrefix prefixes auto-generated thing model keys.
	ThingModelKeyPrefix = "tm_"
)

// ThingModel is the thing model header. Properties are loaded on demand.
type ThingModel struct {
	apis.Meta
	ThingModelKey string        `json:"thingModelKey"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Features      []tsl.Feature `json:"features,omitempty"`
	Properties    []Property    `json:"properties,omitempty"`
	PropertyCount int           `json:"propertyCount"`
}

// Property is a thing model property entity.
type Property struct {
	apis.Meta
	ThingModelID string `json:"-"`
	tsl.Property
}

// NewID returns a new ULID.
func NewID() string {
	return ulid.Make().String()
}

// NewThingModelKey returns an auto-generated thing model key.
func NewThingModelKey() string {
	return ThingModelKeyPrefix + strings.ToLower(ulid.Make().String())
}
