// Package modbus contains shared frame and value logic for Modbus transports.
package modbus

import (
	"fmt"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
)

const (
	FunctionReadCoils            uint8 = 1
	FunctionReadDiscreteInputs   uint8 = 2
	FunctionReadHoldingRegisters uint8 = 3
	FunctionReadInputRegisters   uint8 = 4
	PerRequestMaxCoil                  = 1983
	PerRequestMaxRegister              = 123
)

type Point struct {
	Variable *collection.Variable
	Slave    uint8
	Function uint8
	Address  uint16
	Words    uint16
}

type FrameVariable struct {
	Variable *collection.Variable
	Offset   int
	Words    uint16
}

type Frame struct {
	Slave     uint8
	Function  uint8
	Start     uint16
	Quantity  uint16
	Variables []*FrameVariable
}

type DecodedValue struct {
	Variable *collection.Variable
	Value    any
}

func BuildFrames(variables []*collection.Variable, positionAddress uint) ([]*Frame, error) {
	groups := make(map[frameKey][]*Point)
	for _, variable := range variables {
		point, err := ParsePoint(variable, positionAddress)
		if err != nil {
			return nil, err
		}
		if point == nil {
			continue
		}
		key := frameKey{slave: point.Slave, function: point.Function}
		groups[key] = append(groups[key], point)
	}
	if len(groups) == 0 {
		return nil, nil
	}

	keys := make([]frameKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sortFrameKeys(keys)

	frames := make([]*Frame, 0)
	for _, key := range keys {
		points := groups[key]
		sortPoints(points)
		groupFrames, err := buildGroupFrames(key, points)
		if err != nil {
			return nil, err
		}
		frames = append(frames, groupFrames...)
	}
	return frames, nil
}

type frameKey struct {
	slave    uint8
	function uint8
}

func buildGroupFrames(key frameKey, points []*Point) ([]*Frame, error) {
	var (
		frames  []*Frame
		current *Frame
	)
	maxSpan := uint32(PerRequestMaxRegister)
	if key.function == FunctionReadCoils || key.function == FunctionReadDiscreteInputs {
		maxSpan = PerRequestMaxCoil
	}
	flush := func() {
		if current == nil {
			return
		}
		current.Quantity = uint16(frameEnd(current) - uint32(current.Start))
		frames = append(frames, current)
		current = nil
	}
	for _, point := range points {
		end := uint32(point.Address) + uint32(point.Words)
		if end > 65536 {
			return nil, fmt.Errorf("variable %s address range exceeds 65535", point.Variable.Name)
		}
		if current == nil {
			current = &Frame{Slave: key.slave, Function: key.function, Start: point.Address}
		} else if end > uint32(current.Start)+maxSpan {
			flush()
			current = &Frame{Slave: key.slave, Function: key.function, Start: point.Address}
		}
		offset := int(point.Address - current.Start)
		if key.function == FunctionReadHoldingRegisters || key.function == FunctionReadInputRegisters {
			offset *= 2
		}
		current.Variables = append(current.Variables, &FrameVariable{
			Variable: point.Variable,
			Offset:   offset,
			Words:    point.Words,
		})
	}
	flush()
	return frames, nil
}

func frameEnd(frame *Frame) uint32 {
	var end uint32
	for _, variable := range frame.Variables {
		value := uint32(frame.Start) + uint32(variable.Offset) + 1
		if frame.Function == FunctionReadHoldingRegisters || frame.Function == FunctionReadInputRegisters {
			value = uint32(frame.Start) + uint32(variable.Offset/2) + uint32(variable.Words)
		}
		if value > end {
			end = value
		}
	}
	return end
}

func ParsePoint(variable *collection.Variable, positionAddress uint) (*Point, error) {
	if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
		return nil, nil
	}
	slave, function, address, err := ParseAddress(variable.Address, positionAddress)
	if err != nil {
		return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
	}
	words, err := DataTypeWords(variable.DataType)
	if err != nil {
		return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
	}
	if function == FunctionReadCoils || function == FunctionReadDiscreteInputs {
		if !strings.EqualFold(variable.DataType, "bool") {
			return nil, fmt.Errorf("variable %s: coil/status points must use bool", variable.Name)
		}
		words = 1
	}
	if function == FunctionReadHoldingRegisters || function == FunctionReadInputRegisters {
		if strings.EqualFold(variable.DataType, "bool") && (variable.Bits < 1 || variable.Bits > 16) {
			return nil, fmt.Errorf("variable %s: bool register bits must be 1..16", variable.Name)
		}
	}
	return &Point{Variable: variable, Slave: slave, Function: function, Address: address, Words: words}, nil
}

func ParseAddress(value string, positionAddress uint) (uint8, uint8, uint16, error) {
	raw := strings.TrimSpace(value)
	parts := strings.SplitN(raw, "!", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return 0, 0, 0, fmt.Errorf("address %q must use slave!functionCodeAddress format", value)
	}
	slaveValue, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 8)
	if err != nil || slaveValue == 0 || slaveValue > 247 {
		return 0, 0, 0, fmt.Errorf("address %q has invalid slave", value)
	}
	encoded := strings.TrimSpace(parts[1])
	if len(encoded) < 2 {
		return 0, 0, 0, fmt.Errorf("address %q has no function code", value)
	}
	function := encoded[0] - '0'
	if function < FunctionReadCoils || function > FunctionReadInputRegisters {
		return 0, 0, 0, fmt.Errorf("address %q has invalid function code", value)
	}
	rawAddress, err := strconv.ParseUint(encoded[1:], 10, 16)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("address %q has invalid register address", value)
	}
	if uint64(positionAddress) > rawAddress {
		return 0, 0, 0, fmt.Errorf("address %q is smaller than positionAddress", value)
	}
	return uint8(slaveValue), function, uint16(rawAddress - uint64(positionAddress)), nil
}

func DataTypeWords(dataType string) (uint16, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "int16", "uint16":
		return 1, nil
	case "int32", "float32":
		return 2, nil
	case "int64", "float64":
		return 4, nil
	default:
		return 0, fmt.Errorf("data type %q is not supported by modbus", dataType)
	}
}

func DecodeFrame(frame *Frame, payload []byte, memoryLayout string) ([]DecodedValue, error) {
	values := make([]DecodedValue, 0, len(frame.Variables))
	for _, item := range frame.Variables {
		var (
			value any
			err   error
		)
		switch frame.Function {
		case FunctionReadCoils, FunctionReadDiscreteInputs:
			if item.Offset >= len(payload) {
				return nil, fmt.Errorf("variable %s response offset out of range", item.Variable.Name)
			}
			value = payload[item.Offset] != 0
		case FunctionReadHoldingRegisters, FunctionReadInputRegisters:
			if item.Offset+int(item.Words)*2 > len(payload) {
				return nil, fmt.Errorf("variable %s response offset out of range", item.Variable.Name)
			}
			value, err = decodeRegister(payload[item.Offset:item.Offset+int(item.Words)*2], item.Variable, memoryLayout)
			if err != nil {
				return nil, err
			}
		}
		values = append(values, DecodedValue{Variable: item.Variable, Value: collection.ApplyTransform(value, item.Variable)})
	}
	return values, nil
}

func decodeRegister(data []byte, variable *collection.Variable, memoryLayout string) (any, error) {
	layout := strings.ToUpper(memoryLayout)
	switch strings.ToLower(variable.DataType) {
	case "bool":
		v := parseUint16(data, layout)
		return (1<<(variable.Bits-1))&v != 0, nil
	case "int16":
		return int16(parseUint16(data, layout)), nil
	case "uint16":
		return parseUint16(data, layout), nil
	case "int32":
		return int32(parseUint32(data, layout)), nil
	case "int64":
		return int64(parseUint64(data, layout)), nil
	case "float32":
		return parseFloat32(data, layout), nil
	case "float64":
		return parseFloat64(data, layout), nil
	default:
		return nil, fmt.Errorf("variable %s has unsupported data type %q", variable.Name, variable.DataType)
	}
}

func parseUint16(data []byte, layout string) uint16 {
	if layout == "BADC" || layout == "DCBA" {
		return binutil.ParseUint16LittleEndian(data)
	}
	return binutil.ParseUint16BigEndian(data)
}

func parseUint32(data []byte, layout string) uint32 {
	switch layout {
	case "ABCD":
		return binutil.ParseUint32BigEndian(data)
	case "BADC":
		return binutil.ParseUint32BigEndianByteSwap(data)
	case "CDAB":
		return binutil.ParseUint32LittleEndianByteSwap(data)
	default:
		return binutil.ParseUint32LittleEndian(data)
	}
}

func parseUint64(data []byte, layout string) uint64 {
	switch layout {
	case "ABCD":
		return binutil.ParseUint64BigEndian(data)
	case "BADC":
		return binutil.ParseUint64BigEndianByteSwap(data)
	case "CDAB":
		return binutil.ParseUint64LittleEndianByteSwap(data)
	default:
		return binutil.ParseUint64LittleEndian(data)
	}
}

func parseFloat32(data []byte, layout string) float32 {
	switch layout {
	case "ABCD":
		return binutil.ParseFloat32BigEndian(data)
	case "BADC":
		return binutil.ParseFloat32BigEndianByteSwap(data)
	case "CDAB":
		return binutil.ParseFloat32LittleEndianByteSwap(data)
	default:
		return binutil.ParseFloat32LittleEndian(data)
	}
}

func parseFloat64(data []byte, layout string) float64 {
	switch layout {
	case "ABCD":
		return binutil.ParseFloat64BigEndian(data)
	case "BADC":
		return binutil.ParseFloat64BigEndianByteSwap(data)
	case "CDAB":
		return binutil.ParseFloat64LittleEndianByteSwap(data)
	default:
		return binutil.ParseFloat64LittleEndian(data)
	}
}

func sortFrameKeys(keys []frameKey) {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0; j-- {
			left, right := keys[j-1], keys[j]
			if left.slave < right.slave || (left.slave == right.slave && left.function <= right.function) {
				break
			}
			keys[j-1], keys[j] = right, left
		}
	}
}

func sortPoints(points []*Point) {
	for i := 1; i < len(points); i++ {
		for j := i; j > 0 && points[j].Address < points[j-1].Address; j-- {
			points[j-1], points[j] = points[j], points[j-1]
		}
	}
}
