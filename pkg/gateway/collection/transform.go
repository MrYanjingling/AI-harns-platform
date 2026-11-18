package collection

import (
	"math"
	"strings"
)

// ApplyTransform applies rate, offset, data type conversion and precision.
func ApplyTransform(value any, variable *Variable) any {
	if variable == nil {
		return value
	}
	dataType := strings.ToLower(variable.DataType)
	if dataType == "bit" {
		return value
	}
	if dataType == "bool" {
		if boolean, ok := value.(bool); ok {
			return boolean
		}
		number, ok := ToFloat64(value)
		if !ok {
			return value
		}
		return number != 0
	}
	number, ok := ToFloat64(value)
	if !ok {
		return value
	}
	rate := variable.RateValue()
	if rate != 0 && rate != 1 {
		number *= rate
	}
	number += variable.OffsetValue()
	switch dataType {
	case "int8":
		return int8(number)
	case "uint8":
		return uint8(number)
	case "int16":
		return int16(number)
	case "uint16":
		return uint16(number)
	case "int32":
		return int32(number)
	case "uint32":
		return uint32(number)
	case "int64":
		return int64(number)
	case "uint64":
		return uint64(number)
	case "int":
		return int(number)
	case "uint":
		return uint(number)
	case "float32":
		return float32(applyPrecision(number, variable.PrecisionValue()))
	case "float64", "number":
		return applyPrecision(number, variable.PrecisionValue())
	default:
		return value
	}
}

func applyPrecision(number float64, precision int) float64 {
	if precision <= 0 {
		return number
	}
	factor := math.Pow10(precision)
	return math.Round(number*factor) / factor
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
