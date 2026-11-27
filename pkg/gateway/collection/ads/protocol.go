package ads

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// parsedAddress is one ADS direct address: index group, index offset and,
// for STRING tags, the configured character count.
type parsedAddress struct {
	IndexGroup   uint32
	IndexOffset  uint32
	StringLength int
	ReadOnly     bool
	WriteOnly    bool
}

// parseAddress parses ADS index-group/offset addresses:
//
//	group,offset
//	group,offset.length   (STRING only)
//
// Both group and offset accept decimal or 0x-prefixed hexadecimal values. A
// colon is also accepted in place of the comma for compatibility with earlier
// gateway configurations. Symbol names are not supported.
func parseAddress(value, dataType string) (parsedAddress, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return parsedAddress{}, fmt.Errorf("ads address is empty")
	}
	separator := strings.IndexAny(value, ",:")
	if separator < 0 {
		return parsedAddress{}, fmt.Errorf("ads address %q must be group,offset", value)
	}
	groupText := strings.TrimSpace(value[:separator])
	offsetText := strings.TrimSpace(value[separator+1:])
	if groupText == "" || offsetText == "" {
		return parsedAddress{}, fmt.Errorf("ads address %q must be group,offset", value)
	}
	group, err := parseUint32(groupText)
	if err != nil {
		return parsedAddress{}, fmt.Errorf("invalid ads index group %q", groupText)
	}
	offsetPart := offsetText
	stringLength := 0
	if dot := strings.IndexByte(offsetText, '.'); dot >= 0 {
		offsetPart = offsetText[:dot]
		lengthText := offsetText[dot+1:]
		if lengthText == "" {
			return parsedAddress{}, fmt.Errorf("invalid ads string length %q", offsetText)
		}
		stringLength, err = strconv.Atoi(lengthText)
		if err != nil || stringLength < 1 || stringLength > 255 {
			return parsedAddress{}, fmt.Errorf("invalid ads string length %q", offsetText)
		}
	}
	offset, err := parseUint32(offsetPart)
	if err != nil {
		return parsedAddress{}, fmt.Errorf("invalid ads index offset %q", offsetPart)
	}
	dataType = normalizeDataType(dataType)
	if dataType != "string" && stringLength != 0 {
		return parsedAddress{}, fmt.Errorf("ads address %q has a string length but data type is %s", value, dataType)
	}
	if dataType == "string" && stringLength == 0 {
		return parsedAddress{}, fmt.Errorf("ads STRING address %q requires .length", value)
	}
	readOnly, writeOnly, err := validateIndexAddress(group, offset)
	if err != nil {
		return parsedAddress{}, err
	}
	return parsedAddress{
		IndexGroup:   group,
		IndexOffset:  offset,
		StringLength: stringLength,
		ReadOnly:     readOnly,
		WriteOnly:    writeOnly,
	}, nil
}

func parseUint32(value string) (uint32, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "0x") {
		parsed, err := strconv.ParseUint(value[2:], 16, 32)
		return uint32(parsed), err
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	return uint32(parsed), err
}

// validateIndexAddress applies the standard ADS index-group whitelist and offset rules.
func validateIndexAddress(group, offset uint32) (readOnly, writeOnly bool, err error) {
	switch group {
	case 0x4020:
		if offset > 0xFFFF {
			return false, false, fmt.Errorf("ads index offset 0x%X exceeds 0xFFFF for group 0x4020", offset)
		}
	case 0x4021, 0x4030, 0x4040, 0xF005:
	case 0x4025, 0x4035, 0x4045, 0xF025, 0xF035:
		if offset != 0 {
			return false, false, fmt.Errorf("ads index offset must be zero for group 0x%X", group)
		}
		readOnly = true
	case 0xF003:
		if offset != 0 {
			return false, false, fmt.Errorf("ads index offset must be zero for group 0xF003")
		}
	case 0xF006:
		if offset != 0 {
			return false, false, fmt.Errorf("ads index offset must be zero for group 0xF006")
		}
		writeOnly = true
	case 0xF020, 0xF021, 0xF022, 0xF023, 0xF024, 0xF026, 0xF027, 0xF028, 0xF029, 0xF02A,
		0xF02B, 0xF02C, 0xF02D, 0xF02E, 0xF02F, 0xF030, 0xF031, 0xF080, 0xF081, 0xF082:
	default:
		return false, false, fmt.Errorf("ads index group 0x%X is not supported", group)
	}
	return readOnly, writeOnly, nil
}

func normalizeDataType(dataType string) string {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "boolean", "bit":
		return "bool"
	case "int8", "sint":
		return "int8"
	case "uint8", "usint", "byte":
		return "uint8"
	case "int16", "int":
		return "int16"
	case "uint16", "uint":
		return "uint16"
	case "int32", "dint":
		return "int32"
	case "uint32", "udint":
		return "uint32"
	case "int64", "lint":
		return "int64"
	case "uint64", "ulint":
		return "uint64"
	case "float32", "real":
		return "float32"
	case "float64", "lreal":
		return "float64"
	case "string":
		return "string"
	default:
		return strings.ToLower(strings.TrimSpace(dataType))
	}
}

func typeSize(dataType string, address parsedAddress) (int, error) {
	switch normalizeDataType(dataType) {
	case "bool", "int8", "uint8":
		return 1, nil
	case "int16", "uint16":
		return 2, nil
	case "int32", "uint32", "float32":
		return 4, nil
	case "int64", "uint64", "float64":
		return 8, nil
	case "string":
		if address.StringLength < 1 || address.StringLength > 255 {
			return 0, fmt.Errorf("ads STRING address requires a length between 1 and 255")
		}
		return address.StringLength, nil
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
	default:
		return nil, fmt.Errorf("unsupported ads data type %q", dataType)
	}
}

func encodeValue(value any, dataType string, length int) ([]byte, error) {
	switch normalizeDataType(dataType) {
	case "bool":
		v, ok := value.(bool)
		if !ok {
			number, numberOK := value.(int)
			if !numberOK || (number != 0 && number != 1) {
				return nil, fmt.Errorf("value is not bool")
			}
			v = number != 0
		}
		if v {
			return []byte{1}, nil
		}
		return []byte{0}, nil
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
			return nil, fmt.Errorf("string length is not configured")
		}
		if len(v) > length {
			return nil, fmt.Errorf("string value is longer than %d bytes", length)
		}
		data := make([]byte, length)
		copy(data, v)
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
	if isIPv4(host) {
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

func isIPv4(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 || number > 255 {
			return false
		}
	}
	return true
}
