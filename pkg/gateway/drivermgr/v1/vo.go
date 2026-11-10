package v1

import (
	"encoding/json"
	"time"
)

// DriverVo is one driver metadata record.
type DriverVo struct {
	ID            string          `json:"id"`
	DriverType    string          `json:"driverType"`
	Model         string          `json:"model"`
	BuiltinDriver bool            `json:"builtinDriver"`
	Schema        json.RawMessage `json:"schema"`
	PluginName    string          `json:"pluginName"`
	CreatedTime   time.Time       `json:"createdTime"`
}

// DriverListVo is the unpaginated driver list.
type DriverListVo struct {
	Drivers []*DriverVo `json:"drivers"`
}

// DriverGroupVo groups protocol models by driver type.
type DriverGroupVo struct {
	DriverType string      `json:"driverType"`
	Models     []*DriverVo `json:"models"`
}

// DriverGroupListVo is the grouped driver list.
type DriverGroupListVo struct {
	Groups []*DriverGroupVo `json:"groups"`
}

// DriverSchemaVo returns one driver schema for frontend use.
type DriverSchemaVo struct {
	DriverType  string          `json:"driverType"`
	DriverModel string          `json:"driverModel"`
	Schema      json.RawMessage `json:"schema"`
}
