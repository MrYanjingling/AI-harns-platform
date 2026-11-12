package s7

import (
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
)

func buildWriteRequest(variable *collection.Variable, value any) ([]byte, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	if strings.EqualFold(variable.DataType, "string") || strings.EqualFold(variable.DataType, "byte_array") {
		return nil, fmt.Errorf("s7 string/byte_array write is not supported")
	}
	parsed, err := parseAddress(variable.Address)
	if err != nil {
		return nil, err
	}
	data, bitLength, err := encodeWriteData(value, variable.DataType)
	if err != nil {
		return nil, err
	}
	parameter := newParameterItem(
		writeParameterTransport(variable.DataType),
		dataRequestLength(variable.DataType, parsed.length),
		uint16(parsed.blockSize),
		parsed.area.code(),
		parsed.address,
		parsed.bit,
	)
	ts := transport(variable.DataType)
	dataItem := []byte{0x00, ts}
	dataItem = append(dataItem, binutil.Uint16ToBytesBigEndian(bitLength)...)
	dataItem = append(dataItem, data...)

	maxBytes := 7 + 10 + 2 + 12 + len(dataItem)
	request := []byte{0x03, 0x00}
	request = append(request, binutil.Uint16ToBytesBigEndian(uint16(maxBytes))...)
	request = append(request, 0x02, 0xf0, 0x80)
	request = append(request, 0x32, 0x01, 0x00, 0x00, 0x00, 0x01)
	request = append(request, binutil.Uint16ToBytesBigEndian(2+12)...)
	request = append(request, binutil.Uint16ToBytesBigEndian(uint16(len(dataItem)))...)
	request = append(request, 0x05, 0x01)
	request = append(request, parameter...)
	request = append(request, dataItem...)
	return request, nil
}

func writeParameterTransport(dataType string) uint8 {
	switch strings.ToLower(dataType) {
	case "bool", "bit":
		return 0x01
	default:
		return 0x02
	}
}

func transport(dataType string) uint8 {
	switch strings.ToLower(dataType) {
	case "bool", "bit":
		return 3
	default:
		return 4
	}
}

func encodeWriteData(value any, dataType string) ([]byte, uint16, error) {
	switch strings.ToLower(dataType) {
	case "bool":
		v, ok := value.(bool)
		if !ok {
			return nil, 0, fmt.Errorf("value is not bool")
		}
		if v {
			return []byte{1}, 1, nil
		}
		return []byte{0}, 1, nil
	case "bit":
		v, ok := value.(int)
		if !ok || (v != 0 && v != 1) {
			return nil, 0, fmt.Errorf("value is not bit 0/1")
		}
		return []byte{byte(v)}, 1, nil
	case "int16":
		v, ok := value.(int16)
		if !ok {
			return nil, 0, fmt.Errorf("value is not int16")
		}
		return binutil.Uint16ToBytesBigEndian(uint16(v)), 16, nil
	case "uint16":
		v, ok := value.(uint16)
		if !ok {
			return nil, 0, fmt.Errorf("value is not uint16")
		}
		return binutil.Uint16ToBytesBigEndian(v), 16, nil
	case "int32":
		v, ok := value.(int32)
		if !ok {
			return nil, 0, fmt.Errorf("value is not int32")
		}
		return binutil.Uint32ToBytesBigEndian(uint32(v)), 32, nil
	case "int64":
		v, ok := value.(int64)
		if !ok {
			return nil, 0, fmt.Errorf("value is not int64")
		}
		return binutil.Uint64ToBytesBigEndian(uint64(v)), 64, nil
	case "float32":
		v, ok := value.(float32)
		if !ok {
			return nil, 0, fmt.Errorf("value is not float32")
		}
		return binutil.Float32ToBytesBigEndian(v), 32, nil
	case "float64":
		v, ok := value.(float64)
		if !ok {
			return nil, 0, fmt.Errorf("value is not float64")
		}
		return binutil.Float64ToBytesBigEndian(v), 64, nil
	default:
		return nil, 0, fmt.Errorf("data type %s is not writable", dataType)
	}
}
