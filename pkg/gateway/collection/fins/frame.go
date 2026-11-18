package fins

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

type frameVariable struct {
	Variable *collection.Variable
	Parsed   parsedAddress
	Offset   uint16
}

type readFrame struct {
	Request        []byte
	Area           byte
	Start          uint16
	Bit            uint8
	Length         uint16
	Variables      []*frameVariable
	ResponseLength int
}

func buildReadFrames(variables []*collection.Variable, maxSize int) ([]*readFrame, error) {
	maxWords := maxSize / 2
	if maxWords < 1 {
		maxWords = 1
	}
	type tag struct {
		variable *collection.Variable
		parsed   parsedAddress
	}
	tags := make([]tag, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		parsed, err := parseAddress(variable.Address, variable.DataType, variable.BitsValue())
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		tags = append(tags, tag{variable: variable, parsed: parsed})
	}
	sort.SliceStable(tags, func(i, j int) bool {
		left, right := tags[i].parsed, tags[j].parsed
		if left.areaCode() != right.areaCode() {
			return left.areaCode() < right.areaCode()
		}
		if left.start != right.start {
			return left.start < right.start
		}
		return left.length < right.length
	})
	frames := make([]*readFrame, 0)
	for _, item := range tags {
		parsed := item.parsed
		if parsed.isBit {
			frame := &readFrame{
				Area:           parsed.areaCode(),
				Start:          parsed.start,
				Bit:            parsed.bit,
				Length:         1,
				ResponseLength: 1,
				Variables:      []*frameVariable{{Variable: item.variable, Parsed: parsed, Offset: 0}},
			}
			frame.Request = newReadRequest(frame.Area, frame.Start, frame.Bit, frame.Length)
			frames = append(frames, frame)
			continue
		}
		if len(frames) > 0 {
			last := frames[len(frames)-1]
			lastEnd := uint32(last.Start) + uint32(last.Length)
			newEnd := uint32(parsed.start) + uint32(parsed.length)
			if last.Area == parsed.areaCode() && !last.Variables[0].Parsed.isBit && uint32(parsed.start) <= lastEnd {
				span := newEnd - uint32(last.Start)
				if span <= uint32(maxWords) {
					offset := uint16(uint32(parsed.start) - uint32(last.Start))
					last.Variables = append(last.Variables, &frameVariable{Variable: item.variable, Parsed: parsed, Offset: offset})
					if newEnd > lastEnd {
						last.Length = uint16(newEnd - uint32(last.Start))
						last.ResponseLength = int(last.Length) * 2
					}
					last.Request = newReadRequest(last.Area, last.Start, 0, last.Length)
					continue
				}
			}
		}
		frame := &readFrame{
			Area:           parsed.areaCode(),
			Start:          parsed.start,
			Length:         parsed.length,
			ResponseLength: int(parsed.length) * 2,
			Variables:      []*frameVariable{{Variable: item.variable, Parsed: parsed, Offset: 0}},
		}
		frame.Request = newReadRequest(frame.Area, frame.Start, 0, frame.Length)
		frames = append(frames, frame)
	}
	return frames, nil
}

func newReadRequest(area byte, start uint16, bit uint8, count uint16) []byte {
	request := newRequestHeader(0x0101)
	request = append(request, area)
	request = append(request, byte(start>>8), byte(start))
	request = append(request, bit)
	request = append(request, byte(count>>8), byte(count))
	return request
}

func newWriteRequest(area byte, start uint16, bit uint8, count uint16, data []byte) []byte {
	request := newRequestHeader(0x0102)
	request = append(request, area)
	request = append(request, byte(start>>8), byte(start))
	request = append(request, bit)
	request = append(request, byte(count>>8), byte(count))
	request = append(request, data...)
	return request
}

func newRequestHeader(command uint16) []byte {
	return []byte{
		0x80, 0x00, 0x02, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		byte(command >> 8), byte(command),
	}
}

func validateReadResponse(response []byte, frame *readFrame) ([]byte, error) {
	if len(response) < 14 {
		return nil, fmt.Errorf("fins response is too short")
	}
	if response[10] != 0x01 || response[11] != 0x01 {
		return nil, fmt.Errorf("fins response command mismatch")
	}
	if response[12] != 0 || response[13] != 0 {
		return nil, fmt.Errorf("fins response error %02X%02X", response[12], response[13])
	}
	payload := response[14:]
	if len(payload) < frame.ResponseLength {
		return nil, fmt.Errorf("fins response payload is too short")
	}
	return payload[:frame.ResponseLength], nil
}

func validateWriteResponse(response []byte) error {
	if len(response) < 14 {
		return fmt.Errorf("fins write response is too short")
	}
	if response[10] != 0x01 || response[11] != 0x02 {
		return fmt.Errorf("fins write response command mismatch")
	}
	if response[12] != 0 || response[13] != 0 {
		return fmt.Errorf("fins write response error %02X%02X", response[12], response[13])
	}
	return nil
}

type decodedValue struct {
	Variable *collection.Variable
	Value    any
}

func decodeFrame(frame *readFrame, data []byte) ([]decodedValue, error) {
	values := make([]decodedValue, 0, len(frame.Variables))
	for _, item := range frame.Variables {
		parsed := item.Parsed
		if parsed.isBit {
			if len(data) == 0 {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			value := int(data[0] & 1)
			if strings.EqualFold(item.Variable.DataType, "bool") {
				values = append(values, decodedValue{Variable: item.Variable, Value: collection.ApplyTransform(value != 0, item.Variable)})
			} else {
				values = append(values, decodedValue{Variable: item.Variable, Value: collection.ApplyTransform(value, item.Variable)})
			}
			continue
		}
		base := int(item.Offset) * 2
		length := int(parsed.length) * 2
		if base+length > len(data) {
			return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
		}
		raw := reorder(append([]byte(nil), data[base:base+length]...), parsed.order)
		value, err := decodeWord(raw, item.Variable.DataType, parsed)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", item.Variable.Name, err)
		}
		values = append(values, decodedValue{Variable: item.Variable, Value: collection.ApplyTransform(value, item.Variable)})
	}
	return values, nil
}

func decodeWord(data []byte, dataType string, parsed parsedAddress) (any, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "int16":
		return int16(parseU16(data)), nil
	case "uint16", "word":
		return parseU16(data), nil
	case "int32":
		return int32(parseU32(data)), nil
	case "uint32", "dword":
		return parseU32(data), nil
	case "int64":
		return int64(parseU64(data)), nil
	case "uint64", "lword":
		return parseU64(data), nil
	case "float32":
		return math.Float32frombits(parseU32(data)), nil
	case "float64":
		return math.Float64frombits(parseU64(data)), nil
	case "string":
		length := int(parsed.stringLength)
		if length > len(data) {
			length = len(data)
		}
		raw := data[:length]
		if !parsed.stringHigh {
			raw = swapPairs(raw)
		}
		return strings.TrimRight(string(raw), "\x00"), nil
	case "byte_array":
		length := int(parsed.stringLength)
		if length > len(data) {
			length = len(data)
		}
		raw := data[:length]
		if !parsed.stringHigh {
			raw = swapPairs(raw)
		}
		values := make([]int, 0, len(raw))
		for _, item := range raw {
			values = append(values, int(item))
		}
		return values, nil
	default:
		return nil, fmt.Errorf("data type %q is not supported", dataType)
	}
}

func encodeValue(value any, dataType string, parsed parsedAddress) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool":
		v, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("value is not bool")
		}
		if v {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	case "bit":
		v, ok := value.(int)
		if !ok || (v != 0 && v != 1) {
			return nil, fmt.Errorf("value is not bit 0/1")
		}
		return []byte{byte(v)}, nil
	case "int16":
		v, ok := value.(int16)
		if !ok {
			return nil, fmt.Errorf("value is not int16")
		}
		return reorder(u16(uint16(v)), parsed.order), nil
	case "uint16":
		v, ok := value.(uint16)
		if !ok {
			return nil, fmt.Errorf("value is not uint16")
		}
		return reorder(u16(v), parsed.order), nil
	case "int32":
		v, ok := value.(int32)
		if !ok {
			return nil, fmt.Errorf("value is not int32")
		}
		return reorder(u32(uint32(v)), parsed.order), nil
	case "float32":
		v, ok := value.(float32)
		if !ok {
			return nil, fmt.Errorf("value is not float32")
		}
		return reorder(u32(math.Float32bits(v)), parsed.order), nil
	case "int64":
		v, ok := value.(int64)
		if !ok {
			return nil, fmt.Errorf("value is not int64")
		}
		return reorder(u64(uint64(v)), parsed.order), nil
	case "float64":
		v, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("value is not float64")
		}
		return reorder(u64(math.Float64bits(v)), parsed.order), nil
	case "string":
		v, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("value is not string")
		}
		raw := make([]byte, int(parsed.length)*2)
		copy(raw, v)
		if !parsed.stringHigh {
			raw = swapPairs(raw)
		}
		return reorder(raw, parsed.order), nil
	default:
		return nil, fmt.Errorf("data type %q is not writable", dataType)
	}
}

func reorder(data []byte, order wordOrder) []byte {
	if order.swapBytes {
		data = swapPairs(data)
	}
	if order.reverseWords {
		for i, j := 0, len(data)-2; i < j; i, j = i+2, j-2 {
			data[i], data[j] = data[j], data[i]
			data[i+1], data[j+1] = data[j+1], data[i+1]
		}
	}
	return data
}

func swapPairs(data []byte) []byte {
	out := append([]byte(nil), data...)
	for i := 0; i+1 < len(out); i += 2 {
		out[i], out[i+1] = out[i+1], out[i]
	}
	return out
}

func parseU16(data []byte) uint16 {
	if len(data) < 2 {
		return 0
	}
	return uint16(data[0])<<8 | uint16(data[1])
}

func parseU32(data []byte) uint32 {
	if len(data) < 4 {
		return 0
	}
	return uint32(parseU16(data))<<16 | uint32(parseU16(data[2:]))
}

func parseU64(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	return uint64(parseU32(data))<<32 | uint64(parseU32(data[4:]))
}

func u16(value uint16) []byte { return []byte{byte(value >> 8), byte(value)} }
func u32(value uint32) []byte { return append(u16(uint16(value>>16)), u16(uint16(value))...) }
func u64(value uint64) []byte { return append(u32(uint32(value>>32)), u32(uint32(value))...) }
