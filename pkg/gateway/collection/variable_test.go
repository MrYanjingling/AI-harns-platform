package collection

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/validation"
)

func TestVariableUnmarshalOptionalValues(t *testing.T) {
	var missing Variable
	require.NoError(t, json.Unmarshal([]byte(`{
		"name":"a","dataType":"int16","address":"1!40001","accessMode":"r"
	}`), &missing))
	assert.Nil(t, missing.Bits)
	assert.Nil(t, missing.Rate)
	assert.Nil(t, missing.Offset)
	assert.Nil(t, missing.Precision)
	assert.Equal(t, uint8(0), missing.BitsValue())
	assert.Equal(t, float64(1), missing.RateValue())
	assert.Equal(t, float64(0), missing.OffsetValue())
	assert.Equal(t, 0, missing.PrecisionValue())

	var explicitZero Variable
	require.NoError(t, json.Unmarshal([]byte(`{
		"name":"a","dataType":"int16","address":"1!40001","accessMode":"r",
		"bits":0,"rate":0,"offset":0,"precision":0
	}`), &explicitZero))
	require.NotNil(t, explicitZero.Bits)
	require.NotNil(t, explicitZero.Rate)
	require.NotNil(t, explicitZero.Offset)
	require.NotNil(t, explicitZero.Precision)
	assert.Equal(t, uint8(0), *explicitZero.Bits)
	assert.Equal(t, float64(0), *explicitZero.Rate)
	assert.Equal(t, float64(0), *explicitZero.Offset)
	assert.Equal(t, 0, *explicitZero.Precision)
}

func TestNormalizeVariables(t *testing.T) {
	variables := []*Variable{{
		Name:       " temperature ",
		Type:       " telemetry ",
		DataType:   " float32 ",
		Address:    " 1!40001 ",
		AccessMode: AccessModeRead,
	}}
	require.NoError(t, NormalizeVariables(variables))
	assert.Equal(t, "temperature", variables[0].Name)
	assert.Equal(t, "telemetry", variables[0].Type)
	assert.Equal(t, "float32", variables[0].DataType)
	assert.Equal(t, "1!40001", variables[0].Address)
	assert.Nil(t, variables[0].Rate)
	assert.Equal(t, float64(1), variables[0].RateValue())
}

func TestValidateVariablesRejectsDuplicatesAndInvalidFields(t *testing.T) {
	err := ValidateVariables([]*Variable{
		{Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: AccessModeRead},
		{Name: " temperature ", DataType: "float32", Address: "1!40002", AccessMode: AccessModeRead},
	})
	require.Error(t, err)
	var verr *validation.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "variables[1].name", verr.Issues[0].Field)

	err = ValidateVariables([]*Variable{{
		Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: "bad",
	}})
	require.Error(t, err)
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "variables[0].accessMode", verr.Issues[0].Field)

	err = ValidateVariables([]*Variable{{
		Name: "temperature", DataType: "float32", Address: "", AccessMode: AccessModeRead,
	}})
	require.Error(t, err)
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "variables[0].address", verr.Issues[0].Field)
}

func TestValidateCalculatedVariable(t *testing.T) {
	variable := &Variable{
		Name:         "power",
		DataType:     "float64",
		AccessMode:   AccessModeRead,
		IsCalculated: true,
		Expression:   "voltage * current",
	}
	require.NoError(t, ValidateVariables([]*Variable{variable}))

	variable.AccessMode = AccessModeReadWrite
	require.Error(t, ValidateVariables([]*Variable{variable}))

	variable.AccessMode = AccessModeRead
	variable.Address = "1!40001"
	require.Error(t, ValidateVariables([]*Variable{variable}))

	variable.Address = ""
	variable.Expression = ""
	require.Error(t, ValidateVariables([]*Variable{variable}))
}

func TestValidateNormalVariableRejectsExpression(t *testing.T) {
	err := ValidateVariables([]*Variable{{
		Name: "temperature", DataType: "float32", Address: "1!40001",
		AccessMode: AccessModeRead, Expression: "a + b",
	}})
	require.Error(t, err)
}

func TestValidateVariableBitsRange(t *testing.T) {
	require.NoError(t, ValidateVariables([]*Variable{{
		Name: "flag", DataType: "bool", Address: "1!40001", Bits: testUint8Pointer(16),
		AccessMode: AccessModeRead,
	}}))
	require.Error(t, ValidateVariables([]*Variable{{
		Name: "flag", DataType: "bool", Address: "1!40001", Bits: testUint8Pointer(17),
		AccessMode: AccessModeRead,
	}}))
}
