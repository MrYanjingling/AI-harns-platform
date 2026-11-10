package v1

import "encoding/json"

// CreateDriverReq creates a custom driver.
type CreateDriverReq struct {
	DriverType string          `json:"driverType"`
	Model      string          `json:"model"`
	Schema     json.RawMessage `json:"schema"`
	PluginName string          `json:"pluginName"`
}

// UpdateDriverReq updates a custom driver.
type UpdateDriverReq struct {
	DriverType string          `json:"driverType"`
	Model      string          `json:"model"`
	Schema     json.RawMessage `json:"schema"`
	PluginName string          `json:"pluginName"`
}

// DriverQuery filters driver records.
type DriverQuery struct {
	DriverType string `form:"driverType"`
	Flat       bool   `form:"flat"`
}

// DriverSchemaQuery queries one driver schema by model.
type DriverSchemaQuery struct {
	DriverModel string `form:"driverModel"`
}
