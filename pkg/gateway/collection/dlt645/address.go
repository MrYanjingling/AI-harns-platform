package dlt645

import (
	"fmt"
	"strconv"
	"strings"
)

type meterAddress struct {
	Meter uint64
	DI    []byte // wire order: DI0, DI1, ... as sent on the bus
	Key   string
}

func parseAddress(address, model string) (meterAddress, error) {
	parts := strings.Split(strings.TrimSpace(address), "#")
	want := 4
	if model == Model1997 {
		want = 2
	}
	if len(parts) != 2 {
		return meterAddress{}, fmt.Errorf("address must be meter#%s", diPattern(model))
	}
	meter, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || meter > 999999999999 {
		return meterAddress{}, fmt.Errorf("meter number is invalid")
	}
	diParts := strings.Split(strings.TrimSpace(parts[1]), "-")
	if len(diParts) != want {
		return meterAddress{}, fmt.Errorf("DI must have %d bytes", want)
	}
	wire := make([]byte, want)
	for index, item := range diParts {
		value, err := strconv.ParseUint(strings.TrimSpace(item), 16, 8)
		if err != nil {
			return meterAddress{}, fmt.Errorf("DI byte is invalid")
		}
		wire[want-1-index] = byte(value)
	}
	return meterAddress{Meter: meter, DI: wire, Key: fmt.Sprintf("%d#%x", meter, wire)}, nil
}

func diPattern(model string) string {
	if model == Model1997 {
		return "DI1-DI0, e.g. 12345678#00-00"
	}
	return "DI3-DI2-DI1-DI0, e.g. 12345678#00-00-00-00"
}

func (a meterAddress) Value() uint32 {
	var value uint32
	for index := len(a.DI) - 1; index >= 0; index-- {
		value = value<<8 | uint32(a.DI[index])
	}
	return value
}
