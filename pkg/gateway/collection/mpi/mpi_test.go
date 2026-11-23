package mpi

import (
	"bytes"
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestParseAddress(t *testing.T) {
	address, err := parseAddress("DB1.DBB0", "UINT16")
	if err != nil {
		t.Fatal(err)
	}
	if address.Area != areaDB || address.DB != 1 || address.Start != 0 || address.Length != 2 || address.Type != typeByte {
		t.Fatalf("address = %+v", address)
	}
	bit, err := parseAddress("M0.3", "BOOL")
	if err != nil {
		t.Fatal(err)
	}
	if bit.Area != areaM || bit.Start != 0 || bit.Bit != 3 || bit.Type != typeBool {
		t.Fatalf("bit = %+v", bit)
	}
	timer, err := parseAddress("T4", "UINT16")
	if err != nil {
		t.Fatal(err)
	}
	if timer.Area != areaT || timer.Start != 4 {
		t.Fatalf("timer = %+v", timer)
	}
}

func TestBuildReadRequest(t *testing.T) {
	address, err := parseAddress("DB1.DBB0", "UINT16")
	if err != nil {
		t.Fatal(err)
	}
	client := &mpiClient{}
	request := client.buildReadRequest(address)
	want := []byte{
		0x00, 0x0c, 0x03, 0x03, 0xf1, 0x01,
		0x32, 0x01, 0x00, 0x00, 0x33, 0x01, 0x00, 0x0e, 0x00, 0x00,
		0x04, 0x01, 0x12, 0x0a, 0x10,
		0x10, 0x02, 0x00, 0x02, 0x00, 0x01, 0x84, 0x00, 0x00, 0x00,
		0x10, 0x03, 0x00,
	}
	var sum byte
	for _, value := range want[:len(want)-1] {
		sum ^= value
	}
	want[len(want)-1] = sum
	if !bytes.Equal(request, want) {
		t.Fatalf("request = %x", request)
	}
}

func TestParseReadResult(t *testing.T) {
	frame := make([]byte, 0x16)
	frame[0], frame[1] = 0x00, 0x0c
	frame[5] = 0x01
	frame[0x14] = 0xff
	frame = append(frame, 0x00, 0x02, 0x00, 0x2a)
	frame = append(frame, 0x10, 0x03, 0x00)
	data, err := parseReadResult(frame)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x00, 0x2a}) {
		t.Fatalf("data = %x", data)
	}
	value, err := decodeValue(data, parsedAddress{Length: 2}, &collection.Variable{DataType: "UINT16"})
	if err != nil {
		t.Fatal(err)
	}
	if value != uint16(42) {
		t.Fatalf("value = %v", value)
	}
}
