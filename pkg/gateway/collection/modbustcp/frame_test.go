package modbustcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbus"
)

func TestBuildWireFrames(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead},
		{Name: "pressure", DataType: "int16", Address: "1!40003", AccessMode: collection.AccessModeRead},
		{Name: "status", DataType: "bool", Address: "1!10001", AccessMode: collection.AccessModeRead},
		{Name: "written", DataType: "int16", Address: "1!40004", AccessMode: collection.AccessModeWrite},
		{Name: "calculated", DataType: "float64", IsCalculated: true, Expression: "temperature * 2", AccessMode: collection.AccessModeRead},
	}

	frames, err := buildWireFrames(variables, 1)
	require.NoError(t, err)
	require.Len(t, frames, 2)

	coilFrame := frames[0]
	assert.Equal(t, modbus.FunctionReadCoils, coilFrame.frame.Function)
	assert.Equal(t, uint16(0), coilFrame.frame.Start)
	assert.Equal(t, uint16(1), coilFrame.frame.Quantity)
	assert.Equal(t, 10, coilFrame.responseLen)
	assert.Equal(t, byte(1), coilFrame.request[7])

	registerFrame := frames[1]
	assert.Equal(t, modbus.FunctionReadInputRegisters, registerFrame.frame.Function)
	assert.Equal(t, uint16(0), registerFrame.frame.Start)
	assert.Equal(t, uint16(3), registerFrame.frame.Quantity)
	assert.Equal(t, 15, registerFrame.responseLen)
	require.Len(t, registerFrame.frame.Variables, 2)
	assert.Equal(t, 0, registerFrame.frame.Variables[0].Offset)
	assert.Equal(t, 4, registerFrame.frame.Variables[1].Offset)
}

func TestBuildWireFramesRejectsRegisterBoolWithoutBits(t *testing.T) {
	_, err := buildWireFrames([]*collection.Variable{{
		Name: "status", DataType: "bool", Address: "1!40001", AccessMode: collection.AccessModeRead,
	}}, 1)
	require.Error(t, err)
}

func TestBuildWireFramesSplitsAtRegisterLimit(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "float32", Address: "1!40125", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildWireFrames(variables, 1)
	require.NoError(t, err)
	require.Len(t, frames, 2)
	assert.Equal(t, uint16(0), frames[0].frame.Start)
	assert.Equal(t, uint16(124), frames[1].frame.Start)
}
