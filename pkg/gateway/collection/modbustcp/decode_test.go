package modbustcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbus"
)

func TestDecodeRegisterLayouts(t *testing.T) {
	variable := &collection.Variable{Name: "value", DataType: "float32"}
	frame := &modbus.Frame{
		Function:  modbus.FunctionReadHoldingRegisters,
		Variables: []*modbus.FrameVariable{{Variable: variable, Offset: 0, Words: 2}},
	}
	values, err := modbus.DecodeFrame(frame, []byte{0x3F, 0x80, 0x00, 0x00}, "ABCD")
	require.NoError(t, err)
	assert.Equal(t, float32(1), values[0].Value)

	values, err = modbus.DecodeFrame(frame, []byte{0x80, 0x3F, 0x00, 0x00}, "BADC")
	require.NoError(t, err)
	assert.Equal(t, float32(1), values[0].Value)

	values, err = modbus.DecodeFrame(frame, []byte{0x00, 0x00, 0x3F, 0x80}, "CDAB")
	require.NoError(t, err)
	assert.Equal(t, float32(1), values[0].Value)

	values, err = modbus.DecodeFrame(frame, []byte{0x00, 0x00, 0x80, 0x3F}, "DCBA")
	require.NoError(t, err)
	assert.Equal(t, float32(1), values[0].Value)
}

func TestDecodeBoolBit(t *testing.T) {
	frame := &modbus.Frame{
		Function:  modbus.FunctionReadHoldingRegisters,
		Variables: []*modbus.FrameVariable{{Variable: &collection.Variable{Name: "bit", DataType: "bool", Bits: 3}, Offset: 0, Words: 1}},
	}
	values, err := modbus.DecodeFrame(frame, []byte{0x00, 0x04}, "ABCD")
	require.NoError(t, err)
	assert.Equal(t, true, values[0].Value)

	values, err = modbus.DecodeFrame(frame, []byte{0x00, 0x02}, "ABCD")
	require.NoError(t, err)
	assert.Equal(t, false, values[0].Value)
}

func TestApplyTransform(t *testing.T) {
	variable := &collection.Variable{DataType: "float32", Rate: 0.1, Offset: -1, Precision: 2}
	assert.Equal(t, float32(1.5), collection.ApplyTransform(float32(25), variable))

	variable.Rate = 0
	variable.Offset = 0
	assert.Equal(t, float32(25), collection.ApplyTransform(float32(25), variable))
}

func TestDecodeFrameCoils(t *testing.T) {
	frame := &modbus.Frame{
		Function: modbus.FunctionReadCoils,
		Variables: []*modbus.FrameVariable{{
			Variable: &collection.Variable{Name: "on", DataType: "bool"},
			Offset:   2,
		}},
	}
	values, err := modbus.DecodeFrame(frame, []byte{0, 0, 1, 0}, "ABCD")
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, true, values[0].Value)
}
