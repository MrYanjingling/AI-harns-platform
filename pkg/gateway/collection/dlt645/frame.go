package dlt645

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection/meterlink"
)

func buildReadFrame(address meterAddress, control byte) []byte {
	frame := make([]byte, 0, 16+len(address.DI))
	frame = append(frame, 0xFE, 0xFE, 0xFE, 0xFE, 0x68)
	frame = append(frame, meterlink.EncodeBCDAddress(address.Meter, 6)...)
	frame = append(frame, 0x68, control, byte(len(address.DI)))
	frame = append(frame, meterlink.Mask(address.DI)...)
	frame = append(frame, meterlink.ChecksumSum8(frame[4:]), 0x16)
	return frame
}

func parseReadFrame(frame []byte, address meterAddress, control byte) ([]byte, error) {
	if len(frame) < 12 || frame[0] != 0x68 || frame[7] != 0x68 || frame[len(frame)-1] != 0x16 {
		return nil, fmt.Errorf("invalid DL/T 645 frame")
	}
	length := int(frame[9])
	if len(frame) != 12+length {
		return nil, fmt.Errorf("invalid DL/T 645 length")
	}
	if frame[8] != control {
		if frame[8]&0x40 != 0 {
			return nil, fmt.Errorf("meter returned an error control code 0x%02X", frame[8])
		}
		return nil, fmt.Errorf("unexpected control code 0x%02X", frame[8])
	}
	want := meterlink.EncodeBCDAddress(address.Meter, 6)
	if string(frame[1:7]) != string(want) {
		return nil, fmt.Errorf("meter address mismatch")
	}
	if meterlink.ChecksumSum8(frame[:10+length]) != frame[10+length] {
		return nil, fmt.Errorf("DL/T 645 checksum mismatch")
	}
	data := meterlink.Unmask(frame[10 : 10+length])
	if len(data) < len(address.DI) {
		return nil, fmt.Errorf("response data is too short")
	}
	if string(data[:len(address.DI)]) != string(address.DI) {
		return nil, fmt.Errorf("response DI mismatch")
	}
	return data[len(address.DI):], nil
}

var diDecimals2007 = map[uint32]int{
	0x00000000: 2,
	0x00010000: 2,
	0x00020000: 2,
	0x00030000: 2,
	0x02010100: 1,
	0x02020100: 3,
	0x02030000: 4,
	0x02040000: 4,
}

var diDecimals1997 = map[uint32]int{
	0x0000: 2,
	0x0100: 2,
	0x0200: 2,
	0x0300: 2,
	0x9010: 2,
	0xB611: 0,
	0xB612: 0,
	0xB613: 0,
	0xB621: 2,
	0xB622: 2,
	0xB623: 2,
	0xB630: 4,
	0xB631: 4,
	0xB632: 4,
	0xB633: 4,
	0xB640: 2,
	0xB641: 2,
	0xB642: 2,
	0xB643: 2,
	0xB650: 3,
	0xB651: 3,
	0xB652: 3,
	0xB653: 3,
}

func decodeDI(address meterAddress, payload []byte, dataType string, model string) (any, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty meter data")
	}
	decimals := 0
	table := diDecimals2007
	if model == Model1997 {
		table = diDecimals1997
	}
	if value, ok := table[address.Value()]; ok {
		decimals = value
	}
	digits := meterlink.DecodeBCD(payload)
	raw, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid BCD value")
	}
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "float32", "float64":
		return float64(raw) / math.Pow10(decimals), nil
	case "int16", "int32", "int64":
		return int64(raw), nil
	case "string":
		return digits, nil
	default:
		return raw, nil
	}
}
