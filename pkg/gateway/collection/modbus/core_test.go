package modbus

import (
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestDecodeFrameKeepsSuccessfulValues(t *testing.T) {
	frame := &Frame{
		Function: FunctionReadHoldingRegisters,
		Variables: []*FrameVariable{
			{Variable: &collection.Variable{Name: "a", DataType: "int16"}, Offset: 0, Words: 1},
			{Variable: &collection.Variable{Name: "b", DataType: "int32"}, Offset: 2, Words: 2},
		},
	}
	values, err := DecodeFrame(frame, []byte{0x00, 0x0A}, "ABCD")
	if err == nil {
		t.Fatal("expected decode error")
	}
	if len(values) != 1 || values[0].Variable.Name != "a" || values[0].Value != int16(10) {
		t.Fatalf("unexpected decoded values %+v", values)
	}
}
