package drivermgr

import (
	"encoding/json"
	"time"
)

// Driver describes one driver model and its driver source.
type Driver struct {
	ID            string
	DriverType    string
	Model         string
	BuiltinDriver bool
	Schema        json.RawMessage
	PluginName    string
	CreatedTime   time.Time
}
