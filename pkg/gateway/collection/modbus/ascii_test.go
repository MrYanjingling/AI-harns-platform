package modbus

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildASCIIRequest(t *testing.T) {
	frame := &Frame{Slave: 1, Function: FunctionReadHoldingRegisters, Start: 0, Quantity: 1}
	got := string(BuildASCIIRequest(frame))
	if got != ":010300000001FB\r\n" {
		t.Fatalf("unexpected request %q", got)
	}
}

func TestParseASCIIResponse(t *testing.T) {
	frame := &Frame{Slave: 1, Function: FunctionReadHoldingRegisters, Start: 0, Quantity: 1}
	payload, err := ParseASCIIResponse([]byte(":010302000AF0\r\n"), frame)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, []byte{0, 10}) {
		t.Fatalf("unexpected payload % X", payload)
	}
}

func TestParseASCIIResponseCoils(t *testing.T) {
	frame := &Frame{Slave: 1, Function: FunctionReadCoils, Start: 0, Quantity: 8}
	payload, err := ParseASCIIResponse([]byte(":010101CD30\r\n"), frame)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{1, 0, 1, 1, 0, 0, 1, 1}; !bytes.Equal(payload, want) {
		t.Fatalf("unexpected coil payload %v", payload)
	}
}

func TestParseASCIIResponseRejectsBadLRC(t *testing.T) {
	frame := &Frame{Slave: 1, Function: FunctionReadHoldingRegisters, Start: 0, Quantity: 1}
	if _, err := ParseASCIIResponse([]byte(":010302000AF1\r\n"), frame); err == nil {
		t.Fatal("expected LRC error")
	}
}

func TestBuildAndValidateASCIIWrite(t *testing.T) {
	write := &WriteFrame{Slave: 1, Function: 6, Start: 10, Quantity: 1, Data: []byte{0, 10}}
	if got := string(BuildASCIIWriteRequest(write)); got != ":0106000A000AE5\r\n" {
		t.Fatalf("unexpected write request %q", got)
	}
	if err := ValidateASCIIWriteResponse([]byte(":0106000A000AE5\r\n"), write); err != nil {
		t.Fatal(err)
	}
	if err := ValidateASCIIWriteResponse([]byte(":0106000A000AE4\r\n"), write); err == nil || !strings.Contains(err.Error(), "lrc") {
		t.Fatalf("unexpected response error %v", err)
	}
}
