package snmp

import (
	"fmt"
	"strconv"
	"strings"
)

type Address struct {
	Community string
	OID       string
	Encoded   []byte
}

func ParseAddress(value string) (Address, error) {
	value = strings.TrimSpace(value)
	separator := strings.IndexByte(value, '|')
	if separator <= 0 {
		return Address{}, fmt.Errorf("SNMP address must be community|oid")
	}
	community := value[:separator]
	oidText := strings.TrimSpace(value[separator+1:])
	if len(community) == 0 || len(community) > 64 || strings.ContainsAny(community, "\r\n") {
		return Address{}, fmt.Errorf("invalid SNMP community")
	}
	parts := strings.Split(oidText, ".")
	if len(parts) < 2 || len(parts) > 32 {
		return Address{}, fmt.Errorf("SNMP OID must contain 2..32 segments")
	}
	numbers := make([]uint32, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil {
			return Address{}, fmt.Errorf("invalid SNMP OID segment %q", part)
		}
		numbers = append(numbers, uint32(number))
	}
	if numbers[0] > 2 || (numbers[0] < 2 && numbers[1] > 39) {
		return Address{}, fmt.Errorf("invalid SNMP OID prefix")
	}
	encoded := make([]byte, 0, len(numbers)*2)
	encoded = appendBase128(encoded, uint64(numbers[0])*40+uint64(numbers[1]))
	for _, number := range numbers[2:] {
		encoded = appendBase128(encoded, uint64(number))
	}
	return Address{Community: community, OID: oidText, Encoded: encoded}, nil
}

func appendBase128(data []byte, value uint64) []byte {
	if value == 0 {
		return append(data, 0)
	}
	parts := make([]byte, 0, 5)
	for value > 0 {
		parts = append(parts, byte(value&0x7f))
		value >>= 7
	}
	for index := len(parts) - 1; index >= 0; index-- {
		item := parts[index]
		if index != 0 {
			item |= 0x80
		}
		data = append(data, item)
	}
	return data
}

func decodeOID(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty SNMP OID")
	}
	values := make([]uint64, 0, 8)
	var value uint64
	started := false
	for _, item := range data {
		value = value<<7 | uint64(item&0x7f)
		started = true
		if item&0x80 == 0 {
			values = append(values, value)
			value = 0
			started = false
		}
	}
	if started || len(values) == 0 {
		return "", fmt.Errorf("invalid SNMP OID")
	}
	first := values[0]
	var firstArc, secondArc uint64
	switch {
	case first < 40:
		firstArc, secondArc = 0, first
	case first < 80:
		firstArc, secondArc = 1, first-40
	default:
		firstArc, secondArc = 2, first-80
	}
	parts := []string{strconv.FormatUint(firstArc, 10), strconv.FormatUint(secondArc, 10)}
	for _, item := range values[1:] {
		parts = append(parts, strconv.FormatUint(item, 10))
	}
	return strings.Join(parts, "."), nil
}
