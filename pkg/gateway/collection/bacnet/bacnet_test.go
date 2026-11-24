package bacnet

import (
	"encoding/hex"
	"testing"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		address string
		point   Point
		wantErr bool
	}{
		{"AI0", Point{0, 0, 85}, false},
		{"AI12.Present_Value", Point{0, 12, 85}, false},
		{"AV3.Object_Name", Point{2, 3, 77}, false},
		{"BI7.custom.85", Point{3, 7, 85}, false},
		{"DEV0.Device_Type", Point{8, 0, 31}, false},
		{"MSV9.Units", Point{19, 9, 117}, false},
		{"DEV0", Point{}, true},
		{"AI0.Unknown", Point{}, true},
		{"AI0.custom.9999999", Point{}, true},
		{"XX1", Point{}, true},
	}
	for _, test := range tests {
		point, err := ParseAddress(test.address)
		if (err != nil) != test.wantErr {
			t.Fatalf("%s: err=%v", test.address, err)
		}
		if err == nil && point != test.point {
			t.Fatalf("%s: got %+v", test.address, point)
		}
	}
}

func TestReadRequest(t *testing.T) {
	got := wrapBVLC(append([]byte{0x01, 0x04}, readRequest(0, 0, 85, 1)...))
	want := "810a001101040005010c0c000000001955"
	if hex.EncodeToString(got) != want {
		t.Fatalf("got %X want %s", got, want)
	}
}

func TestParseReadResponse(t *testing.T) {
	apdu, _ := hex.DecodeString("30010c0c0000000019553e443fc000003f")
	value, err := parseReadResponse(apdu)
	if err != nil {
		t.Fatal(err)
	}
	number, ok := value.(float64)
	if !ok || number != 1.5 {
		t.Fatalf("value: %#v", value)
	}
}

func TestWriteRequest(t *testing.T) {
	apdu, err := writeRequest(0, 0, 85, 1, float32(1.5), "FLOAT")
	if err != nil {
		t.Fatal(err)
	}
	want := "0005010f0c0000000019553e443fc000003f"
	if hex.EncodeToString(apdu) != want {
		t.Fatalf("got %X want %s", apdu, want)
	}
}
