package s7

import (
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
)

type frameVariable struct {
	Variable *collection.Variable
	Start    uint
	Bit      uint8
	Length   uint16
}

type frameItem struct {
	RequestData []byte
	Start       uint
	Length      uint16
	Variables   []*frameVariable
}

type s7Frame struct {
	Request    []byte
	DataLength int
	Items      []*frameItem
	Variables  []*frameVariable
}

func buildFrames(variables []*collection.Variable, pdu uint16) ([]*s7Frame, error) {
	tags := make([]*frameVariable, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		parsed, err := parseAddress(variable.Address)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		length, err := variableByteLength(variable, parsed)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		tags = append(tags, &frameVariable{
			Variable: variable,
			Start:    uint(parsed.address),
			Bit:      parsed.bit,
			Length:   length,
		})
	}
	if len(tags) == 0 {
		return nil, nil
	}
	sortFrameVariables(tags)

	items, err := mergeTags(tags)
	if err != nil {
		return nil, err
	}
	return packItems(items, pdu)
}

func variableByteLength(variable *collection.Variable, parsed parsedAddress) (uint16, error) {
	dataType := strings.ToLower(variable.DataType)
	switch dataType {
	case "string", "byte_array":
		if parsed.area != areaDB || parsed.length == 0 {
			return 0, fmt.Errorf("string/byte_array requires a DB length suffix")
		}
		return parsed.length, nil
	case "bool", "bit":
		if parsed.length > 0 {
			return 0, fmt.Errorf("length suffix is only valid for string/byte_array")
		}
		return 1, nil
	case "int16", "uint16":
		if parsed.length > 0 {
			return 0, fmt.Errorf("length suffix is only valid for string/byte_array")
		}
		return 2, nil
	case "int32", "float32":
		if parsed.length > 0 {
			return 0, fmt.Errorf("length suffix is only valid for string/byte_array")
		}
		return 4, nil
	case "int64", "float64":
		if parsed.length > 0 {
			return 0, fmt.Errorf("length suffix is only valid for string/byte_array")
		}
		return 8, nil
	default:
		return 0, fmt.Errorf("unsupported data type %q", variable.DataType)
	}
}

func sortFrameVariables(values []*frameVariable) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0; j-- {
			left, _ := parseAddress(values[j-1].Variable.Address)
			right, _ := parseAddress(values[j].Variable.Address)
			if compareFrameVariable(left, right, values[j-1], values[j]) <= 0 {
				break
			}
			values[j-1], values[j] = values[j], values[j-1]
		}
	}
}

func compareFrameVariable(left, right parsedAddress, leftVar, rightVar *frameVariable) int {
	if left.area != right.area {
		if left.area < right.area {
			return -1
		}
		return 1
	}
	if left.blockSize != right.blockSize {
		if left.blockSize < right.blockSize {
			return -1
		}
		return 1
	}
	if left.address != right.address {
		if left.address < right.address {
			return -1
		}
		return 1
	}
	if leftVar.Length != rightVar.Length {
		if leftVar.Length < rightVar.Length {
			return -1
		}
		return 1
	}
	if left.bit < right.bit {
		return -1
	}
	if left.bit > right.bit {
		return 1
	}
	return 0
}

func mergeTags(tags []*frameVariable) ([]*frameItem, error) {
	items := make([]*frameItem, 0)
	for _, tag := range tags {
		parsed, err := parseAddress(tag.Variable.Address)
		if err != nil {
			return nil, err
		}
		noMerge := isNoMergeType(tag.Variable.DataType)
		merged := false
		if !noMerge && len(items) > 0 {
			last := items[len(items)-1]
			lastParsed, _ := parseAddress(last.Variables[0].Variable.Address)
			lastNoMerge := isNoMergeType(last.Variables[0].Variable.DataType)
			if !lastNoMerge && lastParsed.area == parsed.area && lastParsed.blockSize == parsed.blockSize {
				newEnd := uint32(parsed.address) + uint32(tag.Length)
				if newEnd-uint32(last.Start) < 33 {
					tag.Start = uint(parsed.address - uint32(last.Start))
					last.Variables = append(last.Variables, tag)
					end := uint32(parsed.address) + uint32(tag.Length)
					if end-uint32(last.Start) > uint32(last.Length) {
						last.Length = uint16(end - uint32(last.Start))
					}
					merged = true
				}
			}
		}
		if merged {
			continue
		}
		item := &frameItem{
			Start:     uint(parsed.address),
			Variables: []*frameVariable{tag},
			Length:    tag.Length,
		}
		tag.Start = 0
		items = append(items, item)
	}
	for _, item := range items {
		parsed, _ := parseAddress(item.Variables[0].Variable.Address)
		item.RequestData = newReadParameterItem(item.Length, uint16(parsed.blockSize), parsed.area.code(), uint32(item.Start), 0)
	}
	return items, nil
}

func isNoMergeType(dataType string) bool {
	return strings.EqualFold(dataType, "string") || strings.EqualFold(dataType, "byte_array")
}

func packItems(items []*frameItem, pdu uint16) ([]*s7Frame, error) {
	budget := int(pdu) - 21
	if budget <= 0 {
		return nil, fmt.Errorf("s7 pdu %d is too small", pdu)
	}
	const (
		requestItemSize  = 12
		responseItemSize = 4
		maxProtocolItems = 255
	)
	frames := make([]*s7Frame, 0)
	current := make([]*frameItem, 0)
	requestCost := 0
	responseCost := 0
	flush := func() error {
		if len(current) == 0 {
			return nil
		}
		responseOffset := 0
		for _, item := range current {
			item.Start = uint(responseOffset)
			for _, variable := range item.Variables {
				variable.Start = item.Start + responseItemSize + variable.Start
			}
			responseOffset += responseItemSize + int(item.Length)
		}
		frames = append(frames, newS7Frame(current, responseOffset))
		current = nil
		requestCost = 0
		responseCost = 0
		return nil
	}
	for _, item := range items {
		req := requestItemSize
		resp := responseItemSize + int(item.Length)
		if len(current) > 0 && (requestCost+req > budget || responseCost+resp > budget || len(current) >= maxProtocolItems) {
			if err := flush(); err != nil {
				return nil, err
			}
		}
		if requestCost+req > budget || responseCost+resp > budget {
			return nil, fmt.Errorf("s7 element at %d with length %d exceeds PDU budget", item.Start, item.Length)
		}
		current = append(current, item)
		requestCost += req
		responseCost += resp
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return frames, nil
}

func newS7Frame(items []*frameItem, dataLength int) *s7Frame {
	data := []byte{0x03, 0x00}
	maxBytes := 7 + 10 + 2 + len(items)*12
	data = append(data, binutil.Uint16ToBytesBigEndian(uint16(maxBytes))...)
	data = append(data, 0x02, 0xf0, 0x80)
	data = append(data, 0x32, 0x01, 0x00, 0x00, 0x00, 0x01)
	data = append(data, binutil.Uint16ToBytesBigEndian(uint16(2+len(items)*12))...)
	data = append(data, binutil.Uint16ToBytesBigEndian(uint16(0))...)
	data = append(data, uint8(4), uint8(len(items)))
	variables := make([]*frameVariable, 0)
	for _, item := range items {
		data = append(data, item.RequestData...)
		variables = append(variables, item.Variables...)
	}
	return &s7Frame{Request: data, DataLength: dataLength, Items: items, Variables: variables}
}

func newParameterItem(transportSize uint8, length uint16, dbNumber uint16, zone uint8, address uint32, bitAddress uint8) []byte {
	item := []byte{0x12, 0x0a, 0x10, transportSize}
	item = append(item, binutil.Uint16ToBytesBigEndian(length)...)
	item = append(item, binutil.Uint16ToBytesBigEndian(dbNumber)...)
	item = append(item, zone)
	item = append(item, uint8((address<<3)/256/256))
	item = append(item, uint8((address<<3)/256%256))
	item = append(item, uint8((address<<3)%256)+bitAddress)
	return item
}

func newReadParameterItem(length uint16, dbNumber uint16, zone uint8, address uint32, bitAddress uint8) []byte {
	return newParameterItem(0x02, length, dbNumber, zone, address, bitAddress)
}
