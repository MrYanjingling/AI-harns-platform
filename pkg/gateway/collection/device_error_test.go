package collection

import (
	"errors"
	"testing"
)

func TestDeviceActionErrorWrapsTransportError(t *testing.T) {
	action := ActionValue{Index: 2, Variable: &Variable{Name: "speed"}}
	err := DeviceActionError(action, errors.New("read timeout"))
	var deviceErr *DeviceError
	if !errors.As(err, &deviceErr) {
		t.Fatalf("expected DeviceError, got %T", err)
	}
	if deviceErr.Field != "actions[2].name" || deviceErr.MessageID != "error.device_action_failed" {
		t.Fatalf("unexpected device error %+v", deviceErr)
	}
	if len(deviceErr.Args) != 1 || deviceErr.Args[0] != "read timeout" {
		t.Fatalf("unexpected args %+v", deviceErr.Args)
	}
}

func TestDeviceActionErrorKeepsExistingField(t *testing.T) {
	action := ActionValue{Index: 1, Variable: &Variable{Name: "speed"}}
	original := &DeviceError{Field: "actions[7].name", MessageID: "error.hostlink_end_code", Code: "16"}
	err := DeviceActionError(action, original)
	var deviceErr *DeviceError
	if !errors.As(err, &deviceErr) || deviceErr != original {
		t.Fatalf("expected original DeviceError, got %+v", err)
	}
	if deviceErr.Field != "actions[7].name" {
		t.Fatalf("field was overwritten: %+v", deviceErr)
	}
}
