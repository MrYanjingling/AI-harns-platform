package modbustcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection/modbus"
)

func TestParseVariableAddress(t *testing.T) {
	slave, function, address, err := modbus.ParseAddress("1!40001", 1)
	require.NoError(t, err)
	assert.Equal(t, uint8(1), slave)
	assert.Equal(t, modbus.FunctionReadInputRegisters, function)
	assert.Equal(t, uint16(0), address)

	_, _, address, err = modbus.ParseAddress("1!40001", 0)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), address)

	_, _, address, err = modbus.ParseAddress("2!30010", 1)
	require.NoError(t, err)
	assert.Equal(t, uint16(9), address)

	_, _, _, err = modbus.ParseAddress("1!40001", 2)
	require.Error(t, err)
	_, _, _, err = modbus.ParseAddress("bad", 0)
	require.Error(t, err)
	_, _, _, err = modbus.ParseAddress("0!40001", 0)
	require.Error(t, err)
}

func TestDataTypeWords(t *testing.T) {
	for dataType, want := range map[string]uint16{
		"bool": 1, "int16": 1, "uint16": 1,
		"int32": 2, "float32": 2,
		"int64": 4, "float64": 4,
	} {
		got, err := modbus.DataTypeWords(dataType)
		require.NoError(t, err, dataType)
		assert.Equal(t, want, got, dataType)
	}
	_, err := modbus.DataTypeWords("string")
	require.Error(t, err)
}
