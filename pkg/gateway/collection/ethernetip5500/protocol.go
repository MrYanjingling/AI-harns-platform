package ethernetip5500

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type cipSegment struct {
	symbol   string
	index    byte
	hasIndex bool
}

type parsedAddress struct {
	segments []cipSegment
	bit      uint8
	bitSet   bool
	typeCode uint16
}

// parseAddress parses an EtherNet/IP tag path. Supported forms:
//
//	Tag
//	Tag.SubTag
//	Tag[3]
//	Program:MainProgram.Tag
func parseAddress(value, dataType string) (parsedAddress, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return parsedAddress{}, fmt.Errorf("ethernet/ip address is empty")
	}
	if strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return parsedAddress{}, fmt.Errorf("invalid ethernet/ip address %q", value)
	}
	parts := strings.Split(value, ".")
	segments := make([]cipSegment, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return parsedAddress{}, fmt.Errorf("invalid ethernet/ip address %q", value)
		}
		segment := cipSegment{}
		if index := strings.Index(part, "["); index >= 0 {
			if !strings.HasSuffix(part, "]") || index == 0 {
				return parsedAddress{}, fmt.Errorf("invalid ethernet/ip array segment %q", part)
			}
			segment.symbol = part[:index]
			indexValue, err := strconv.Atoi(part[index+1 : len(part)-1])
			if err != nil || indexValue < 0 || indexValue > 255 {
				return parsedAddress{}, fmt.Errorf("invalid ethernet/ip array index %q", part)
			}
			segment.index = byte(indexValue)
			segment.hasIndex = true
		} else {
			segment.symbol = part
		}
		segments = append(segments, segment)
	}
	result := parsedAddress{segments: segments}
	dataType = normalizeDataType(dataType)
	code, err := cipTypeCode(dataType)
	if err != nil {
		return parsedAddress{}, err
	}
	result.typeCode = code
	return result, nil
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func normalizeDataType(dataType string) string {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "sint":
		return "int8"
	case "usint", "byte":
		return "uint8"
	case "int":
		return "int16"
	case "uint":
		return "uint16"
	case "dint":
		return "int32"
	case "udint":
		return "uint32"
	case "lint":
		return "int64"
	case "ulint":
		return "uint64"
	case "real":
		return "float32"
	case "lreal":
		return "float64"
	default:
		return strings.ToLower(strings.TrimSpace(dataType))
	}
}

func cipTypeCode(dataType string) (uint16, error) {
	switch normalizeDataType(dataType) {
	case "bool", "bit":
		return 0xC1, nil
	case "int8":
		return 0xC2, nil
	case "int16":
		return 0xC3, nil
	case "int32":
		return 0xC4, nil
	case "int64":
		return 0xC5, nil
	case "uint8":
		return 0xC6, nil
	case "uint16":
		return 0xC7, nil
	case "uint32":
		return 0xC8, nil
	case "uint64":
		return 0xC9, nil
	case "float32":
		return 0xCA, nil
	case "float64":
		return 0xCB, nil
	case "string":
		return 0xD0, nil
	case "word":
		return 0xD2, nil
	case "dword":
		return 0xD3, nil
	case "lword":
		return 0xD4, nil
	case "byte_array":
		return 0xC2, nil
	default:
		return 0, fmt.Errorf("unsupported ethernet/ip data type %q", dataType)
	}
}

func buildPath(address parsedAddress) ([]byte, error) {
	path := make([]byte, 0, len(address.segments)*4)
	for _, segment := range address.segments {
		if segment.symbol == "" {
			return nil, fmt.Errorf("empty ethernet/ip symbol")
		}
		if len(segment.symbol) > 255 {
			return nil, fmt.Errorf("ethernet/ip symbol %q is too long", segment.symbol)
		}
		path = append(path, 0x91, byte(len(segment.symbol)))
		path = append(path, []byte(segment.symbol)...)
		if len(segment.symbol)%2 != 0 {
			path = append(path, 0)
		}
		if segment.hasIndex {
			path = append(path, 0x28, segment.index)
		}
	}
	if len(path)%2 != 0 {
		path = append(path, 0)
	}
	return path, nil
}

func readRequest(address parsedAddress) ([]byte, error) {
	path, err := buildPath(address)
	if err != nil {
		return nil, err
	}
	request := []byte{0x4C, byte(len(path) / 2)}
	request = append(request, path...)
	request = append(request, 1, 0)
	return request, nil
}

func writeRequest(address parsedAddress, data []byte) ([]byte, error) {
	path, err := buildPath(address)
	if err != nil {
		return nil, err
	}
	request := []byte{0x4D, byte(len(path) / 2)}
	request = append(request, path...)
	request = append(request, byte(address.typeCode), byte(address.typeCode>>8), 1, 0)
	request = append(request, data...)
	return request, nil
}

func parseReadResponse(response []byte) ([]byte, uint16, error) {
	if len(response) < 4 {
		return nil, 0, fmt.Errorf("ethernet/ip response is too short")
	}
	if response[0] != 0xCC {
		if response[2] != 0 {
			return nil, 0, fmt.Errorf("ethernet/ip read status %d", response[2])
		}
		return nil, 0, fmt.Errorf("ethernet/ip read response service is %02X", response[0])
	}
	if response[2] != 0 {
		return nil, 0, fmt.Errorf("ethernet/ip read status %d", response[2])
	}
	additional := int(response[3])
	offset := 4 + additional
	if offset+2 > len(response) {
		return nil, 0, fmt.Errorf("ethernet/ip read response is too short")
	}
	dataType := binary.LittleEndian.Uint16(response[offset:])
	rest := response[offset+2:]
	data := rest
	if size := cipTypeSize(dataType); size > 0 && len(rest) >= 2 {
		count := int(binary.LittleEndian.Uint16(rest))
		if count > 0 && len(rest) == 2+count*size {
			data = rest[2:]
		}
	}
	return data, dataType, nil
}

func cipTypeSize(dataType uint16) int {
	switch dataType {
	case 0xC1, 0xC2, 0xC6:
		return 1
	case 0xC3, 0xC7, 0xD2:
		return 2
	case 0xC4, 0xC8, 0xCA, 0xD3:
		return 4
	case 0xC5, 0xC9, 0xCB, 0xD4:
		return 8
	default:
		return 0
	}
}

func parseWriteResponse(response []byte) error {
	if len(response) < 4 {
		return fmt.Errorf("ethernet/ip write response is too short")
	}
	if response[0] != 0xCD {
		if response[2] != 0 {
			return fmt.Errorf("ethernet/ip write status %d", response[2])
		}
		return fmt.Errorf("ethernet/ip write response service is %02X", response[0])
	}
	if response[2] != 0 {
		return fmt.Errorf("ethernet/ip write status %d", response[2])
	}
	return nil
}

func decodeValue(data []byte, dataType string, address parsedAddress) (any, error) {
	switch normalizeDataType(dataType) {
	case "bool":
		if len(data) == 0 {
			return nil, fmt.Errorf("ethernet/ip bool response is empty")
		}
		if address.bitSet {
			return bitValue(data, address.bit), nil
		}
		return data[0] != 0, nil
	case "bit":
		if len(data) == 0 {
			return nil, fmt.Errorf("ethernet/ip bit response is empty")
		}
		if address.bitSet {
			if bitValue(data, address.bit) {
				return 1, nil
			}
			return 0, nil
		}
		return int(data[0] & 1), nil
	case "int8":
		if len(data) < 1 {
			return nil, fmt.Errorf("ethernet/ip int8 response is too short")
		}
		return int8(data[0]), nil
	case "uint8":
		if len(data) < 1 {
			return nil, fmt.Errorf("ethernet/ip uint8 response is too short")
		}
		return data[0], nil
	case "int16":
		if len(data) < 2 {
			return nil, fmt.Errorf("ethernet/ip int16 response is too short")
		}
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "uint16", "word":
		if len(data) < 2 {
			return nil, fmt.Errorf("ethernet/ip uint16 response is too short")
		}
		return binary.LittleEndian.Uint16(data), nil
	case "int32":
		if len(data) < 4 {
			return nil, fmt.Errorf("ethernet/ip int32 response is too short")
		}
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "uint32", "dword":
		if len(data) < 4 {
			return nil, fmt.Errorf("ethernet/ip uint32 response is too short")
		}
		return binary.LittleEndian.Uint32(data), nil
	case "int64":
		if len(data) < 8 {
			return nil, fmt.Errorf("ethernet/ip int64 response is too short")
		}
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "uint64", "lword":
		if len(data) < 8 {
			return nil, fmt.Errorf("ethernet/ip uint64 response is too short")
		}
		return binary.LittleEndian.Uint64(data), nil
	case "float32":
		if len(data) < 4 {
			return nil, fmt.Errorf("ethernet/ip float32 response is too short")
		}
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "float64":
		if len(data) < 8 {
			return nil, fmt.Errorf("ethernet/ip float64 response is too short")
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	case "string":
		if len(data) < 2 {
			return nil, fmt.Errorf("ethernet/ip string response is too short")
		}
		length := int(binary.LittleEndian.Uint16(data))
		if length > len(data)-2 {
			length = len(data) - 2
		}
		return strings.TrimRight(string(data[2:2+length]), "\x00"), nil
	case "byte_array":
		values := make([]int, 0, len(data))
		for _, item := range data {
			values = append(values, int(item))
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported ethernet/ip data type %q", dataType)
	}
}

func bitValue(data []byte, bit uint8) bool {
	switch {
	case len(data) >= 4:
		return (binary.LittleEndian.Uint32(data)>>bit)&1 != 0
	case len(data) >= 2:
		return (binary.LittleEndian.Uint16(data)>>bit)&1 != 0
	default:
		return (data[0]>>bit)&1 != 0
	}
}

func encodeValue(value any, dataType string, address parsedAddress) ([]byte, error) {
	switch normalizeDataType(dataType) {
	case "bool":
		v, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("value is not bool")
		}
		if address.bitSet {
			return nil, fmt.Errorf("bool bit writes require read-modify-write")
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
		if address.bitSet {
			return nil, fmt.Errorf("bit writes require read-modify-write")
		}
		return []byte{byte(v)}, nil
	case "int8":
		v, ok := value.(int8)
		if !ok {
			return nil, fmt.Errorf("value is not int8")
		}
		return []byte{byte(v)}, nil
	case "uint8":
		v, ok := value.(uint8)
		if !ok {
			return nil, fmt.Errorf("value is not uint8")
		}
		return []byte{v}, nil
	case "int16":
		v, ok := value.(int16)
		if !ok {
			return nil, fmt.Errorf("value is not int16")
		}
		return le16(uint16(v)), nil
	case "uint16":
		v, ok := value.(uint16)
		if !ok {
			return nil, fmt.Errorf("value is not uint16")
		}
		return le16(v), nil
	case "int32":
		v, ok := value.(int32)
		if !ok {
			return nil, fmt.Errorf("value is not int32")
		}
		return le32(uint32(v)), nil
	case "uint32":
		v, ok := value.(uint32)
		if !ok {
			return nil, fmt.Errorf("value is not uint32")
		}
		return le32(v), nil
	case "float32":
		v, ok := value.(float32)
		if !ok {
			return nil, fmt.Errorf("value is not float32")
		}
		return le32(math.Float32bits(v)), nil
	case "int64":
		v, ok := value.(int64)
		if !ok {
			return nil, fmt.Errorf("value is not int64")
		}
		return le64(uint64(v)), nil
	case "uint64":
		v, ok := value.(uint64)
		if !ok {
			return nil, fmt.Errorf("value is not uint64")
		}
		return le64(v), nil
	case "float64":
		v, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("value is not float64")
		}
		return le64(math.Float64bits(v)), nil
	case "string":
		v, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("value is not string")
		}
		length := len(v)
		if length%2 != 0 {
			length++
		}
		data := make([]byte, 2+length)
		binary.LittleEndian.PutUint16(data, uint16(length))
		copy(data[2:], v)
		return data, nil
	case "byte_array":
		values, ok := value.([]int)
		if !ok {
			return nil, fmt.Errorf("value is not byte array")
		}
		data := make([]byte, 0, len(values))
		for _, item := range values {
			data = append(data, byte(item))
		}
		return data, nil
	default:
		return nil, fmt.Errorf("data type %q is not writable", dataType)
	}
}

func le16(value uint16) []byte {
	return []byte{byte(value), byte(value >> 8)}
}
func le32(value uint32) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)}
}
func le64(value uint64) []byte {
	return append(le32(uint32(value)), le32(uint32(value>>32))...)
}

const (
	maxBatchTags  = 50
	maxBatchBytes = 0x400
)

func buildMultipleRequest(requests [][]byte) ([]byte, error) {
	count := len(requests)
	if count == 0 {
		return nil, fmt.Errorf("ethernet/ip multiple request is empty")
	}
	if count > 255 {
		return nil, fmt.Errorf("ethernet/ip multiple request has too many services")
	}
	request := []byte{0x0A, 0x02, 0x20, 0x02, 0x24, 0x01, byte(count), byte(count >> 8)}
	offset := (count + 1) * 2
	for _, item := range requests {
		request = append(request, byte(offset), byte(offset>>8))
		offset += len(item)
	}
	for _, item := range requests {
		request = append(request, item...)
	}
	return request, nil
}

func parseMultipleResponse(response []byte) ([][]byte, error) {
	if len(response) < 4 {
		return nil, fmt.Errorf("ethernet/ip multiple response is too short")
	}
	if response[0] != 0x8A {
		return nil, fmt.Errorf("ethernet/ip multiple response service is %02X", response[0])
	}
	if response[2] != 0 {
		return nil, fmt.Errorf("ethernet/ip multiple status %d", response[2])
	}
	offset := 4 + int(response[3])
	if offset+2 > len(response) {
		return nil, fmt.Errorf("ethernet/ip multiple response is too short")
	}
	data := response[offset:]
	count := int(binary.LittleEndian.Uint16(data))
	if count <= 0 || count > 255 {
		return nil, fmt.Errorf("ethernet/ip multiple response count %d", count)
	}
	if len(data) < 2+count*2 {
		return nil, fmt.Errorf("ethernet/ip multiple response offsets are too short")
	}
	items := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		start := int(binary.LittleEndian.Uint16(data[2+i*2:]))
		var end int
		if i+1 < count {
			end = int(binary.LittleEndian.Uint16(data[2+(i+1)*2:]))
		} else {
			end = len(data)
		}
		if start < 0 || end < start || end > len(data) {
			return nil, fmt.Errorf("ethernet/ip multiple response offset is invalid")
		}
		items = append(items, data[start:end])
	}
	return items, nil
}
