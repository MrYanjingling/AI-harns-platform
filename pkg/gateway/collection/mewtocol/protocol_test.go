package mewtocol

import (
	"math"
	"testing"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		address  string
		dataType string
		bits     uint8
		area     string
		word     int
		bit      int
		words    int
		isBit    bool
	}{
		{"X0.0", "BOOL", 0, "X", 0, 0, 1, true},
		{"X1.5", "BIT", 0, "X", 15, 5, 1, true},
		{"R10", "BOOL", 3, "R", 103, 3, 1, true},
		{"DT100", "INT16", 0, "DT", 100, 0, 1, false},
		{"DT100", "FLOAT32", 0, "DT", 100, 0, 2, false},
		{"DT100", "FLOAT64", 0, "DT", 100, 0, 4, false},
		{"DT100.10", "STRING", 0, "DT", 100, 0, 5, false},
		{"LD20", "UINT32", 0, "LD", 20, 0, 2, false},
		{"FL5", "INT32", 0, "FL", 5, 0, 2, false},
		{"IX3", "INT16", 0, "IX", 3, 0, 1, false},
	}
	for _, tt := range tests {
		address, err := parseAddress(tt.address, tt.dataType, tt.bits)
		if err != nil {
			t.Fatalf("%s: %v", tt.address, err)
		}
		if address.Area != tt.area || address.Word != tt.word || address.Bit != tt.bit ||
			address.Words != tt.words || address.IsBit != tt.isBit {
			t.Fatalf("%s: got %+v", tt.address, address)
		}
	}
}

func TestBuildReadRequests(t *testing.T) {
	address, err := parseAddress("DT0", "INT16", 0)
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildReadRequest(1, address)
	if err != nil {
		t.Fatal(err)
	}
	expected := "%01#RDD0000000000"
	if string(request[:len(expected)]) != expected {
		t.Fatalf("unexpected request %q", request)
	}
	if request[len(request)-1] != '\r' {
		t.Fatal("request must end with CR")
	}

	address, _ = parseAddress("X1.5", "BOOL", 0)
	request, err = buildReadRequest(1, address)
	if err != nil {
		t.Fatal(err)
	}
	expected = "%01#RCCX00150015"
	if string(request[:len(expected)]) != expected {
		t.Fatalf("unexpected request %q", request)
	}
}

func TestParseResponse(t *testing.T) {
	raw := []byte("%01RD000A" + "43" + "\r")
	command, payload, err := parseResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if command != "RD" || len(payload) != 2 || payload[0] != 0 || payload[1] != 10 {
		t.Fatalf("unexpected response %q % X", command, payload)
	}
}

func TestDecodeAndEncodeValue(t *testing.T) {
	address, _ := parseAddress("DT0", "FLOAT32", 0)
	value, err := decodeValue(address, []byte{0x41, 0x20, 0x00, 0x00})
	if err != nil {
		t.Fatal(err)
	}
	if value != float32(10) {
		t.Fatalf("unexpected value %v", value)
	}
	data, err := encodeValue(address, float64(10))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4 || binaryBigEndianUint32(data) != math.Float32bits(10) {
		t.Fatalf("unexpected data % X", data)
	}

	bit, _ := parseAddress("X0.3", "BOOL", 0)
	value, err = decodeValue(bit, []byte{0, 8})
	if err != nil {
		t.Fatal(err)
	}
	if value != true {
		t.Fatalf("unexpected bit value %v", value)
	}
}

func binaryBigEndianUint32(data []byte) uint32 {
	return uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
}

func TestBuildWriteRequests(t *testing.T) {
	address, _ := parseAddress("DT0", "FLOAT32", 0)
	data, err := encodeValue(address, float64(10))
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildWriteRequest(1, address, data)
	if err != nil {
		t.Fatal(err)
	}
	if string(request[:len("%01#WDD0000000001")]) != "%01#WDD0000000001" {
		t.Fatalf("unexpected write request %q", request)
	}

	bit, _ := parseAddress("X0.0", "BOOL", 0)
	data, err = encodeValue(bit, true)
	if err != nil {
		t.Fatal(err)
	}
	request, err = buildWriteRequest(1, bit, data)
	if err != nil {
		t.Fatal(err)
	}
	if string(request[:len("%01#WCSX00001")]) != "%01#WCSX00001" {
		t.Fatalf("unexpected bit write request %q", request)
	}
}
