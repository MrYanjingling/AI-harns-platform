package s7

import (
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
)

type decodedValue struct {
	variable *collection.Variable
	value    any
}

func decodeFrame(frame *s7Frame, data []byte) ([]decodedValue, error) {
	values := make([]decodedValue, 0, len(frame.Variables))
	for _, item := range frame.Variables {
		base := int(item.Start) + 4
		if base >= len(data) {
			return nil, fmt.Errorf("variable %s response offset out of range", item.Variable.Name)
		}
		var value any
		switch strings.ToLower(item.Variable.DataType) {
		case "bool":
			value = (1<<item.Bit)&data[base] != 0
		case "uint16":
			if base+2 > len(data) {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			value = binutil.ParseUint16BigEndian(data[base:])
		case "int16":
			if base+2 > len(data) {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			value = int16(binutil.ParseUint16BigEndian(data[base:]))
		case "int32":
			if base+4 > len(data) {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			value = int32(binutil.ParseUint32BigEndian(data[base:]))
		case "float32":
			if base+4 > len(data) {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			value = binutil.ParseFloat32BigEndian(data[base:])
		case "int64":
			if base+8 > len(data) {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			value = int64(binutil.ParseUint64BigEndian(data[base:]))
		case "float64":
			if base+8 > len(data) {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			value = binutil.ParseFloat64BigEndian(data[base:])
		case "string":
			if base+int(item.Length) > len(data) {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			value = strings.TrimRight(string(data[base:base+int(item.Length)]), "\x00")
		case "byte_array":
			if base+int(item.Length) > len(data) {
				return nil, fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			values := make([]int, 0, item.Length)
			for _, b := range data[base : base+int(item.Length)] {
				values = append(values, int(b))
			}
			value = values
		default:
			return nil, fmt.Errorf("variable %s data type %q is not supported", item.Variable.Name, item.Variable.DataType)
		}
		values = append(values, decodedValue{
			variable: item.Variable,
			value:    collection.ApplyTransform(value, item.Variable),
		})
	}
	return values, nil
}
