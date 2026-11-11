package collection

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var numericActionValuePattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// ActionValue is one validated write action.
type ActionValue struct {
	Index    int
	Variable *Variable
	Value    any
}

// ActionHandler is implemented by collectors that support writes.
type ActionHandler interface {
	DeliverAction(ctx context.Context, actions []ActionValue) error
}

// DecodeActionValue converts an engineering value string into a raw typed value.
func DecodeActionValue(variable *Variable, text string) (any, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	value := strings.TrimSpace(text)
	if value == "" {
		return nil, fmt.Errorf("value is empty")
	}
	if strings.EqualFold(variable.DataType, "bool") {
		switch strings.ToLower(value) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("value %q is not a bool", text)
		}
	}
	if strings.EqualFold(variable.DataType, "string") {
		return value, nil
	}
	if !numericActionValuePattern.MatchString(value) {
		return nil, fmt.Errorf("value %q must be a decimal number", text)
	}
	valueRat, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("value %q is invalid", text)
	}
	rate, err := floatRat(variable.Rate)
	if err != nil {
		return nil, err
	}
	if rate.Sign() == 0 {
		rate.SetInt64(1)
	}
	offset, err := floatRat(variable.Offset)
	if err != nil {
		return nil, err
	}
	rawRat := new(big.Rat).Quo(new(big.Rat).Sub(valueRat, offset), rate)

	switch strings.ToLower(variable.DataType) {
	case "int16":
		return decodeIntAction(rawRat, "-32768", "32767", "int16")
	case "uint16":
		return decodeUintAction(rawRat, "65535", "uint16")
	case "int32":
		return decodeIntAction(rawRat, "-2147483648", "2147483647", "int32")
	case "int64":
		return decodeInt64Action(rawRat)
	case "float32":
		number, _ := rawRat.Float64()
		if math.IsInf(number, 0) || math.IsNaN(number) {
			return nil, fmt.Errorf("value cannot be represented as float32")
		}
		if math.Abs(number) > math.MaxFloat32 {
			return nil, fmt.Errorf("value is out of float32 range")
		}
		return float32(number), nil
	case "float64", "number":
		number, _ := rawRat.Float64()
		if math.IsInf(number, 0) || math.IsNaN(number) {
			return nil, fmt.Errorf("value cannot be represented as float64")
		}
		return number, nil
	default:
		return nil, fmt.Errorf("data type %q is not writable", variable.DataType)
	}
}

func decodeIntAction(value *big.Rat, min, max, name string) (any, error) {
	if !value.IsInt() {
		return nil, fmt.Errorf("value does not resolve to an integer for %s", name)
	}
	number := new(big.Int).Set(value.Num())
	minValue, _ := new(big.Int).SetString(min, 10)
	maxValue, _ := new(big.Int).SetString(max, 10)
	if number.Cmp(minValue) < 0 || number.Cmp(maxValue) > 0 {
		return nil, fmt.Errorf("value is out of %s range", name)
	}
	switch name {
	case "int16":
		return int16(number.Int64()), nil
	case "int32":
		return int32(number.Int64()), nil
	default:
		return number.Int64(), nil
	}
}

func decodeInt64Action(value *big.Rat) (any, error) {
	if !value.IsInt() {
		return nil, fmt.Errorf("value does not resolve to an int64")
	}
	number := new(big.Int).Set(value.Num())
	minValue, _ := new(big.Int).SetString("-9223372036854775808", 10)
	maxValue, _ := new(big.Int).SetString("9223372036854775807", 10)
	if number.Cmp(minValue) < 0 || number.Cmp(maxValue) > 0 {
		return nil, fmt.Errorf("value is out of int64 range")
	}
	return number.Int64(), nil
}

func decodeUintAction(value *big.Rat, max, name string) (any, error) {
	if !value.IsInt() {
		return nil, fmt.Errorf("value does not resolve to an integer for %s", name)
	}
	number := new(big.Int).Set(value.Num())
	if number.Sign() < 0 {
		return nil, fmt.Errorf("value is out of %s range", name)
	}
	maxValue, _ := new(big.Int).SetString(max, 10)
	if number.Cmp(maxValue) > 0 {
		return nil, fmt.Errorf("value is out of %s range", name)
	}
	return uint16(number.Int64()), nil
}

func floatRat(value float64) (*big.Rat, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("rate or offset is not finite")
	}
	text := strconv.FormatFloat(value, 'f', -1, 64)
	rat, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, fmt.Errorf("rate or offset is invalid")
	}
	return rat, nil
}
