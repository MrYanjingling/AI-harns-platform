package cjt188

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection/meterlink"
)

func buildReadFrame(address meterAddress, control byte, withSER bool) []byte {
	frame := make([]byte, 0, 13+len(address.DI))
	frame = append(frame, 0x68, address.Type)
	frame = append(frame, meterlink.EncodeBCDAddress(address.Meter, 7)...)
	length := len(address.DI)
	if withSER {
		length++
	}
	frame = append(frame, control, byte(length))
	frame = append(frame, address.DI...)
	if withSER {
		frame = append(frame, 0x00)
	}
	frame = append(frame, meterlink.ChecksumSum8(frame), 0x16)
	return frame
}

func parseReadFrame(frame []byte, address meterAddress, control byte, withSER bool) ([]byte, error) {
	if len(frame) < 13 || frame[0] != 0x68 || frame[len(frame)-1] != 0x16 {
		return nil, fmt.Errorf("invalid CJ/T 188 frame")
	}
	length := int(frame[10])
	if len(frame) != 13+length {
		return nil, fmt.Errorf("invalid CJ/T 188 length")
	}
	if frame[9] != control {
		if frame[9]&0x40 != 0 {
			return nil, fmt.Errorf("meter returned an error control code 0x%02X", frame[9])
		}
		return nil, fmt.Errorf("unexpected control code 0x%02X", frame[9])
	}
	if frame[1] != address.Type {
		return nil, fmt.Errorf("meter type mismatch")
	}
	want := meterlink.EncodeBCDAddress(address.Meter, 7)
	if string(frame[2:9]) != string(want) {
		return nil, fmt.Errorf("meter address mismatch")
	}
	if meterlink.ChecksumSum8(frame[:11+length]) != frame[11+length] {
		return nil, fmt.Errorf("CJ/T 188 checksum mismatch")
	}
	data := frame[11 : 11+length]
	offset := len(address.DI)
	if withSER {
		offset++
	}
	if len(data) < offset {
		return nil, fmt.Errorf("response data is too short")
	}
	if string(data[:len(address.DI)]) != string(address.DI) {
		return nil, fmt.Errorf("response DI mismatch")
	}
	return data[offset:], nil
}

type diSpec struct {
	length   int
	decimals int
}

var diSpecs = map[uint16]diSpec{
	0x901F: {length: 4, decimals: 2},
	0x911F: {length: 4, decimals: 2},
	0x9020: {length: 4, decimals: 2},
	0x8105: {length: 4, decimals: 2},
	0xD100: {length: 4, decimals: 2},
}

func decodeDI(address meterAddress, payload []byte, dataType string) (any, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty meter data")
	}
	di := uint16(address.DI[0]) | uint16(address.DI[1])<<8
	spec := diSpecs[di]
	decoded := payload
	if spec.length > 0 {
		if spec.length > len(decoded) {
			spec.length = len(decoded)
		}
		decoded = decoded[:spec.length]
	}
	digits := meterlink.DecodeBCD(decoded)
	raw, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid BCD value")
	}
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "float32", "float64":
		return float64(raw) / math.Pow10(spec.decimals), nil
	case "int16", "int32", "int64":
		return int64(raw), nil
	case "string":
		return digits, nil
	default:
		return raw, nil
	}
}
