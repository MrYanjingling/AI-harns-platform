package modbustcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestCompileAndCalculate(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "voltage", DataType: "float64", Address: "1!40001", AccessMode: collection.AccessModeRead},
		{Name: "current", DataType: "float64", Address: "1!40003", AccessMode: collection.AccessModeRead},
		{Name: "power", DataType: "float64", IsCalculated: true, Expression: "voltage * current", AccessMode: collection.AccessModeRead},
		{Name: "energy", DataType: "float64", IsCalculated: true, Expression: "power * 2", AccessMode: collection.AccessModeRead},
	}
	calcs, err := compileCalculations(variables)
	require.NoError(t, err)
	require.Len(t, calcs, 2)
	assert.Equal(t, "power", calcs[0].Variable.Name)
	assert.Equal(t, "energy", calcs[1].Variable.Name)

	meta := &collection.CollectionMeta{Variables: variables}
	meta.IndexCollection()
	meta.SetVariableValue("voltage", float64(2))
	meta.SetVariableValue("current", float64(3))
	c := &ModbusTcpCollection{}
	c.SetCollectionMeta(meta)
	c.runtime = &collectorRuntime{calcs: calcs}
	require.NoError(t, c.calculate())
	value, ok := c.GetVariableValue("power")
	require.True(t, ok)
	assert.Equal(t, float64(6), value)
	value, ok = c.GetVariableValue("energy")
	require.True(t, ok)
	assert.Equal(t, float64(12), value)
}

func TestCompileCalculationCycle(t *testing.T) {
	_, err := compileCalculations([]*collection.Variable{
		{Name: "a", DataType: "float64", IsCalculated: true, Expression: "b", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "float64", IsCalculated: true, Expression: "a", AccessMode: collection.AccessModeRead},
	})
	require.Error(t, err)
}
