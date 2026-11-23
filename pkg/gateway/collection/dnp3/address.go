package dnp3

import (
	"fmt"
	"strings"
)

type pointAddress struct {
	Group     byte
	Variation byte
	Index     uint16
	IsCROB    bool
	Counter   byte
	OnTime    uint32
	OffTime   uint32
}

func parseAddress(value string) (pointAddress, error) {
	text := strings.TrimSpace(value)
	var address pointAddress
	if strings.HasPrefix(strings.ToUpper(text), "CROB.") {
		var counter uint8
		var onTime, offTime uint32
		if n, err := fmt.Sscanf(text, "CROB.%d:%d:%d:%d", &address.Index, &counter, &onTime, &offTime); err != nil || n != 4 {
			return pointAddress{}, fmt.Errorf("invalid DNP3 CROB address %q", value)
		}
		address.IsCROB = true
		address.Group = 12
		address.Variation = 1
		address.Counter = counter
		address.OnTime = onTime
		address.OffTime = offTime
		return address, nil
	}
	var attribute string
	if n, err := fmt.Sscanf(text, "%d.%d.%d.%s", &address.Group, &address.Variation, &address.Index, &attribute); err == nil && n == 4 {
		if attribute != "value" {
			return pointAddress{}, fmt.Errorf("invalid DNP3 address attribute %q", attribute)
		}
		return address, nil
	}
	if n, err := fmt.Sscanf(text, "%d.%d.%d", &address.Group, &address.Variation, &address.Index); err != nil || n != 3 {
		return pointAddress{}, fmt.Errorf("invalid DNP3 address %q", value)
	}
	return address, nil
}

func (a pointAddress) key() uint32 {
	return uint32(a.Group)<<24 | uint32(a.Variation)<<16 | uint32(a.Index)
}
