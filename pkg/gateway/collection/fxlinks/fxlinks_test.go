package fxlinks

import (
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestFXLinksFrames(t *testing.T) {
	req := buildReadRequest(0, 0, 1, true, false, "D", 0, 2)
	want := "\x0500FFWR0D0000022B"
	if string(req) != want {
		t.Fatalf("WR frame = %q, want %q", string(req), want)
	}
	resp := []byte("\x0200FF12345678\x0393")
	data, err := parseReadResponse(resp, true, false, 2)
	if err != nil {
		t.Fatal(err)
	}
	p0, _ := parseAddress(&collection.Variable{Name: "d0", DataType: "UINT16", Address: "D0"})
	p1, _ := parseAddress(&collection.Variable{Name: "d1", DataType: "UINT16", Address: "D1"})
	v0, err := decodePoint(p0, data, 0)
	if err != nil || v0 != uint16(0x1234) {
		t.Fatalf("d0=%v err=%v", v0, err)
	}
	v1, err := decodePoint(p1, data, 0)
	if err != nil || v1 != uint16(0x5678) {
		t.Fatalf("d1=%v err=%v", v1, err)
	}
}

func TestFXLinksBitAndWrite(t *testing.T) {
	req := buildBitWriteRequest(0, 0, 1, true, "M", 10, []bool{true})
	if req[0] != ctrlENQ || string(req[1:16]) != "00FFBW0M0010011" {
		t.Fatalf("BW frame = %q", string(req))
	}
	if err := parseWriteResponse([]byte("\x0600FF")); err != nil {
		t.Fatal(err)
	}
	if err := parseWriteResponse([]byte("\x1500FF02")); err == nil {
		t.Fatal("expected NAK error")
	}
}
