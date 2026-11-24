package fx

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	stx byte = 0x02
	etx byte = 0x03

	cmdRead  byte = '0'
	cmdWrite byte = '1'
	cmdOn    byte = '7'
	cmdOff   byte = '8'

	ack byte = 0x06
)

var hexDigits = "0123456789abcdef"

type fxPoint struct {
	variable  *collection.Variable
	device    string
	number    uint32
	bitDevice bool
	byteAddr  uint16
	bitAddr   uint16
	bit       int
	wordStart uint16
	bytes     int
	wordBit   int
	dataType  string
}

func parseAddress(variable *collection.Variable) (*fxPoint, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	address := strings.ToUpper(strings.TrimSpace(variable.Address))
	prefix, rest := splitDevice(address)
	if prefix == "" {
		return nil, fmt.Errorf("FX address %q has an unknown device", variable.Address)
	}
	numberText := rest
	bit := -1
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		numberText = rest[:dot]
		value, err := strconv.Atoi(rest[dot+1:])
		if err != nil || value < 0 || value > 15 {
			return nil, fmt.Errorf("FX bit %q must be between 0 and 15", rest[dot+1:])
		}
		bit = value
	}
	number, err := strconv.ParseUint(numberText, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("FX device number %q is invalid", numberText)
	}
	if number > 0xFFFF {
		return nil, fmt.Errorf("FX device number %d exceeds 65535", number)
	}
	dataType := strings.ToUpper(strings.TrimSpace(variable.DataType))
	point := &fxPoint{variable: variable, device: prefix, number: uint32(number), bit: bit, wordBit: -1, dataType: dataType}
	switch prefix {
	case "X":
		point.bitDevice = true
		point.bitAddr = uint16(0x400 + number)
		point.byteAddr = uint16(number>>3) + 0x80
	case "Y":
		point.bitDevice = true
		point.bitAddr = uint16(0x500 + number)
		point.byteAddr = uint16(number>>3) + 0xA0
	case "M":
		point.bitDevice = true
		point.bitAddr = uint16(0x800 + number)
		point.byteAddr = uint16(number>>3) + 0x100
	case "S":
		point.bitDevice = true
		point.bitAddr = uint16(number)
		point.byteAddr = uint16(number >> 3)
	case "TS":
		point.bitDevice = true
		point.bitAddr = uint16(0x600 + number)
		point.byteAddr = uint16(number>>3) + 0xC0
	case "CS":
		point.bitDevice = true
		point.bitAddr = uint16(0xE00 + number)
		point.byteAddr = uint16(number>>3) + 0x1C0
	case "TN", "T":
		point.wordStart = uint16(0x800 + number)
		point.bytes = 2
	case "CN", "C":
		point.wordStart = uint16(0xA00 + number)
		point.bytes = 2
	case "D":
		point.wordStart = uint16(number*2) + 0x1000
		point.bytes = 2
	default:
		return nil, fmt.Errorf("FX device %q is not supported", prefix)
	}
	if point.bitDevice {
		if bit >= 0 {
			return nil, fmt.Errorf("FX bit device %q must not include a bit suffix", variable.Address)
		}
		if dataType != "BIT" && dataType != "BOOL" {
			return nil, fmt.Errorf("FX bit device %q only supports BIT/BOOL", variable.Address)
		}
		point.bytes = 1
		point.bit = int(number & 7)
		return point, nil
	}
	if bit >= 0 {
		if dataType != "BIT" && dataType != "BOOL" {
			return nil, fmt.Errorf("FX bit suffix requires BIT/BOOL")
		}
		point.wordBit = bit
		point.bytes = 2
		return point, nil
	}
	bytes, err := bytesForType(dataType)
	if err != nil {
		return nil, err
	}
	point.bytes = bytes
	return point, nil
}

func splitDevice(address string) (string, string) {
	for _, prefix := range []string{"TS", "TN", "CS", "CN", "X", "Y", "M", "S", "T", "C", "D"} {
		if strings.HasPrefix(address, prefix) {
			return prefix, address[len(prefix):]
		}
	}
	return "", address
}

func bytesForType(dataType string) (int, error) {
	switch dataType {
	case "INT16", "UINT16":
		return 2, nil
	case "INT32", "UINT32", "FLOAT32", "FLOAT":
		return 4, nil
	case "INT64", "UINT64", "FLOAT64", "DOUBLE":
		return 8, nil
	default:
		return 0, fmt.Errorf("FX data type %s is not supported", dataType)
	}
}

func buildReadRequest(start uint16, bytes int) []byte {
	frame := make([]byte, 0, 11)
	frame = append(frame, stx, cmdRead)
	frame = append(frame, hex4(start)...)
	frame = append(frame, hex2(byte(bytes))...)
	frame = append(frame, etx)
	frame = append(frame, checksum(frame[1:])...)
	return frame
}

func buildWriteRequest(start uint16, data []byte) []byte {
	frame := make([]byte, 0, 11+2*len(data))
	frame = append(frame, stx, cmdWrite)
	frame = append(frame, hex4(start)...)
	frame = append(frame, hex2(byte(len(data)))...)
	frame = append(frame, bin2hex(data)...)
	frame = append(frame, etx)
	frame = append(frame, checksum(frame[1:])...)
	return frame
}

func buildBitRequest(on bool, address uint16) []byte {
	command := cmdOff
	if on {
		command = cmdOn
	}
	frame := []byte{stx, command}
	frame = append(frame, hex4LE(address)...)
	frame = append(frame, etx)
	frame = append(frame, checksum(frame[1:])...)
	return frame
}

func parseReadResponse(response []byte, bytes int) ([]byte, error) {
	expected := 2*bytes + 4
	if len(response) != expected {
		return nil, fmt.Errorf("FX read response length %d, want %d", len(response), expected)
	}
	if response[0] != stx || response[1+2*bytes] != etx {
		return nil, fmt.Errorf("FX read response framing mismatch")
	}
	if string(checksum(response[1:2+2*bytes])) != string(response[2+2*bytes:]) {
		return nil, fmt.Errorf("FX read response checksum mismatch")
	}
	return hex2bin(response[1 : 1+2*bytes])
}

func parseWriteAck(response []byte) error {
	if len(response) != 1 || response[0] != ack {
		return fmt.Errorf("FX write ACK mismatch")
	}
	return nil
}

func checksum(data []byte) []byte {
	var value byte
	for _, b := range data {
		value += b
	}
	return hex2(value)
}

func hex2(value byte) []byte {
	return []byte{hexDigits[value>>4], hexDigits[value&0x0F]}
}

func hex4(value uint16) []byte {
	return []byte{hexDigits[value>>12&0x0F], hexDigits[value>>8&0x0F], hexDigits[value>>4&0x0F], hexDigits[value&0x0F]}
}

func bin2hex(data []byte) []byte {
	out := make([]byte, 0, 2*len(data))
	for _, b := range data {
		out = append(out, hexDigits[b>>4], hexDigits[b&0x0F])
	}
	return out
}

func hex2bin(data []byte) ([]byte, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("FX hex data has odd length")
	}
	out := make([]byte, len(data)/2)
	for i := range out {
		high, ok := hexValue(data[i*2])
		if !ok {
			return nil, fmt.Errorf("FX hex data is invalid")
		}
		low, ok := hexValue(data[i*2+1])
		if !ok {
			return nil, fmt.Errorf("FX hex data is invalid")
		}
		out[i] = high<<4 | low
	}
	return out, nil
}

func hexValue(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func hex4LE(value uint16) []byte {
	return []byte{hexDigits[value>>4&0x0F], hexDigits[value&0x0F], hexDigits[value>>12&0x0F], hexDigits[value>>8&0x0F]}
}

func decodeWord(data []byte, dataType string) (any, error) {
	switch dataType {
	case "INT16":
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "UINT16":
		return binary.LittleEndian.Uint16(data), nil
	case "INT32":
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "UINT32":
		return binary.LittleEndian.Uint32(data), nil
	case "INT64":
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "UINT64":
		return binary.LittleEndian.Uint64(data), nil
	case "FLOAT32", "FLOAT":
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "FLOAT64", "DOUBLE":
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	default:
		return nil, fmt.Errorf("FX data type %s is not supported", dataType)
	}
}

func encodeWord(item *fxPoint, value any) ([]byte, error) {
	data := make([]byte, item.bytes)
	switch item.dataType {
	case "INT16":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint16(data, uint16(int16(number)))
	case "UINT16":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint16(data, uint16(number))
	case "INT32":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(data, uint32(int32(number)))
	case "UINT32":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(data, uint32(number))
	case "INT64":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint64(data, uint64(number))
	case "UINT64":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint64(data, number)
	case "FLOAT32", "FLOAT":
		number, err := valueAsFloat64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(data, math.Float32bits(float32(number)))
	case "FLOAT64", "DOUBLE":
		number, err := valueAsFloat64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint64(data, math.Float64bits(number))
	default:
		return nil, fmt.Errorf("FX data type %s is not writable", item.dataType)
	}
	return data, nil
}

func valueAsBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case int16:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case uint16:
		return v != 0, nil
	case uint32:
		return v != 0, nil
	case uint64:
		return v != 0, nil
	case float32:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("FX value %v is not a bit", value)
	}
}

func valueAsInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("FX value %v is not numeric", value)
	}
}

func valueAsUint64(value any) (uint64, error) {
	switch v := value.(type) {
	case int16:
		return uint64(v), nil
	case int32:
		return uint64(v), nil
	case int64:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case float32:
		return uint64(v), nil
	case float64:
		return uint64(v), nil
	default:
		return 0, fmt.Errorf("FX value %v is not numeric", value)
	}
}

func valueAsFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("FX value %v is not numeric", value)
	}
}
