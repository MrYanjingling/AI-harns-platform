package dnp3

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

const (
	appFunctionRead          = 1
	appFunctionResponse      = 0x81
	appFunctionUnsolicited   = 0x82
	appFunctionDirectOperate = 5
)

type objectValue struct {
	Group     byte
	Variation byte
	Index     uint16
	Value     any
}

func buildClassPoll(seq byte, variations []byte) []byte {
	app := []byte{0xC0 | (seq & 0x0F), appFunctionRead}
	for _, variation := range variations {
		app = append(app, 60, variation, 0x06)
	}
	return app
}

func buildTimeSync(seq byte, now time.Time) []byte {
	app := []byte{0xC0 | (seq & 0x0F), 2, 50, 1, 0x07, 1}
	millis := uint64(now.UnixMilli())
	for index := 0; index < 6; index++ {
		app = append(app, byte(millis>>(8*index)))
	}
	return app
}

func buildCROBRequest(address pointAddress, value any) ([]byte, error) {
	control, err := crobControlCode(value)
	if err != nil {
		return nil, err
	}
	app := []byte{0xC0, appFunctionDirectOperate, 12, 1, 0x28, 1, 0}
	point := make([]byte, 13)
	binary.LittleEndian.PutUint16(point[0:], address.Index)
	point[2] = control
	point[3] = address.Counter
	binary.LittleEndian.PutUint32(point[4:], address.OnTime)
	binary.LittleEndian.PutUint32(point[8:], address.OffTime)
	return append(app, point...), nil
}

func crobControlCode(value any) (byte, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1, nil
		}
		return 2, nil
	case int:
		if v == 0 || v == 1 {
			if v == 1 {
				return 1, nil
			}
			return 2, nil
		}
	case uint8:
		if v == 0 || v == 1 {
			if v == 1 {
				return 1, nil
			}
			return 2, nil
		}
	case string:
		switch v {
		case "true", "1":
			return 1, nil
		case "false", "0":
			return 2, nil
		}
	}
	return 0, fmt.Errorf("CROB value must be bool or 0/1")
}

func parseAppResponse(app []byte) ([]objectValue, error) {
	if len(app) < 2 {
		return nil, fmt.Errorf("DNP3 application response is too short")
	}
	function := app[1]
	if function != appFunctionResponse && function != appFunctionUnsolicited {
		return nil, fmt.Errorf("unexpected DNP3 application function 0x%02X", function)
	}
	offset := 2
	if function == appFunctionResponse || function == appFunctionUnsolicited {
		if len(app) < 4 {
			return nil, fmt.Errorf("DNP3 application IIN is missing")
		}
		offset += 2
	}
	return parseObjects(app[offset:])
}

func parseObjects(data []byte) ([]objectValue, error) {
	values := make([]objectValue, 0)
	offset := 0
	for offset < len(data) {
		if offset+3 > len(data) {
			return values, fmt.Errorf("DNP3 object header is truncated")
		}
		group := data[offset]
		variation := data[offset+1]
		qualifier := data[offset+2]
		offset += 3
		start, count, prefixBytes, err := parseQualifier(data, &offset, qualifier)
		if err != nil {
			return values, err
		}
		total := count
		if qualifier == 0x06 {
			total = -1
		}
		for index := 0; total < 0 || index < total; index++ {
			if offset >= len(data) {
				if total < 0 {
					break
				}
				return values, fmt.Errorf("DNP3 object data is truncated")
			}
			itemIndex := start + uint16(index)
			if prefixBytes == 1 {
				if offset+1 > len(data) {
					return values, fmt.Errorf("DNP3 8-bit prefix is truncated")
				}
				itemIndex = uint16(data[offset])
				offset++
			} else if prefixBytes == 2 {
				if offset+2 > len(data) {
					return values, fmt.Errorf("DNP3 16-bit prefix is truncated")
				}
				itemIndex = binary.LittleEndian.Uint16(data[offset:])
				offset += 2
			}
			size := objectSize(group, variation)
			if size <= 0 || offset+size > len(data) {
				return values, fmt.Errorf("unsupported DNP3 object %d.%d", group, variation)
			}
			value, err := decodeObject(data[offset:offset+size], group, variation, itemIndex)
			if err != nil {
				return values, err
			}
			values = append(values, objectValue{Group: group, Variation: variation, Index: itemIndex, Value: value})
			offset += size
		}
	}
	return values, nil
}

func parseQualifier(data []byte, offset *int, qualifier byte) (uint16, int, int, error) {
	switch qualifier {
	case 0x00:
		if *offset+2 > len(data) {
			return 0, 0, 0, fmt.Errorf("DNP3 8-bit range is truncated")
		}
		start := uint16(data[*offset])
		stop := uint16(data[*offset+1])
		*offset += 2
		if stop < start {
			return 0, 0, 0, fmt.Errorf("DNP3 range %d..%d is invalid", start, stop)
		}
		return start, int(stop-start) + 1, 0, nil
	case 0x01:
		if *offset+4 > len(data) {
			return 0, 0, 0, fmt.Errorf("DNP3 16-bit range is truncated")
		}
		start := binary.LittleEndian.Uint16(data[*offset:])
		stop := binary.LittleEndian.Uint16(data[*offset+2:])
		*offset += 4
		if stop < start {
			return 0, 0, 0, fmt.Errorf("DNP3 range %d..%d is invalid", start, stop)
		}
		return start, int(stop-start) + 1, 0, nil
	case 0x06:
		return 0, 0, 0, nil
	case 0x07:
		if *offset+1 > len(data) {
			return 0, 0, 0, fmt.Errorf("DNP3 8-bit count is truncated")
		}
		count := int(data[*offset])
		*offset = *offset + 1
		return 0, count, 0, nil
	case 0x08:
		if *offset+2 > len(data) {
			return 0, 0, 0, fmt.Errorf("DNP3 16-bit count is truncated")
		}
		count := int(binary.LittleEndian.Uint16(data[*offset:]))
		*offset += 2
		return 0, count, 0, nil
	case 0x17:
		if *offset+1 > len(data) {
			return 0, 0, 0, fmt.Errorf("DNP3 8-bit indexed count is truncated")
		}
		count := int(data[*offset])
		*offset++
		return 0, count, 1, nil
	case 0x28:
		if *offset+2 > len(data) {
			return 0, 0, 0, fmt.Errorf("DNP3 16-bit indexed count is truncated")
		}
		count := int(binary.LittleEndian.Uint16(data[*offset:]))
		*offset += 2
		return 0, count, 2, nil
	default:
		return 0, 0, 0, fmt.Errorf("unsupported DNP3 qualifier 0x%02X", qualifier)
	}
}

func objectSize(group, variation byte) int {
	switch group {
	case 1, 10:
		switch variation {
		case 1, 2:
			return 1
		case 3, 4:
			return 7
		}
	case 2, 11:
		switch variation {
		case 1:
			return 1
		case 2:
			return 7
		case 3:
			return 3
		}
	case 3:
		switch variation {
		case 1, 2:
			return 1
		}
	case 4:
		switch variation {
		case 1:
			return 1
		case 2:
			return 7
		case 3:
			return 3
		}
	case 12:
		if variation == 1 {
			return 11
		}
	case 20, 21:
		switch variation {
		case 1, 5:
			return 5
		case 2, 6:
			return 3
		}
	case 22, 23:
		switch variation {
		case 1:
			return 5
		case 2:
			return 3
		case 5:
			return 11
		case 6:
			return 9
		}
	case 30, 31:
		switch variation {
		case 1, 5:
			return 5
		case 2:
			return 3
		case 3:
			return 4
		case 4:
			return 2
		case 6:
			return 9
		}
	case 32:
		switch variation {
		case 1, 5:
			return 5
		case 2:
			return 3
		case 3, 7:
			return 11
		case 4, 6:
			return 9
		case 8:
			return 15
		}
	case 40, 41:
		switch variation {
		case 1, 3:
			return 5
		case 2:
			return 3
		case 4:
			return 9
		}
	case 42:
		switch variation {
		case 1, 5:
			return 5
		case 2:
			return 3
		case 3, 7:
			return 11
		case 4, 6:
			return 9
		case 8:
			return 15
		}
	case 50:
		switch variation {
		case 1:
			return 6
		case 4:
			return 11
		}
	case 110, 111:
		return int(variation)
	}
	return 0
}

func decodeObject(data []byte, group, variation byte, index uint16) (any, error) {
	switch group {
	case 1, 10:
		if len(data) < 1 {
			return nil, fmt.Errorf("DNP3 binary data is too short")
		}
		if variation == 2 {
			return data[0]&0x80 != 0, nil
		}
		return data[0]&1 != 0, nil
	case 2, 11:
		if len(data) < 1 {
			return nil, fmt.Errorf("DNP3 binary event data is too short")
		}
		return data[0]&0x80 != 0, nil
	case 3, 4:
		if len(data) < 1 {
			return nil, fmt.Errorf("DNP3 double-bit data is too short")
		}
		return (data[0] >> 6) & 0x03, nil
	case 12:
		if len(data) < 11 {
			return nil, fmt.Errorf("DNP3 CROB data is too short")
		}
		return data[0], nil
	case 20, 21, 22, 23:
		switch variation {
		case 1, 5:
			if len(data) < 5 {
				return nil, fmt.Errorf("DNP3 counter data is too short")
			}
			return binary.LittleEndian.Uint32(data[1:]), nil
		case 2, 6:
			if len(data) < 3 {
				return nil, fmt.Errorf("DNP3 counter data is too short")
			}
			return binary.LittleEndian.Uint16(data[1:]), nil
		}
	case 30, 31, 32, 40, 41, 42:
		switch variation {
		case 1:
			if len(data) < 5 {
				return nil, fmt.Errorf("DNP3 analog data is too short")
			}
			return int32(binary.LittleEndian.Uint32(data[1:])), nil
		case 2:
			if len(data) < 3 {
				return nil, fmt.Errorf("DNP3 analog data is too short")
			}
			return int16(binary.LittleEndian.Uint16(data[1:])), nil
		case 3:
			if group == 32 || group == 42 {
				if len(data) < 5 {
					return nil, fmt.Errorf("DNP3 analog event data is too short")
				}
				return int32(binary.LittleEndian.Uint32(data[1:])), nil
			}
			if len(data) < 4 {
				return nil, fmt.Errorf("DNP3 analog data is too short")
			}
			return int32(binary.LittleEndian.Uint32(data)), nil
		case 4:
			if group == 32 || group == 42 {
				if len(data) < 3 {
					return nil, fmt.Errorf("DNP3 analog event data is too short")
				}
				return int16(binary.LittleEndian.Uint16(data[1:])), nil
			}
			if len(data) < 2 {
				return nil, fmt.Errorf("DNP3 analog data is too short")
			}
			return int16(binary.LittleEndian.Uint16(data)), nil
		case 5, 7:
			if len(data) < 5 {
				return nil, fmt.Errorf("DNP3 analog float data is too short")
			}
			return math.Float32frombits(binary.LittleEndian.Uint32(data[1:])), nil
		case 6, 8:
			if len(data) < 9 {
				return nil, fmt.Errorf("DNP3 analog double data is too short")
			}
			return math.Float64frombits(binary.LittleEndian.Uint64(data[1:])), nil
		}
	case 50:
		if len(data) >= 6 {
			return data[:6], nil
		}
	case 110, 111:
		return string(data), nil
	}
	return nil, fmt.Errorf("unsupported DNP3 object %d.%d", group, variation)
}
