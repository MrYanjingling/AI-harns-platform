package modbus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestBuildWriteCoilAndRegister(t *testing.T) {
	coil, err := BuildWrite(&collection.Variable{
		Name: "start", DataType: "bool", Address: "1!10001", AccessMode: collection.AccessModeReadWrite,
	}, true, 1, "ABCD")
	require.NoError(t, err)
	assert.Equal(t, uint8(5), coil.Function)
	assert.Equal(t, []byte{0xFF, 0x00}, coil.Data)

	register, err := BuildWrite(&collection.Variable{
		Name: "speed", DataType: "int16", Address: "1!30001", AccessMode: collection.AccessModeReadWrite,
	}, int16(42), 1, "ABCD")
	require.NoError(t, err)
	assert.Equal(t, uint8(6), register.Function)
	assert.Equal(t, []byte{0x00, 0x2A}, register.Data)

	multi, err := BuildWrite(&collection.Variable{
		Name: "temperature", DataType: "float32", Address: "1!30001", AccessMode: collection.AccessModeReadWrite,
	}, float32(1), 1, "ABCD")
	require.NoError(t, err)
	assert.Equal(t, uint8(16), multi.Function)
	assert.Equal(t, uint16(2), multi.Quantity)
}

func TestBuildWriteRejectsReadOnlyFunctionCodes(t *testing.T) {
	_, err := BuildWrite(&collection.Variable{
		Name: "input", DataType: "int16", Address: "1!40001", AccessMode: collection.AccessModeReadWrite,
	}, int16(1), 1, "ABCD")
	require.Error(t, err)
}

func TestBuildWriteAcceptsUppercaseDataType(t *testing.T) {
	register, err := BuildWrite(&collection.Variable{
		Name: "speed", DataType: "INT16", Address: "1!30001", AccessMode: collection.AccessModeReadWrite,
	}, int16(42), 1, "ABCD")
	require.NoError(t, err)
	assert.Equal(t, uint8(6), register.Function)

	_, err = BuildWrite(&collection.Variable{
		Name: "flag", DataType: "BOOL", Address: "1!30002", AccessMode: collection.AccessModeReadWrite,
	}, true, 1, "ABCD")
	require.Error(t, err)
}
