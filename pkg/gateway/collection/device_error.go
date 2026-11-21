package collection

import "fmt"

// DeviceError carries a protocol-level device failure for API presentation.
type DeviceError struct {
	Field     string
	Code      string
	MessageID string
	Args      []interface{}
	Err       error
}

func (e *DeviceError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return fmt.Sprintf("device returned error code %s", e.Code)
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "device returned an error"
}

func (e *DeviceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
