package df1

import (
	"bytes"
	"testing"
)

func TestParseAddress(t *testing.T) {
	address, err := ParseAddress("N7:0", "INT16")
	if err != nil || address.FileType != fileInteger || address.FileNum != 7 || address.ByteSize != 2 {
		t.Fatalf("address=%+v err=%v", address, err)
	}
	address, err = ParseAddress("B3:0/3", "BIT")
	if err != nil || address.FileType != fileBit || address.Bit != 3 {
		t.Fatalf("bit=%+v err=%v", address, err)
	}
	address, err = ParseAddress("T4:0.PRE", "INT16")
	if err != nil || address.FileType != fileTimer || address.Sub != "PRE" || address.ByteSize != 6 {
		t.Fatalf("timer=%+v err=%v", address, err)
	}
	if _, err := ParseAddress("T4:0", "INT16"); err == nil {
		t.Fatal("expected timer subtype error")
	}
}

func TestBuildFrameCRC(t *testing.T) {
	payload := []byte{1, 0, 0x0F, 0, 0, 0, 0xA2, 2, 7, fileInteger, 0, 0}
	frame := buildFrame(payload)
	want := []byte{0x10, 0x02, 1, 0, 0x0F, 0, 0, 0, 0xA2, 2, 7, 0x89, 0, 0, 0x10, 0x03, 0x6F, 0x20}
	if !bytes.Equal(frame, want) {
		t.Fatalf("frame = % X", frame)
	}
}

func TestParseAndDecode(t *testing.T) {
	address, _ := ParseAddress("N7:0", "UINT16")
	frame := []byte{0x10, 0x02, 1, 0, 0x4F, 0, 0, 0, 0xFF, 0x7F, 0x10, 0x03, 0xE3, 0x55}
	data, err := parseResponse(frame, address)
	if err != nil {
		t.Fatal(err)
	}
	value, err := decode(data, address, "UINT16")
	if err != nil || value != uint16(0x7FFF) {
		t.Fatalf("value=%v err=%v", value, err)
	}
}
