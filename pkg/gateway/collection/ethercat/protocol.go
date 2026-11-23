package ethercat

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var directAddressPattern = regexp.MustCompile(`^(?:0[xX][0-9a-fA-F]+):(?:0[xX][0-9a-fA-F]+|[0-9]+)$`)

type parsedAddress struct {
	Symbol      string
	IndexGroup  uint32
	IndexOffset uint32
	Direct      bool
}

// parseAddress parses an ADS symbol name or a direct index-group/index-offset
// pair. Direct addresses use the form 0x1F020:0x0 or 0x1F020:16.
func parseAddress(value string) (parsedAddress, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return parsedAddress{}, fmt.Errorf("ads address is empty")
	}
	if !directAddressPattern.MatchString(value) {
		if len(value) > 1024 {
			return parsedAddress{}, fmt.Errorf("ads symbol name is too long")
		}
		return parsedAddress{Symbol: value}, nil
	}
	parts := strings.SplitN(value, ":", 2)
	group, err := parseUint32(parts[0])
	if err != nil {
		return parsedAddress{}, fmt.Errorf("invalid ads index group %q", parts[0])
	}
	offsetText := parts[1]
	offset, err := parseUint32(offsetText)
	if err != nil {
		return parsedAddress{}, fmt.Errorf("invalid ads index offset %q", offsetText)
	}
	return parsedAddress{IndexGroup: group, IndexOffset: offset, Direct: true}, nil
}

func parseUint32(value string) (uint32, error) {
	if strings.HasPrefix(strings.ToLower(value), "0x") {
		parsed, err := strconv.ParseUint(value[2:], 16, 32)
		return uint32(parsed), err
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	return uint32(parsed), err
}

func normalizeDataType(dataType string) string {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "boolean":
		return "bool"
	case "bit":
		return "bit"
	case "int8", "sint":
		return "int8"
	case "uint8", "usint", "byte":
		return "uint8"
	case "int16", "int":
		return "int16"
	case "uint16", "uint", "word":
		return "uint16"
	case "int32", "dint":
		return "int32"
	case "uint32", "udint", "dword":
		return "uint32"
	case "int64", "lint":
		return "int64"
	case "uint64", "ulint", "lword":
		return "uint64"
	case "float32", "real":
		return "float32"
	case "float64", "lreal":
		return "float64"
	case "string":
		return "string"
	case "byte_array":
		return "byte_array"
	default:
		return strings.ToLower(strings.TrimSpace(dataType))
	}
}

func typeSize(dataType string, stringLength int) (int, error) {
	if stringLength <= 0 {
		stringLength = DefaultStringLength
	}
	switch normalizeDataType(dataType) {
	case "bool", "bit", "int8", "uint8":
		return 1, nil
	case "int16", "uint16":
		return 2, nil
	case "int32", "uint32", "float32":
		return 4, nil
	case "int64", "uint64", "float64":
		return 8, nil
	case "string", "byte_array":
		return stringLength, nil
	default:
		return 0, fmt.Errorf("unsupported ads data type %q", dataType)
	}
}

func decodeValue(data []byte, dataType string) (any, error) {
	switch normalizeDataType(dataType) {
	case "bool":
		if len(data) < 1 {
			return nil, fmt.Errorf("ads bool response is too short")
		}
		return data[0] != 0, nil
	case "bit":
		if len(data) < 1 {
			return nil, fmt.Errorf("ads bit response is too short")
		}
		return int(data[0] & 1), nil
	case "int8":
		if len(data) < 1 {
			return nil, fmt.Errorf("ads int8 response is too short")
		}
		return int8(data[0]), nil
	case "uint8":
		if len(data) < 1 {
			return nil, fmt.Errorf("ads uint8 response is too short")
		}
		return data[0], nil
	case "int16":
		if len(data) < 2 {
			return nil, fmt.Errorf("ads int16 response is too short")
		}
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "uint16":
		if len(data) < 2 {
			return nil, fmt.Errorf("ads uint16 response is too short")
		}
		return binary.LittleEndian.Uint16(data), nil
	case "int32":
		if len(data) < 4 {
			return nil, fmt.Errorf("ads int32 response is too short")
		}
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "uint32":
		if len(data) < 4 {
			return nil, fmt.Errorf("ads uint32 response is too short")
		}
		return binary.LittleEndian.Uint32(data), nil
	case "int64":
		if len(data) < 8 {
			return nil, fmt.Errorf("ads int64 response is too short")
		}
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "uint64":
		if len(data) < 8 {
			return nil, fmt.Errorf("ads uint64 response is too short")
		}
		return binary.LittleEndian.Uint64(data), nil
	case "float32":
		if len(data) < 4 {
			return nil, fmt.Errorf("ads float32 response is too short")
		}
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "float64":
		if len(data) < 8 {
			return nil, fmt.Errorf("ads float64 response is too short")
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	case "string":
		if len(data) == 0 {
			return nil, fmt.Errorf("ads string response is empty")
		}
		end := len(data)
		for index, item := range data {
			if item == 0 {
				end = index
				break
			}
		}
		return string(data[:end]), nil
	case "byte_array":
		values := make([]int, 0, len(data))
		for _, item := range data {
			values = append(values, int(item))
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported ads data type %q", dataType)
	}
}

func encodeValue(value any, dataType string, length int) ([]byte, error) {
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
	case "float32":
		v, ok := value.(float32)
		if !ok {
			return nil, fmt.Errorf("value is not float32")
		}
		return le32(math.Float32bits(v)), nil
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
		if length <= 0 {
			length = DefaultStringLength
		}
		if len(v) > length {
			return nil, fmt.Errorf("string value is longer than %d bytes", length)
		}
		data := make([]byte, length)
		copy(data, v)
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

func parseAMSNetID(value string) ([6]byte, error) {
	var result [6]byte
	value = strings.TrimSpace(value)
	if value == "" {
		return result, fmt.Errorf("ads net id is empty")
	}
	parts := strings.Split(value, ".")
	if len(parts) != 6 {
		return result, fmt.Errorf("ads net id %q must contain 6 octets", value)
	}
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 || number > 255 {
			return result, fmt.Errorf("invalid ads net id octet %q", part)
		}
		result[index] = byte(number)
	}
	return result, nil
}

func formatAMSNetID(netID [6]byte) string {
	return fmt.Sprintf("%d.%d.%d.%d.%d.%d", netID[0], netID[1], netID[2], netID[3], netID[4], netID[5])
}

// resolveTargetNetID returns the configured target AMS Net ID, or derives
// IP.1.1 from an IPv4 location.
func resolveTargetNetID(location, configured string) ([6]byte, error) {
	if strings.TrimSpace(configured) != "" {
		return parseAMSNetID(configured)
	}
	host := strings.TrimSpace(location)
	if parsed := net.ParseIP(host); parsed != nil && parsed.To4() != nil {
		return parseAMSNetID(host + ".1.1")
	}
	return [6]byte{}, fmt.Errorf("ads net id is required when location is not an IPv4 address")
}

func resolveLocalNetID(configured string) ([6]byte, error) {
	if strings.TrimSpace(configured) == "" {
		configured = "127.0.0.1.1.1"
	}
	return parseAMSNetID(configured)
}
