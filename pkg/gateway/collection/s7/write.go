package s7

import (
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
)

const (
	s7WriteHeaderBytes    = 10
	s7WriteParamHeadBytes = 2
	s7WriteParamItemBytes = 12
	s7WriteMaxItems       = 20
)

type writeItem struct {
	action    collection.ActionValue
	parameter []byte
	dataItem  []byte
	oddData   bool
}

type writeBatch struct {
	request []byte
	actions []collection.ActionValue
}

func buildWriteRequest(variable *collection.Variable, value any) ([]byte, error) {
	item, err := buildWriteItem(collection.ActionValue{Variable: variable, Value: value})
	if err != nil {
		return nil, err
	}
	return encodeWriteBatch([]writeItem{item}).request, nil
}

func buildWriteItem(action collection.ActionValue) (writeItem, error) {
	variable := action.Variable
	if variable == nil {
		return writeItem{}, fmt.Errorf("variable is nil")
	}
	if strings.EqualFold(variable.DataType, "string") || strings.EqualFold(variable.DataType, "byte_array") {
		return writeItem{}, fmt.Errorf("s7 string/byte_array write is not supported")
	}
	parsed, err := parseAddress(variable.Address)
	if err != nil {
		return writeItem{}, err
	}
	data, bitLength, err := encodeWriteData(action.Value, variable.DataType)
	if err != nil {
		return writeItem{}, err
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
	dataItem := make([]byte, 0, 4+len(data))
	dataItem = append(dataItem, 0x00, ts)
	dataItem = append(dataItem, binutil.Uint16ToBytesBigEndian(bitLength)...)
	dataItem = append(dataItem, data...)
	return writeItem{
		action:    action,
		parameter: parameter,
		dataItem:  dataItem,
		oddData:   len(data)%2 != 0,
	}, nil
}

func buildWriteBatches(items []writeItem, pdu uint16) []writeBatch {
	if len(items) == 0 {
		return nil
	}
	if pdu < 240 {
		pdu = DefaultPDUSize
	}
	budget := int(pdu) - s7WriteHeaderBytes - s7WriteParamHeadBytes
	batches := make([]writeBatch, 0, (len(items)+s7WriteMaxItems-1)/s7WriteMaxItems)
	current := make([]writeItem, 0, min(len(items), s7WriteMaxItems))
	used := 0
	flush := func() {
		if len(current) == 0 {
			return
		}
		batches = append(batches, encodeWriteBatch(current))
		current = current[:0]
		used = 0
	}
	for _, item := range items {
		itemCost := s7WriteParamItemBytes + len(item.dataItem)
		prevPad := 0
		if len(current) > 0 && current[len(current)-1].oddData {
			prevPad = 1
		}
		if len(current) > 0 && (used+prevPad+itemCost > budget || len(current) >= s7WriteMaxItems) {
			flush()
			prevPad = 0
		}
		current = append(current, item)
		used += prevPad + itemCost
	}
	flush()
	return batches
}

func encodeWriteBatch(items []writeItem) writeBatch {
	paramLen := s7WriteParamHeadBytes + len(items)*s7WriteParamItemBytes
	dataLen := 0
	for i, item := range items {
		dataLen += len(item.dataItem)
		if i < len(items)-1 && item.oddData {
			dataLen++
		}
	}
	totalBytes := 7 + s7WriteHeaderBytes + paramLen + dataLen
	request := make([]byte, 0, totalBytes)
	request = append(request, 0x03, 0x00)
	request = append(request, binutil.Uint16ToBytesBigEndian(uint16(totalBytes))...)
	request = append(request, 0x02, 0xf0, 0x80)
	request = append(request, 0x32, 0x01, 0x00, 0x00, 0x00, 0x01)
	request = append(request, binutil.Uint16ToBytesBigEndian(uint16(paramLen))...)
	request = append(request, binutil.Uint16ToBytesBigEndian(uint16(dataLen))...)
	request = append(request, 0x05, uint8(len(items)))
	actions := make([]collection.ActionValue, len(items))
	for i, item := range items {
		request = append(request, item.parameter...)
		actions[i] = item.action
	}
	for i, item := range items {
		request = append(request, item.dataItem...)
		if i < len(items)-1 && item.oddData {
			request = append(request, 0x00)
		}
	}
	return writeBatch{request: request, actions: actions}
}

func validateWriteResponse(response []byte, actions []collection.ActionValue) error {
	if len(actions) == 0 {
		return nil
	}
	if len(response) < 21+len(actions) {
		return collection.DeviceActionError(actions[0], fmt.Errorf("s7 write failed: response is too short"))
	}
	if response[17] != 0 || response[18] != 0 {
		return collection.DeviceActionError(actions[0], fmt.Errorf("s7 write response error %d/%d", response[17], response[18]))
	}
	for i, action := range actions {
		if response[21+i] != 0xFF {
			return collection.DeviceActionError(action, fmt.Errorf("s7 write failed: item %d return code 0x%02x", i, response[21+i]))
		}
	}
	return nil
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
