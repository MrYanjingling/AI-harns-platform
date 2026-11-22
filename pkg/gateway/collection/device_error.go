package collection

import (
	"errors"
	"fmt"
)

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

// DeviceActionError converts a protocol/transport failure from a write action
// into a DeviceError so the API returns a device response instead of 10017.
func DeviceActionError(action ActionValue, err error) error {
	if err == nil {
		return nil
	}
	field := fmt.Sprintf("actions[%d].name", action.Index)
	var deviceErr *DeviceError
	if errors.As(err, &deviceErr) {
		if deviceErr.Field == "" {
			deviceErr.Field = field
		}
		return deviceErr
	}
	return &DeviceError{
		Field:     field,
		MessageID: "error.device_action_failed",
		Args:      []interface{}{err.Error()},
		Err:       err,
	}
}
