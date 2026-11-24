package bacnet

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

func readRequest(objectType uint16, instance uint32, property uint32, invoke uint8) []byte {
	apdu := []byte{0x00, 0x05, invoke, 12}
	apdu = appendContext(apdu, 0, encodeObjectID(objectType, instance))
	apdu = appendContext(apdu, 1, uintBytes(uint64(property)))
	return apdu
}

func writeRequest(objectType uint16, instance uint32, property uint32, invoke uint8, value any, dataType string) ([]byte, error) {
	tag, err := encodePointValue(objectType, property, value, dataType)
	if err != nil {
		return nil, err
	}
	apdu := []byte{0x00, 0x05, invoke, 15}
	apdu = appendContext(apdu, 0, encodeObjectID(objectType, instance))
	apdu = appendContext(apdu, 1, uintBytes(uint64(property)))
	apdu = append(apdu, 0x3E)
	apdu = append(apdu, tag...)
	apdu = append(apdu, 0x3F)
	return apdu, nil
}

func encodeObjectID(objectType uint16, instance uint32) []byte {
	value := uint32(objectType&0x3FF)<<22 | instance&0x3FFFFF
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, value)
	return data
}

func appendContext(data []byte, number byte, value []byte) []byte {
	return append(data, appendTag(number, true, value)...)
}

func appendApplication(data []byte, number byte, value []byte) []byte {
	return append(data, appendTag(number, false, value)...)
}

func appendTag(number byte, context bool, value []byte) []byte {
	header := number << 4
	if context {
		header |= 0x08
	}
	length := len(value)
	switch {
	case length <= 4:
		header |= byte(length)
		data := []byte{header}
		return append(data, value...)
	case length <= 253:
		header |= 5
		data := []byte{header, byte(length)}
		return append(data, value...)
	default:
		header |= 5
		data := []byte{header, 254, byte(length >> 8), byte(length)}
		return append(data, value...)
	}
}

func uintBytes(value uint64) []byte {
	if value <= 0xFF {
		return []byte{byte(value)}
	}
	if value <= 0xFFFF {
		return []byte{byte(value >> 8), byte(value)}
	}
	if value <= 0xFFFFFF {
		return []byte{byte(value >> 16), byte(value >> 8), byte(value)}
	}
	return []byte{byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value)}
}

func encodePointValue(objectType uint16, property uint32, value any, dataType string) ([]byte, error) {
	if property == propertyPresentValue {
		switch objectType {
		case 0, 1, 2:
			number, ok := numericFloat(value)
			if !ok {
				return nil, fmt.Errorf("BACnet analog Present_Value requires FLOAT")
			}
			data := make([]byte, 4)
			binary.BigEndian.PutUint32(data, math.Float32bits(float32(number)))
			return appendApplication(nil, 4, data), nil
		case 3, 4, 5:
			number, ok := numericUint(value)
			if !ok {
				boolean, boolOK := value.(bool)
				if !boolOK {
					return nil, fmt.Errorf("BACnet binary Present_Value requires BOOL or BIT")
				}
				if boolean {
					number = 1
				}
				ok = true
			}
			if !ok || number > 1 {
				return nil, fmt.Errorf("BACnet binary Present_Value must be 0 or 1")
			}
			return appendApplication(nil, 9, []byte{byte(number)}), nil
		case 13, 14, 19, 23:
			number, ok := numericUint(value)
			if !ok {
				return nil, fmt.Errorf("BACnet multi-state Present_Value requires an unsigned value")
			}
			return appendApplication(nil, 2, uintBytes(number)), nil
		}
	}
	return encodeApplicationValue(value, dataType)
}

func encodeApplicationValue(value any, dataType string) ([]byte, error) {
	switch strings.ToLower(dataType) {
	case "bool":
		boolean, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("BACnet BOOL value is %T", value)
		}
		if boolean {
			return []byte{0x11}, nil
		}
		return []byte{0x10}, nil
	case "string":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("BACnet STRING value is %T", value)
		}
		return appendApplication(nil, 7, append([]byte{0}, []byte(text)...)), nil
	case "float", "float32":
		number, ok := numericFloat(value)
		if !ok {
			return nil, fmt.Errorf("BACnet FLOAT value is %T", value)
		}
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, math.Float32bits(float32(number)))
		return appendApplication(nil, 4, data), nil
	case "double", "float64":
		number, ok := numericFloat(value)
		if !ok {
			return nil, fmt.Errorf("BACnet DOUBLE value is %T", value)
		}
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, math.Float64bits(number))
		return appendApplication(nil, 5, data), nil
	case "int8", "int16", "int32", "int64":
		number, ok := numericInt(value)
		if !ok {
			return nil, fmt.Errorf("BACnet signed value is %T", value)
		}
		return appendApplication(nil, 3, signedBytes(number)), nil
	case "uint8", "uint16", "uint32", "uint64":
		number, ok := numericUint(value)
		if !ok {
			return nil, fmt.Errorf("BACnet unsigned value is %T", value)
		}
		return appendApplication(nil, 2, uintBytes(number)), nil
	default:
		return nil, fmt.Errorf("unsupported BACnet data type %q", dataType)
	}
}

func numericFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func numericInt(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	default:
		return 0, false
	}
}

func numericUint(value any) (uint64, bool) {
	switch typed := value.(type) {
	case int:
		return uint64(typed), typed >= 0
	case uint8:
		return uint64(typed), true
	case uint16:
		return uint64(typed), true
	case uint32:
		return uint64(typed), true
	case uint64:
		return typed, true
	default:
		return 0, false
	}
}

func signedBytes(value int64) []byte {
	if value >= -128 && value <= 127 {
		return []byte{byte(value)}
	}
	if value >= -32768 && value <= 32767 {
		return []byte{byte(value >> 8), byte(value)}
	}
	if value >= -2147483648 && value <= 2147483647 {
		return []byte{byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value)}
	}
	return []byte{byte(uint64(value) >> 56), byte(uint64(value) >> 48), byte(uint64(value) >> 40), byte(uint64(value) >> 32), byte(uint64(value) >> 24), byte(uint64(value) >> 16), byte(uint64(value) >> 8), byte(value)}
}

func decodeApplicationValue(data []byte) (any, int, error) {
	if len(data) == 0 {
		return nil, 0, fmt.Errorf("empty BACnet application value")
	}
	tag := data[0]
	number := tag >> 4
	length := int(tag & 0x07)
	index := 1
	if length == 5 {
		if len(data) < 2 {
			return nil, 0, fmt.Errorf("truncated BACnet application length")
		}
		length = int(data[1])
		index = 2
		if length == 254 {
			if len(data) < 4 {
				return nil, 0, fmt.Errorf("truncated BACnet extended length")
			}
			length = int(data[2])<<8 | int(data[3])
			index = 4
		}
	}
	if number == 1 {
		return length != 0, 1, nil
	}
	if len(data) < index+length {
		return nil, 0, fmt.Errorf("truncated BACnet application value")
	}
	payload := data[index : index+length]
	next := index + length
	switch number {
	case 0:
		return nil, next, nil
	case 2:
		return unsignedValue(payload), next, nil
	case 3:
		return signedValue(payload), next, nil
	case 4:
		if len(payload) != 4 {
			return nil, 0, fmt.Errorf("invalid BACnet REAL length")
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(payload))), next, nil
	case 5:
		if len(payload) != 8 {
			return nil, 0, fmt.Errorf("invalid BACnet DOUBLE length")
		}
		return math.Float64frombits(binary.BigEndian.Uint64(payload)), next, nil
	case 6:
		return fmt.Sprintf("%X", payload), next, nil
	case 7:
		text := payload
		if len(text) > 0 {
			text = text[1:]
		}
		return strings.TrimRight(string(text), "\x00"), next, nil
	case 8:
		if len(payload) == 0 {
			return "", next, nil
		}
		unused := int(payload[0])
		bits := payload[1:]
		if len(bits) == 1 && unused == 7 {
			return bits[0]&1 != 0, next, nil
		}
		return fmt.Sprintf("%X", bits), next, nil
	case 9:
		return unsignedValue(payload), next, nil
	case 10:
		if len(payload) != 4 {
			return nil, 0, fmt.Errorf("invalid BACnet DATE length")
		}
		return fmt.Sprintf("%04d-%02d-%02d", 1900+int(payload[0]), int(payload[1]), int(payload[2])), next, nil
	case 11:
		if len(payload) != 4 {
			return nil, 0, fmt.Errorf("invalid BACnet TIME length")
		}
		return fmt.Sprintf("%02d:%02d:%02d.%02d", payload[0], payload[1], payload[2], payload[3]), next, nil
	case 12:
		if len(payload) != 4 {
			return nil, 0, fmt.Errorf("invalid BACnet object id length")
		}
		value := binary.BigEndian.Uint32(payload)
		return fmt.Sprintf("%d:%d", value>>22, value&0x3FFFFF), next, nil
	default:
		return fmt.Sprintf("%X", payload), next, nil
	}
}

func unsignedValue(data []byte) uint64 {
	var value uint64
	for _, item := range data {
		value = value<<8 | uint64(item)
	}
	return value
}

func signedValue(data []byte) int64 {
	if len(data) == 0 {
		return 0
	}
	value := int64(int8(data[0]))
	for _, item := range data[1:] {
		value = value<<8 | int64(item)
	}
	return value
}
