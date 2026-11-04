package timeseries

import "time"

// Database names.
const (
	DefaultDataDatabase = "harns_timeseries"
	DefaultLogDatabase  = "harns_timeseries_log"
)

// Table name prefixes.
const (
	PropertyTablePrefix      = "device_property_"
	PropertyLogTablePrefix   = "device_property_log_"
	EventTablePrefix         = "device_event_"
	EventLogTablePrefix      = "device_event_log_"
	CommandRecordTablePrefix = "device_command_record_"
)

// GatewayPropertyLogMeasurement is the fixed measurement for gateway reports.
const GatewayPropertyLogMeasurement = "gateway_property_log"

// Tag keys.
const (
	TagModelID       = "model_id"
	TagDeviceID      = "device_id"
	TagDeviceKey     = "device_key"
	TagEventIdentify = "event_identify"
	TagGatewayType   = "gateway_type"
	TagGatewayID     = "gateway_id"
)

// Field keys.
const (
	FieldSuccess             = "success"
	FieldPayload             = "payload"
	FieldErrorMsg            = "error_msg"
	FieldData                = "data"
	FieldCommandPayload      = "command_payload"
	FieldCommandReplyPayload = "command_reply_payload"
	FieldMessageID           = "message_id"
)

// PropertyTable returns the property measurement name for a model.
func PropertyTable(modelKey string) string { return PropertyTablePrefix + modelKey }

// PropertyLogTable returns the property log measurement name for a model.
func PropertyLogTable(modelKey string) string { return PropertyLogTablePrefix + modelKey }

// EventTable returns the event measurement name for a model.
func EventTable(modelKey string) string { return EventTablePrefix + modelKey }

// EventLogTable returns the event log measurement name for a model.
func EventLogTable(modelKey string) string { return EventLogTablePrefix + modelKey }

// CommandRecordTable returns the command record measurement name for a model.
func CommandRecordTable(modelKey string) string { return CommandRecordTablePrefix + modelKey }

// Point is one InfluxDB line protocol point.
type Point struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]any
	Time        time.Time
}

// PropertyPoint is one property sample.
type PropertyPoint struct {
	ModelID   string
	DeviceID  string
	DeviceKey string
	Time      time.Time
	Values    map[string]any
}

// PropertyLog is one property report request log.
type PropertyLog struct {
	ModelID   string
	DeviceID  string
	DeviceKey string
	Time      time.Time
	Success   bool
	Payload   string
	ErrorMsg  string
}

// GatewayPropertyLog is one gateway property report request log.
type GatewayPropertyLog struct {
	GatewayType string
	GatewayID   string
	Time        time.Time
	Success     bool
	Payload     string
	ErrorMsg    string
}

// EventPoint is one event sample.
type EventPoint struct {
	ModelID       string
	DeviceID      string
	DeviceKey     string
	EventIdentify string
	Time          time.Time
	Data          string
}

// EventLog is one event report request log.
type EventLog struct {
	ModelID   string
	DeviceID  string
	DeviceKey string
	Time      time.Time
	Success   bool
	Payload   string
	ErrorMsg  string
}

// CommandRecord is one command request record.
type CommandRecord struct {
	ModelID             string
	DeviceID            string
	DeviceKey           string
	Time                time.Time
	Success             bool
	CommandPayload      string
	CommandReplyPayload string
	MessageID           string
}
