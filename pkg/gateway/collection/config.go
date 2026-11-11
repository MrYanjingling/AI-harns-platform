package collection

import "encoding/json"

// ConfigRedactor optionally masks sensitive protocol config fields.
type ConfigRedactor interface {
	RedactConfig(json.RawMessage) json.RawMessage
}
