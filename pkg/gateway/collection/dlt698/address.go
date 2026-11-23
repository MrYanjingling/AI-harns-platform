package dlt698

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func parseMeterAddress(value string) ([]byte, error) {
	clean := strings.NewReplacer(" ", "", "-", "", ":", "").Replace(strings.TrimSpace(value))
	if clean == "" {
		return nil, fmt.Errorf("meter address is required")
	}
	if len(clean)%2 != 0 || len(clean) > 16 {
		return nil, fmt.Errorf("meter address must be 1-8 hex bytes")
	}
	raw, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("meter address is invalid")
	}
	return raw, nil
}

func buildAddressArea(meter []byte, client byte) []byte {
	area := make([]byte, 0, len(meter)+2)
	area = append(area, byte(len(meter)-1))
	for index := len(meter) - 1; index >= 0; index-- {
		area = append(area, meter[index])
	}
	area = append(area, client)
	return area
}

func parseOAD(value string) ([4]byte, error) {
	var oad [4]byte
	clean := strings.NewReplacer(" ", "", "-", "", ":", "").Replace(strings.TrimSpace(value))
	if len(clean) != 8 {
		return oad, fmt.Errorf("OAD must be 8 hex characters")
	}
	raw, err := hex.DecodeString(clean)
	if err != nil {
		return oad, fmt.Errorf("OAD is invalid")
	}
	copy(oad[:], raw)
	return oad, nil
}
