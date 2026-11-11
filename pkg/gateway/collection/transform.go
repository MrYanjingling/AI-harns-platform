package collection

import (
	"math"
	"strings"
)

// ApplyTransform applies rate, offset, precision and data type conversion.
func ApplyTransform(value any, variable *Variable) any {
	if variable == nil || strings.EqualFold(variable.DataType, "bool") {
		return value
	}
	number, ok := ToFloat64(value)
	if !ok {
		return value
	}
	if variable.Rate != 0 && variable.Rate != 1 {
		number *= variable.Rate
	}
	number += variable.Offset
	if variable.Precision > 0 {
		factor := math.Pow10(variable.Precision)
		number = math.Round(number*factor) / factor
	}
	switch strings.ToLower(variable.DataType) {
	case "int16":
		return int16(number)
	case "uint16":
		return uint16(number)
	case "int32":
		return int32(number)
	case "int64":
		return int64(number)
	case "float32":
		return float32(number)
	case "float64", "number":
		return number
	default:
		return value
	}
}

// ToFloat64 converts a numeric value to float64.
func ToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}
