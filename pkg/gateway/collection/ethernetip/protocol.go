package ethernetip

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

const (
	cipServiceRead        = 0x4C
	cipServiceWrite       = 0x4D
	cipServiceForwardOpen = 0x54

	cipTypeBool   = 0xC1
	cipTypeSInt   = 0xC2
	cipTypeInt    = 0xC3
	cipTypeDInt   = 0xC4
	cipTypeLInt   = 0xC5
	cipTypeUSInt  = 0xC6
	cipTypeUInt   = 0xC7
	cipTypeUDInt  = 0xC8
	cipTypeULInt  = 0xC9
	cipTypeReal   = 0xCA
	cipTypeLReal  = 0xCB
	cipTypeString = 0xD0
	cipTypeWord   = 0xD2
	cipTypeDWord  = 0xD3
	cipTypeLWord  = 0xD4
)

// parsedAddress is one CIP symbolic path and its declared CIP data type.
type parsedAddress struct {
	segments []string
	typeCode uint16
}

// parseAddress parses a CIP tag path. Neuron supports one ANSI symbol segment
// per dot-separated name: "Tag", "Tag.Member", "Program:MainProgram.Tag".
func parseAddress(value, dataType string) (parsedAddress, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return parsedAddress{}, fmt.Errorf("ethernet/ip address is empty")
	}
	if strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return parsedAddress{}, fmt.Errorf("invalid ethernet/ip address %q", value)
	}
	segments := strings.Split(value, ".")
	pathBytes := 0
	for _, segment := range segments {
		if segment == "" {
			return parsedAddress{}, fmt.Errorf("invalid ethernet/ip address %q", value)
		}
		if len(segment) > 255 {
			return parsedAddress{}, fmt.Errorf("ethernet/ip symbol %q is too long", segment)
		}
		pathBytes += 2 + len(segment)
		if len(segment)%2 != 0 {
			pathBytes++
		}
	}
	if pathBytes/2 > 255 {
		return parsedAddress{}, fmt.Errorf("ethernet/ip address %q is too long", value)
	}
	code, err := cipTypeCode(dataType)
	if err != nil {
		return parsedAddress{}, err
	}
	return parsedAddress{segments: segments, typeCode: code}, nil
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
		return cipTypeBool, nil
	case "int8":
		return cipTypeSInt, nil
	case "uint8":
		return cipTypeUSInt, nil
	case "int16":
		return cipTypeInt, nil
	case "uint16":
		return cipTypeUInt, nil
	case "int32":
		return cipTypeDInt, nil
	case "uint32":
		return cipTypeUDInt, nil
	case "int64":
		return cipTypeLInt, nil
	case "uint64":
		return cipTypeULInt, nil
	case "float32":
		return cipTypeReal, nil
	case "float64":
		return cipTypeLReal, nil
	case "string":
		return cipTypeString, nil
	case "word":
		return cipTypeWord, nil
	case "dword":
		return cipTypeDWord, nil
	case "lword":
		return cipTypeLWord, nil
	default:
		return 0, fmt.Errorf("unsupported ethernet/ip data type %q", dataType)
	}
}

// checkType mirrors Neuron's check_type: a response type is accepted when it
// belongs to the same CIP family as the configured type.
func checkType(ctype, expected uint16) bool {
	switch ctype {
	case cipTypeBool, cipTypeSInt, cipTypeUSInt, 0xD1: // 0xD1 is CIP BYTE
		return expected == cipTypeBool || expected == cipTypeSInt || expected == cipTypeUSInt
	case cipTypeInt, cipTypeUInt, cipTypeWord:
		return expected == cipTypeInt || expected == cipTypeUInt || expected == cipTypeWord
	case cipTypeDInt, cipTypeUDInt, cipTypeReal, cipTypeDWord:
		return expected == cipTypeDInt || expected == cipTypeUDInt || expected == cipTypeReal || expected == cipTypeDWord
	case cipTypeLInt, cipTypeULInt, cipTypeLReal, cipTypeLWord:
		return expected == cipTypeLInt || expected == cipTypeULInt || expected == cipTypeLReal || expected == cipTypeLWord
	case cipTypeString:
		return expected == cipTypeString
	default:
		return false
	}
}

func cipTypeSize(dataType uint16) int {
	switch dataType {
	case cipTypeBool, cipTypeSInt, cipTypeUSInt:
		return 1
	case cipTypeInt, cipTypeUInt, cipTypeWord:
		return 2
	case cipTypeDInt, cipTypeUDInt, cipTypeReal, cipTypeDWord:
		return 4
	case cipTypeLInt, cipTypeULInt, cipTypeLReal, cipTypeLWord:
		return 8
	default:
		return 0
	}
}

func buildPath(address parsedAddress) ([]byte, byte, error) {
	path := make([]byte, 0, len(address.segments)*8)
	for _, segment := range address.segments {
		path = append(path, 0x91, byte(len(segment)))
		path = append(path, []byte(segment)...)
		if len(segment)%2 != 0 {
			path = append(path, 0)
		}
	}
	return path, byte(len(path) / 2), nil
}

func readRequest(sequence uint16, address parsedAddress) ([]byte, error) {
	path, words, err := buildPath(address)
	if err != nil {
		return nil, err
	}
	request := make([]byte, 0, 4+len(path)+2)
	request = append(request, byte(sequence), byte(sequence>>8))
	request = append(request, cipServiceRead, words)
	request = append(request, path...)
	request = append(request, 1, 0)
	return request, nil
}

func writeRequest(sequence uint16, address parsedAddress, data []byte) ([]byte, error) {
	path, words, err := buildPath(address)
	if err != nil {
		return nil, err
	}
	request := make([]byte, 0, 6+len(path)+4+len(data)+2)
	request = append(request, byte(sequence), byte(sequence>>8))
	request = append(request, cipServiceWrite, words)
	request = append(request, path...)
	request = append(request, byte(address.typeCode), byte(address.typeCode>>8), 1, 0)
	if address.typeCode == cipTypeString {
		length := len(data)
		if length%2 != 0 {
			length++
		}
		request = append(request, byte(length), byte(length>>8))
	}
	request = append(request, data...)
	if writeNeedsPad(address.typeCode, len(data)) {
		request = append(request, 0)
	}
	return request, nil
}

// writeNeedsPad reports whether Neuron's CIP write payload appends an
// alignment byte. BOOL, SINT, USINT and odd-length STRING values are padded.
func writeNeedsPad(dataType uint16, length int) bool {
	switch dataType {
	case cipTypeBool, cipTypeSInt, cipTypeUSInt:
		return true
	case cipTypeString:
		return length%2 != 0
	default:
		return false
	}
}

// parseReadResponse validates a connected Read Tag reply and returns the raw
// tag bytes without the CIP type word.
func parseReadResponse(response []byte, address parsedAddress) ([]byte, error) {
	body, err := parseCIPReply(response, cipServiceRead)
	if err != nil {
		return nil, err
	}
	if len(body) < 2 {
		return nil, fmt.Errorf("ethernet/ip read response is too short")
	}
	dataType := binary.LittleEndian.Uint16(body)
	if !checkType(dataType, address.typeCode) {
		return nil, fmt.Errorf("ethernet/ip read data type %02X does not match %02X", dataType, address.typeCode)
	}
	rest := body[2:]
	if dataType == cipTypeString {
		if len(rest) < 2 {
			return nil, fmt.Errorf("ethernet/ip string response is too short")
		}
		return rest, nil
	}
	size := cipTypeSize(dataType)
	if size == 0 {
		return nil, fmt.Errorf("unsupported ethernet/ip response type %02X", dataType)
	}
	if len(rest) < size {
		return nil, fmt.Errorf("ethernet/ip read response is too short")
	}
	return rest[:size], nil
}

// parseWriteResponse validates a connected Write Tag reply.
func parseWriteResponse(response []byte) error {
	_, err := parseCIPReply(response, cipServiceWrite)
	return err
}

// parseCIPReply returns the payload after service, path size and status.
func parseCIPReply(response []byte, service byte) ([]byte, error) {
	if len(response) < 4 {
		return nil, fmt.Errorf("ethernet/ip response is too short")
	}
	if response[0] != service|0x80 {
		return nil, fmt.Errorf("ethernet/ip response service is %02X", response[0])
	}
	if response[1] != 0 {
		return nil, fmt.Errorf("ethernet/ip response path size is %d", response[1])
	}
	status := response[2]
	additionalWords := int(response[3])
	if status != 0 {
		return nil, fmt.Errorf("ethernet/ip CIP status 0x%02X", status)
	}
	offset := 4 + additionalWords*2
	if offset > len(response) {
		return nil, fmt.Errorf("ethernet/ip response status is truncated")
	}
	return response[offset:], nil
}

func decodeValue(data []byte, dataType string) (any, error) {
	switch normalizeDataType(dataType) {
	case "bool":
		if len(data) < 1 {
			return nil, fmt.Errorf("ethernet/ip bool response is empty")
		}
		return data[0] != 0, nil
	case "bit":
		if len(data) < 1 {
			return nil, fmt.Errorf("ethernet/ip bit response is empty")
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
	default:
		return nil, fmt.Errorf("unsupported ethernet/ip data type %q", dataType)
	}
}

func encodeValue(value any, dataType string) ([]byte, error) {
	switch normalizeDataType(dataType) {
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
	case "uint16", "word":
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
	case "uint32", "dword":
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
	case "uint64", "lword":
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
		return []byte(v), nil
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
