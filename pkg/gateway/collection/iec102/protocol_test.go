package iec102

import (
	"bytes"
	"testing"
)

func TestParseAddress(t *testing.T) {
	ra, ioa, err := parseAddress("10.255")
	if err != nil || ra != 10 || ioa != 255 {
		t.Fatalf("unexpected address %d.%d %v", ra, ioa, err)
	}
	if _, _, err := parseAddress("1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestVariableFrameRoundTrip(t *testing.T) {
	asdu := []byte{1, 1, 3, 0, 1, 2, 3, 1}
	frame := buildVariableFrame(0x53, 1, asdu)
	got, err := readLinkFrame(bytes.NewReader(frame))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, asdu) {
		t.Fatalf("unexpected ASDU % X", got)
	}
}

func TestParseASDU(t *testing.T) {
	values, err := parseASDU([]byte{1, 1, 3, 0, 1, 2, 3, 1})
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := values["2.3"]; !ok || value != true {
		t.Fatalf("unexpected values %+v", values)
	}
}
