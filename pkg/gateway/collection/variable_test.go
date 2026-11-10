package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/validation"
)

func TestNormalizeVariables(t *testing.T) {
	variables := []*Variable{{
		Name:       " temperature ",
		Type:       " telemetry ",
		DataType:   " float32 ",
		Address:    " holding:40001 ",
		AccessMode: AccessModeRead,
	}}
	require.NoError(t, NormalizeVariables(variables))
	assert.Equal(t, "temperature", variables[0].Name)
	assert.Equal(t, "telemetry", variables[0].Type)
	assert.Equal(t, "float32", variables[0].DataType)
	assert.Equal(t, "holding:40001", variables[0].Address)
	assert.Equal(t, float64(1), variables[0].Rate)
}

func TestValidateVariablesRejectsDuplicatesAndInvalidFields(t *testing.T) {
	err := ValidateVariables([]*Variable{
		{Name: "temperature", DataType: "float32", Address: "holding:1", AccessMode: AccessModeRead},
		{Name: " temperature ", DataType: "float32", Address: "holding:2", AccessMode: AccessModeRead},
	})
	require.Error(t, err)
	var verr *validation.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "variables[1].name", verr.Issues[0].Field)

	err = ValidateVariables([]*Variable{{
		Name: "temperature", DataType: "float32", Address: "holding:1", AccessMode: "bad",
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
	variable.Address = "holding:1"
	require.Error(t, ValidateVariables([]*Variable{variable}))

	variable.Address = ""
	variable.Expression = ""
	require.Error(t, ValidateVariables([]*Variable{variable}))
}

func TestValidateNormalVariableRejectsExpression(t *testing.T) {
	err := ValidateVariables([]*Variable{{
		Name: "temperature", DataType: "float32", Address: "holding:1",
		AccessMode: AccessModeRead, Expression: "a + b",
	}})
	require.Error(t, err)
}
