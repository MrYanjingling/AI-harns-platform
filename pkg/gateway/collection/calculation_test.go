package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/validation"
)

func TestValidateCalculationsRejectsMissingReference(t *testing.T) {
	err := ValidateCalculations([]*Variable{
		{Name: "one", DataType: "float64", Address: "1!40001", AccessMode: AccessModeRead},
		{Name: "two", DataType: "float64", Address: "1!40003", AccessMode: AccessModeRead},
		{Name: "control", DataType: "float64", Address: "1!40005", AccessMode: AccessModeRead},
		{Name: "result", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "(one+two+three)*control"},
	})
	require.Error(t, err)
	var verr *validation.ValidationError
	require.ErrorAs(t, err, &verr)
	require.Len(t, verr.Issues, 1)
	assert.Equal(t, "variables[3].expression", verr.Issues[0].Field)
	assert.Equal(t, "validate.calculated_variable_reference_missing", verr.Issues[0].MsgID)
	assert.Equal(t, []interface{}{"result", "three"}, verr.Issues[0].Args)
}

func TestValidateCalculationsAllowsCalculatedDependency(t *testing.T) {
	err := ValidateCalculations([]*Variable{
		{Name: "voltage", DataType: "float64", Address: "1!40001", AccessMode: AccessModeRead},
		{Name: "current", DataType: "float64", Address: "1!40003", AccessMode: AccessModeRead},
		{Name: "power", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "voltage * current"},
		{Name: "energy", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "power * 2"},
	})
	require.NoError(t, err)
}

func TestValidateCalculationsRejectsCycle(t *testing.T) {
	err := ValidateCalculations([]*Variable{
		{Name: "a", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "b + 1"},
		{Name: "b", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "a + 1"},
	})
	require.Error(t, err)
	var verr *validation.ValidationError
	require.ErrorAs(t, err, &verr)
	require.Len(t, verr.Issues, 1)
	assert.Equal(t, "variables[0].expression", verr.Issues[0].Field)
	assert.Equal(t, "validate.calculated_variable_cycle", verr.Issues[0].MsgID)
}

func TestValidateCalculationsRejectsInvalidExpression(t *testing.T) {
	err := ValidateCalculations([]*Variable{
		{Name: "a", DataType: "float64", Address: "1!40001", AccessMode: AccessModeRead},
		{Name: "bad", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "a +"},
	})
	require.Error(t, err)
	var verr *validation.ValidationError
	require.ErrorAs(t, err, &verr)
	require.Len(t, verr.Issues, 1)
	assert.Equal(t, "variables[1].expression", verr.Issues[0].Field)
	assert.Equal(t, "validate.calculated_variable_expression_invalid", verr.Issues[0].MsgID)
}

func TestValidateCalculationsAllowsBuiltinNamedVariables(t *testing.T) {
	err := ValidateCalculations([]*Variable{
		{Name: "one", DataType: "float64", Address: "1!40001", AccessMode: AccessModeRead},
		{Name: "two", DataType: "float64", Address: "1!40003", AccessMode: AccessModeRead},
		{Name: "zero", DataType: "float64", Address: "1!40005", AccessMode: AccessModeRead},
		{Name: "control", DataType: "float64", Address: "1!40007", AccessMode: AccessModeRead},
		{Name: "result", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "(one+two+zero)*control"},
	})
	require.NoError(t, err)
}

func TestValidateCalculationsCompactsCompilerError(t *testing.T) {
	err := ValidateCalculations([]*Variable{
		{Name: "a", DataType: "float64", Address: "1!40001", AccessMode: AccessModeRead},
		{Name: "bad", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "a +"},
	})
	require.Error(t, err)
	var verr *validation.ValidationError
	require.ErrorAs(t, err, &verr)
	require.Len(t, verr.Issues, 1)
	require.Len(t, verr.Issues[0].Args, 2)
	message, ok := verr.Issues[0].Args[1].(string)
	require.True(t, ok)
	assert.NotContains(t, message, "\n")
	assert.LessOrEqual(t, len([]rune(message)), 163)
}

func TestCalculateAppliesDataTypeAndPrecision(t *testing.T) {
	variables := []*Variable{
		{Name: "two", DataType: "float64", AccessMode: AccessModeRead},
		{Name: "asBool", DataType: "bool", IsCalculated: true, AccessMode: AccessModeRead, Expression: "(two+1)*199"},
		{Name: "asInt", DataType: "int16", IsCalculated: true, AccessMode: AccessModeRead, Expression: "(two+1)*199"},
		{Name: "asFloat", DataType: "float64", IsCalculated: true, AccessMode: AccessModeRead, Expression: "(two+1)*199.126"},
		{Name: "asPreciseFloat", DataType: "float64", Precision: testIntPointer(2), IsCalculated: true, AccessMode: AccessModeRead, Expression: "(two+1)*199.126"},
	}
	calcs, err := CompileCalculations(variables)
	require.NoError(t, err)

	meta := &CollectionMeta{Variables: variables}
	meta.IndexCollection()
	meta.SetVariableValue("two", float64(0))
	require.NoError(t, Calculate(meta, calcs))

	value, ok := meta.GetVariableValue("asBool")
	require.True(t, ok)
	assert.Equal(t, true, value)
	value, ok = meta.GetVariableValue("asInt")
	require.True(t, ok)
	assert.Equal(t, int16(199), value)
	value, ok = meta.GetVariableValue("asFloat")
	require.True(t, ok)
	assert.Equal(t, float64(199.126), value)
	value, ok = meta.GetVariableValue("asPreciseFloat")
	require.True(t, ok)
	assert.Equal(t, float64(199.13), value)
}
