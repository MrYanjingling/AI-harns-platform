package heidenhain

import (
	"testing"
)

func TestParseAddress(t *testing.T) {
	address, err := ParseAddress("W100", "UINT16")
	if err != nil || address.Area != 'W' || address.Number != 100 || address.Size != 2 {
		t.Fatalf("word = %+v err=%v", address, err)
	}
	address, err = ParseAddress("B0.3", "BIT")
	if err != nil || address.Area != 'B' || address.Bit != 3 {
		t.Fatalf("bit = %+v err=%v", address, err)
	}
	address, err = ParseAddress("S0.10", "STRING")
	if err != nil || address.Area != 'S' || address.Size != 10 {
		t.Fatalf("string = %+v err=%v", address, err)
	}
	address, err = ParseAddress("machinePosition.X", "DOUBLE")
	if err != nil || address.Kind != kindMachinePosition || address.Axis != 0 {
		t.Fatalf("position = %+v err=%v", address, err)
	}
	address, err = ParseAddress("spindleOverride", "DOUBLE")
	if err != nil || address.Kind != kindSpindleOverride {
		t.Fatalf("spindleOverride = %+v err=%v", address, err)
	}
	address, err = ParseAddress("spindleToolLength", "DOUBLE")
	if err != nil || address.Kind != kindToolLength {
		t.Fatalf("spindleToolLength = %+v err=%v", address, err)
	}
	if _, err := ParseAddress("D2", "UINT32"); err == nil {
		t.Fatal("expected alignment error")
	}
}

func TestDecodePLC(t *testing.T) {
	value, err := decodePLC([]byte{0x34, 0x12}, Address{Size: 2}, "UINT16")
	if err != nil || value != uint16(0x1234) {
		t.Fatalf("value=%v err=%v", value, err)
	}
	value, err = decodePLC([]byte{0x05}, Address{Bit: 1}, "BIT")
	if err != nil || value != int(0) {
		t.Fatalf("bit=%v err=%v", value, err)
	}
}

func TestDecodeCNC(t *testing.T) {
	value, err := decodeProgramState([]byte{0, 0, 0, 7, 'M', 'A', 'I', 'N', 0, 'C', 'U', 'R', 0}, kindProgramMain)
	if err != nil || value != "MAIN" {
		t.Fatalf("main=%v err=%v", value, err)
	}
	value, err = decodeProgramState([]byte{0, 0, 0, 7, 'M', 'A', 'I', 'N', 0, 'C', 'U', 'R', 0}, kindProgramCurrent)
	if err != nil || value != "CUR" {
		t.Fatalf("current=%v err=%v", value, err)
	}
	value, err = decodeOverride([]byte{0, 0, 0, 80, 0, 0, 0, 90, 0, 0, 0, 100}, kindSpindleOverride)
	if err != nil || value != float64(0.9) {
		t.Fatalf("override=%v err=%v", value, err)
	}
}
