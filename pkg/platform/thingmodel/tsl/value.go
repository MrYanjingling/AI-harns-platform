package tsl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// IsNullValue reports whether raw is JSON null or empty.
func IsNullValue(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}

// ConvertValue converts a reported JSON value into the Go representation used
// by timeseries storage. It validates the value against the data type rules.
func ConvertValue(def DataTypeDetail, raw json.RawMessage) (any, error) {
	if IsNullValue(raw) {
		return nil, fmt.Errorf("value is null")
	}

	switch def.Type {
	case DataTypeInt, DataTypeLong:
		return convertInteger(def, raw)
	case DataTypeFloat, DataTypeDouble:
		return convertFloat(def, raw)
	case DataTypeBool:
		return convertBool(def, raw)
	case DataTypeEnum:
		return convertEnum(def, raw)
	case DataTypeText:
		return convertText(def, raw)
	case DataTypeTime:
		return convertString(raw)
	case DataTypeArray:
		return convertArray(def, raw)
	case DataTypeStruct:
		return convertStruct(def, raw)
	default:
		return nil, fmt.Errorf("unsupported data type %q", def.Type)
	}
}

func convertInteger(def DataTypeDetail, raw json.RawMessage) (any, error) {
	text, err := scalarText(raw)
	if err != nil {
		return nil, err
	}
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid integer value %q", text)
	}
	if err := validateValueRange(def, new(big.Rat).SetInt64(value)); err != nil {
		return nil, err
	}
	return value, nil
}

func convertFloat(def DataTypeDetail, raw json.RawMessage) (any, error) {
	text, err := scalarText(raw)
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("invalid number value %q", text)
	}
	rat, ok := new(big.Rat).SetString(text)
	if !ok {
		rat = new(big.Rat).SetFloat64(value)
	}
	if rat == nil {
		return nil, fmt.Errorf("invalid number value %q", text)
	}
	if err := validateValueRange(def, rat); err != nil {
		return nil, err
	}
	return value, nil
}

func convertBool(def DataTypeDetail, raw json.RawMessage) (any, error) {
	trimmed := bytes.TrimSpace(raw)
	switch string(trimmed) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	text, err := scalarText(raw)
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	switch strings.ToLower(text) {
	case "1", "true":
		return true, nil
	case "0", "false":
		return false, nil
	}
	if def.TrueValue != nil && text == strings.TrimSpace(*def.TrueValue) {
		return true, nil
	}
	if def.FalseValue != nil && text == strings.TrimSpace(*def.FalseValue) {
		return false, nil
	}
	return nil, fmt.Errorf("invalid bool value %q", text)
}

func convertEnum(def DataTypeDetail, raw json.RawMessage) (any, error) {
	if def.EnumSpec == nil || len(def.EnumSpec.Values) == 0 {
		return nil, fmt.Errorf("enum spec is required")
	}
	text, err := scalarText(raw)
	if err != nil {
		return nil, err
	}
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid enum value %q", text)
	}
	for _, item := range def.EnumSpec.Values {
		if int64(item.Value) == value {
			return value, nil
		}
	}
	return nil, fmt.Errorf("enum value %d is not defined", value)
}

func convertText(def DataTypeDetail, raw json.RawMessage) (any, error) {
	text, err := rawString(raw)
	if err != nil {
		return nil, err
	}
	if def.Length != nil && strings.TrimSpace(*def.Length) != "" {
		limit, err := strconv.Atoi(strings.TrimSpace(*def.Length))
		if err == nil && limit > 0 && utf8.RuneCountInString(text) > limit {
			return nil, fmt.Errorf("text length %d exceeds %d", utf8.RuneCountInString(text), limit)
		}
	}
	return text, nil
}

func convertString(raw json.RawMessage) (any, error) {
	return rawString(raw)
}

func convertArray(def DataTypeDetail, raw json.RawMessage) (any, error) {
	if def.ArraySpec == nil {
		return nil, fmt.Errorf("array spec is required")
	}

	payload, err := jsonPayload(raw)
	if err != nil {
		return nil, err
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(payload, &rawItems); err != nil {
		return nil, fmt.Errorf("invalid array value: %w", err)
	}
	if def.ArraySpec.Size != nil && strings.TrimSpace(*def.ArraySpec.Size) != "" {
		limit, err := strconv.Atoi(strings.TrimSpace(*def.ArraySpec.Size))
		if err == nil && limit > 0 && len(rawItems) > limit {
			return nil, fmt.Errorf("array length %d exceeds %d", len(rawItems), limit)
		}
	}

	items := make([]any, 0, len(rawItems))
	for i, rawItem := range rawItems {
		value, err := ConvertValue(def.ArraySpec.Element, rawItem)
		if err != nil {
			return nil, fmt.Errorf("element[%d]: %w", i, err)
		}
		items = append(items, value)
	}
	return items, nil
}

func convertStruct(def DataTypeDetail, raw json.RawMessage) (any, error) {
	if def.StructSpec == nil {
		return nil, fmt.Errorf("struct spec is required")
	}

	payload, err := jsonPayload(raw)
	if err != nil {
		return nil, err
	}
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rawFields); err != nil {
		return nil, fmt.Errorf("invalid struct value: %w", err)
	}

	fields := make(map[string]any, len(def.StructSpec.Fields))
	for _, field := range def.StructSpec.Fields {
		rawField, ok := rawFields[field.Identifier]
		if !ok || IsNullValue(rawField) {
			continue
		}
		value, err := ConvertValue(field.DataType, rawField)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Identifier, err)
		}
		fields[field.Identifier] = value
	}
	return fields, nil
}

func validateValueRange(def DataTypeDetail, value *big.Rat) error {
	if def.Min != nil && strings.TrimSpace(*def.Min) != "" {
		min, ok := new(big.Rat).SetString(strings.TrimSpace(*def.Min))
		if ok && value.Cmp(min) < 0 {
			return fmt.Errorf("value is below min %s", strings.TrimSpace(*def.Min))
		}
	}
	if def.Max != nil && strings.TrimSpace(*def.Max) != "" {
		max, ok := new(big.Rat).SetString(strings.TrimSpace(*def.Max))
		if ok && value.Cmp(max) > 0 {
			return fmt.Errorf("value is above max %s", strings.TrimSpace(*def.Max))
		}
	}
	return nil
}

func scalarText(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("value is empty")
	}
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return "", fmt.Errorf("invalid string value: %w", err)
		}
		return text, nil
	}
	return string(trimmed), nil
}

func rawString(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return "", fmt.Errorf("value must be a string")
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return "", fmt.Errorf("invalid string value: %w", err)
	}
	return text, nil
}

func jsonPayload(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("value is empty")
	}
	if trimmed[0] != '"' {
		return trimmed, nil
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return nil, fmt.Errorf("invalid string value: %w", err)
	}
	return json.RawMessage(text), nil
}
