package hostlinkcmode

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

type parsedAddress struct {
	unit         string
	area         string
	address      uint16
	length       uint16
	isBit        bool
	stringLength uint16
	stringHigh   bool
	readCommand  string
	writeCommand string
}

type areaCommands struct {
	read  string
	write string
	bit   bool
}

var hostlinkAreas = []struct {
	prefix string
	area   areaCommands
}{
	{"CIO", areaCommands{read: "RR", write: "WR"}},
	{"LR", areaCommands{read: "RL", write: "WL"}},
	{"HR", areaCommands{read: "RH", write: "WH"}},
	{"TD", areaCommands{read: "RC", write: "WC"}},
	{"TS", areaCommands{read: "RG", write: "WG", bit: true}},
	{"A", areaCommands{read: "RJ", write: "WJ"}},
	{"D", areaCommands{read: "RD", write: "WD"}},
}

func parseAddress(value, dataType string) (parsedAddress, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	separator := strings.Index(value, "!")
	if separator <= 0 {
		return parsedAddress{}, fmt.Errorf("hostlink address requires station!area")
	}
	unit := value[:separator]
	if len(unit) == 1 {
		unit = "0" + unit
	}
	if len(unit) != 2 {
		return parsedAddress{}, fmt.Errorf("invalid hostlink station %q", value[:separator])
	}
	rest := value[separator+1:]
	var area areaCommands
	areaName := ""
	found := false
	for _, item := range hostlinkAreas {
		if strings.HasPrefix(rest, item.prefix) {
			area = item.area
			areaName = item.prefix
			rest = rest[len(item.prefix):]
			found = true
			break
		}
	}
	if !found {
		return parsedAddress{}, fmt.Errorf("unsupported hostlink area in %q", value)
	}
	dot := strings.Index(rest, ".")
	addressPart := rest
	suffix := ""
	if dot >= 0 {
		addressPart = rest[:dot]
		suffix = rest[dot+1:]
	}
	if addressPart == "" || len(addressPart) > 4 {
		return parsedAddress{}, fmt.Errorf("invalid hostlink address %q", value)
	}
	address, err := strconv.ParseUint(addressPart, 10, 16)
	if err != nil {
		return parsedAddress{}, fmt.Errorf("invalid hostlink address %q", value)
	}
	result := parsedAddress{
		unit:         unit,
		area:         areaName,
		address:      uint16(address),
		readCommand:  area.read,
		writeCommand: area.write,
		isBit:        area.bit,
	}
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "string", "byte_array":
		if suffix == "" {
			return parsedAddress{}, fmt.Errorf("hostlink string address requires length")
		}
		lengthPart := suffix
		high := true
		if strings.HasSuffix(suffix, "H") {
			lengthPart = suffix[:len(suffix)-1]
		} else if strings.HasSuffix(suffix, "L") {
			lengthPart = suffix[:len(suffix)-1]
			high = false
		} else {
			return parsedAddress{}, fmt.Errorf("hostlink string address requires H or L direction")
		}
		length, err := strconv.ParseUint(lengthPart, 10, 16)
		if err != nil || length == 0 || length > 58 {
			return parsedAddress{}, fmt.Errorf("invalid hostlink string length %q", suffix)
		}
		if area.bit {
			return parsedAddress{}, fmt.Errorf("hostlink bit area does not support strings")
		}
		result.stringLength = uint16(length)
		result.stringHigh = high
		result.length = uint16((length + 1) / 2)
		return result, nil
	case "bit", "bool":
		if !area.bit {
			return parsedAddress{}, fmt.Errorf("hostlink bit type requires TS area")
		}
		if suffix != "" {
			return parsedAddress{}, fmt.Errorf("hostlink TS address does not use a bit suffix")
		}
		result.length = 1
		return result, nil
	}
	if suffix != "" {
		return parsedAddress{}, fmt.Errorf("hostlink suffix is only valid for string addresses")
	}
	if area.bit {
		return parsedAddress{}, fmt.Errorf("hostlink TS area only supports bit types")
	}
	result.length = dataTypeWords(dataType)
	if result.length == 0 {
		return parsedAddress{}, fmt.Errorf("unsupported hostlink data type %q", dataType)
	}
	return result, nil
}

func dataTypeWords(dataType string) uint16 {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "int16", "uint16", "word":
		return 1
	case "int32", "uint32", "float32", "dword":
		return 2
	case "int64", "uint64", "float64", "lword":
		return 4
	default:
		return 0
	}
}

func buildReadRequest(address parsedAddress) []byte {
	body := "@" + address.unit + address.readCommand +
		fmt.Sprintf("%04d", address.address) +
		fmt.Sprintf("%04d", address.length)
	return finishFrame(body)
}

func buildWriteRequest(address parsedAddress, data string) []byte {
	body := "@" + address.unit + address.writeCommand +
		fmt.Sprintf("%04d", address.address) + data
	return finishFrame(body)
}

func finishFrame(body string) []byte {
	fcs := 0
	for _, char := range []byte(body) {
		fcs ^= int(char)
	}
	return []byte(fmt.Sprintf("%s%02X*\r", body, fcs))
}

type response struct {
	command string
	data    string
}

func parseResponse(raw []byte, address parsedAddress) (response, error) {
	text := strings.TrimRight(string(raw), "\r\n")
	star := strings.Index(text, "*")
	if len(text) < 9 || text[0] != '@' || star < 0 {
		return response{}, fmt.Errorf("invalid hostlink response")
	}
	body := text[1:star]
	if len(body) < 6 {
		return response{}, fmt.Errorf("hostlink response is too short")
	}
	unit := body[0:2]
	command := body[2:4]
	endCode := body[4:6]
	if unit != address.unit {
		return response{}, fmt.Errorf("hostlink station mismatch: got %s", unit)
	}
	if command != address.readCommand && command != address.writeCommand {
		return response{}, fmt.Errorf("hostlink command mismatch: got %s", command)
	}
	if endCode != "00" {
		return response{}, hostlinkEndCodeError(endCode)
	}
	return response{command: command, data: body[6:]}, nil
}

func hostlinkEndCodeError(code string) error {
	deviceErr := &collection.DeviceError{Code: code}
	if code == "16" {
		deviceErr.MessageID = "error.hostlink_end_code_16"
		return deviceErr
	}
	deviceErr.MessageID = "error.hostlink_end_code"
	deviceErr.Args = []interface{}{code}
	return deviceErr
}

func decodeResponse(response response, address parsedAddress, dataType string) (any, error) {
	if address.isBit {
		return decodeBitValue(response.data, 0, dataType)
	}
	raw, err := decodeHexWords(response.data, int(address.length))
	if err != nil {
		return nil, err
	}
	return decodeWords(raw, address, dataType)
}

func decodeBitValue(data string, offset int, dataType string) (any, error) {
	if offset < 0 || offset >= len(data) {
		return nil, fmt.Errorf("hostlink bit response is too short")
	}
	value := int(data[offset] - '0')
	if strings.EqualFold(dataType, "bool") {
		return value != 0, nil
	}
	return value, nil
}

func decodeWords(raw []byte, address parsedAddress, dataType string) (any, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "string":
		length := int(address.stringLength)
		if length > len(raw) {
			length = len(raw)
		}
		if !address.stringHigh {
			raw = swapPairs(raw)
		}
		return strings.TrimRight(string(raw[:length]), "\x00"), nil
	case "byte_array":
		length := int(address.stringLength)
		if length > len(raw) {
			length = len(raw)
		}
		if !address.stringHigh {
			raw = swapPairs(raw)
		}
		values := make([]int, 0, length)
		for _, item := range raw[:length] {
			values = append(values, int(item))
		}
		return values, nil
	case "int16":
		return int16(parseU16(raw)), nil
	case "uint16", "word":
		return parseU16(raw), nil
	case "int32":
		return int32(parseU32(raw)), nil
	case "uint32", "dword":
		return parseU32(raw), nil
	case "int64":
		return int64(parseU64(raw)), nil
	case "uint64", "lword":
		return parseU64(raw), nil
	case "float32":
		return math.Float32frombits(parseU32(raw)), nil
	case "float64":
		return math.Float64frombits(parseU64(raw)), nil
	default:
		return nil, fmt.Errorf("unsupported hostlink data type %q", dataType)
	}
}

func encodeResponseData(value any, address parsedAddress, dataType string) (string, error) {
	if address.isBit {
		switch v := value.(type) {
		case bool:
			if v {
				return "1", nil
			}
			return "0", nil
		case int:
			if v == 1 {
				return "1", nil
			}
			if v == 0 {
				return "0", nil
			}
		}
		return "", fmt.Errorf("value is not bit 0/1")
	}
	var raw []byte
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "int16":
		v, ok := value.(int16)
		if !ok {
			return "", fmt.Errorf("value is not int16")
		}
		raw = u16(uint16(v))
	case "uint16":
		v, ok := value.(uint16)
		if !ok {
			return "", fmt.Errorf("value is not uint16")
		}
		raw = u16(v)
	case "int32":
		v, ok := value.(int32)
		if !ok {
			return "", fmt.Errorf("value is not int32")
		}
		raw = u32(uint32(v))
	case "float32":
		v, ok := value.(float32)
		if !ok {
			return "", fmt.Errorf("value is not float32")
		}
		raw = u32(math.Float32bits(v))
	case "int64":
		v, ok := value.(int64)
		if !ok {
			return "", fmt.Errorf("value is not int64")
		}
		raw = u64(uint64(v))
	case "float64":
		v, ok := value.(float64)
		if !ok {
			return "", fmt.Errorf("value is not float64")
		}
		raw = u64(math.Float64bits(v))
	case "string":
		v, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("value is not string")
		}
		raw = make([]byte, int(address.length)*2)
		copy(raw, v)
		if !address.stringHigh {
			raw = swapPairs(raw)
		}
	default:
		return "", fmt.Errorf("data type %q is not writable", dataType)
	}
	var builder strings.Builder
	for i := 0; i+1 < len(raw); i += 2 {
		builder.WriteString(fmt.Sprintf("%02X%02X", raw[i], raw[i+1]))
	}
	return builder.String(), nil
}

func decodeHexWords(value string, count int) ([]byte, error) {
	if len(value) < count*4 {
		return nil, fmt.Errorf("hostlink response data is too short")
	}
	out := make([]byte, 0, count*2)
	for i := 0; i < count; i++ {
		word := value[i*4 : i*4+4]
		parsed, err := strconv.ParseUint(word, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid hostlink data %q", word)
		}
		out = append(out, byte(parsed>>8), byte(parsed))
	}
	return out, nil
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
