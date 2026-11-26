package modbus

import (
	"errors"
	"fmt"
)

// ExceptionError is a protocol-level Modbus exception response.
type ExceptionError struct {
	Function uint8
	Code     uint8
}

func (e *ExceptionError) Error() string {
	if e == nil {
		return ""
	}
	if text, ok := exceptionDescriptions[e.Code]; ok {
		return fmt.Sprintf("modbus exception: %s (function %d, code %d)", text, e.Function, e.Code)
	}
	return fmt.Sprintf("modbus exception: function %d, code %d", e.Function, e.Code)
}

// IsException reports whether err is a Modbus exception response.
func IsException(err error) bool {
	var exception *ExceptionError
	return errors.As(err, &exception)
}

var exceptionDescriptions = map[uint8]string{
	1:  "illegal function",
	2:  "illegal data address",
	3:  "illegal data value",
	4:  "server device failure",
	5:  "acknowledge",
	6:  "server device busy",
	8:  "memory parity error",
	10: "gateway path unavailable",
	11: "gateway target failed to respond",
}
