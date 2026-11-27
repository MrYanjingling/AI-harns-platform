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

func TestEncodePointValueNumericTypes(t *testing.T) {
	// Analog Present_Value accepts int16/int32/uint16/uint32 and encodes as 4-byte REAL (tag 4).
	for _, val := range []any{int16(12), int32(345), uint16(67), uint32(890)} {
		got, err := encodePointValue(2, propertyPresentValue, val, "INT16")
		if err != nil {
			t.Fatalf("analog encode %T failed: %v", val, err)
		}
		if len(got) != 5 || got[0] != 0x44 {
			t.Fatalf("analog encode %T: got %X, want 5-byte REAL tag 0x44", val, got)
		}
	}

	// Multi-State Present_Value accepts non-negative int16/int32 and rejects negative values.
	got1, err := encodePointValue(19, propertyPresentValue, int16(3), "INT16")
	if err != nil || hex.EncodeToString(got1) != "2103" {
		t.Fatalf("MSV int16(3): got %X err=%v", got1, err)
	}
	got2, err := encodePointValue(19, propertyPresentValue, int32(300), "INT32")
	if err != nil || hex.EncodeToString(got2) != "22012c" {
		t.Fatalf("MSV int32(300): got %X err=%v", got2, err)
	}
	if _, err := encodePointValue(19, propertyPresentValue, int16(-1), "INT16"); err == nil {
		t.Fatalf("expected error for negative MSV value")
	}
}
