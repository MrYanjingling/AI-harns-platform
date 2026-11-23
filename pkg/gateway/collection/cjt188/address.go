package cjt188

import (
	"fmt"
	"strconv"
	"strings"
)

type meterAddress struct {
	Type  byte
	Meter uint64
	DI    []byte // wire order: low byte, high byte
	Key   string
}

func parseAddress(address string) (meterAddress, error) {
	parts := strings.Split(strings.TrimSpace(address), "#")
	if len(parts) != 3 {
		return meterAddress{}, fmt.Errorf("address must be type#meter#DI")
	}
	meterType, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 16, 8)
	if err != nil {
		return meterAddress{}, fmt.Errorf("meter type is invalid")
	}
	meter, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return meterAddress{}, fmt.Errorf("meter number is invalid")
	}
	di, err := strconv.ParseUint(strings.TrimSpace(parts[2]), 16, 16)
	if err != nil {
		return meterAddress{}, fmt.Errorf("DI is invalid")
	}
	return meterAddress{
		Type: byte(meterType), Meter: meter,
		DI:  []byte{byte(di), byte(di >> 8)},
		Key: fmt.Sprintf("%02x#%d#%04x", meterType, meter, di),
	}, nil
}
