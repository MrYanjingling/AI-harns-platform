package tsl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertValueScalars(t *testing.T) {
	tests := []struct {
		name string
		def  DataTypeDetail
		raw  string
		want any
	}{
		{"int-number", DataTypeDetail{Type: DataTypeInt}, `42`, int64(42)},
		{"int-string", DataTypeDetail{Type: DataTypeLong}, `"9007199254740993"`, int64(9007199254740993)},
		{"float-number", DataTypeDetail{Type: DataTypeDouble, Min: strPtr("0.1"), Max: strPtr("0.1")}, `0.1`, 0.1},
		{"bool-one", DataTypeDetail{Type: DataTypeBool, TrueValue: strPtr("on")}, `"1"`, true},
		{"bool-zero", DataTypeDetail{Type: DataTypeBool, FalseValue: strPtr("off")}, `"0"`, false},
		{"bool-label", DataTypeDetail{Type: DataTypeBool, TrueValue: strPtr("on"), FalseValue: strPtr("off")}, `"off"`, false},
		{"enum", DataTypeDetail{Type: DataTypeEnum, EnumSpec: &EnumSpec{Values: []EnumValue{{Value: 1}}}}, `"1"`, int64(1)},
		{"text", DataTypeDetail{Type: DataTypeText, Length: strPtr("3")}, `"温度"`, "温度"},
		{"time", DataTypeDetail{Type: DataTypeTime}, `"2026-01-02T03:04:05Z"`, "2026-01-02T03:04:05Z"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConvertValue(tc.def, json.RawMessage(tc.raw))
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConvertValueRejectsInvalidScalars(t *testing.T) {
	tests := []struct {
		name string
		def  DataTypeDetail
		raw  string
	}{
		{"int-decimal", DataTypeDetail{Type: DataTypeInt}, `1.5`},
		{"int-max", DataTypeDetail{Type: DataTypeInt, Max: strPtr("100")}, `101`},
		{"float-max", DataTypeDetail{Type: DataTypeDouble, Max: strPtr("0.1")}, `0.11`},
		{"float-nan", DataTypeDetail{Type: DataTypeDouble}, `"NaN"`},
		{"bool", DataTypeDetail{Type: DataTypeBool}, `"yes"`},
		{"enum", DataTypeDetail{Type: DataTypeEnum, EnumSpec: &EnumSpec{Values: []EnumValue{{Value: 1}}}}, `2`},
		{"text-length", DataTypeDetail{Type: DataTypeText, Length: strPtr("1")}, `"温度"`},
		{"text-not-string", DataTypeDetail{Type: DataTypeText}, `1`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ConvertValue(tc.def, json.RawMessage(tc.raw))
			require.Error(t, err)
		})
	}
}

func TestConvertValueArray(t *testing.T) {
	def := DataTypeDetail{
		Type: DataTypeArray,
		ArraySpec: &ArraySpec{
			Size:    strPtr("3"),
			Element: DataTypeDetail{Type: DataTypeInt},
		},
	}

	got, err := ConvertValue(def, json.RawMessage(`"[1,2,3]"`))
	require.NoError(t, err)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, got)

	_, err = ConvertValue(def, json.RawMessage(`"[1,2,3,4]"`))
	require.Error(t, err)

	_, err = ConvertValue(def, json.RawMessage(`"[1,2.5]"`))
	require.Error(t, err)
}

func TestConvertValueStruct(t *testing.T) {
	def := DataTypeDetail{
		Type: DataTypeStruct,
		StructSpec: &StructSpec{Fields: []StructField{
			{Identifier: "high", DataType: DataTypeDetail{Type: DataTypeDouble, Min: strPtr("0"), Max: strPtr("100")}},
			{Identifier: "enabled", DataType: DataTypeDetail{Type: DataTypeBool}},
		}},
	}

	got, err := ConvertValue(def, json.RawMessage(`"{\"high\":80,\"enabled\":\"1\",\"unknown\":true}"`))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"high": float64(80), "enabled": true}, got)

	_, err = ConvertValue(def, json.RawMessage(`{"high":101}`))
	require.Error(t, err)
}

func TestConvertValueNull(t *testing.T) {
	_, err := ConvertValue(DataTypeDetail{Type: DataTypeInt}, json.RawMessage(`null`))
	require.Error(t, err)
	assert.True(t, IsNullValue(json.RawMessage(`null`)))
	assert.True(t, IsNullValue(nil))
	assert.False(t, IsNullValue(json.RawMessage(`"null"`)))
}
