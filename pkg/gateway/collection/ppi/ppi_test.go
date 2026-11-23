package ppi

import (
	"bytes"
	"testing"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/mpi"
)

func TestBuildReadRequest(t *testing.T) {
	address, err := mpi.ParseAddress("DB1.DBB0", "UINT16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildReadRequest(2, address)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		0x68, 0x1b, 0x1b, 0x68, 0x02, 0x00, 0x6c,
		0x32, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0x00, 0x00,
		0x04, 0x01, 0x12, 0x0a, 0x10,
		0x02, 0x00, 0x02, 0x00, 0x01, 0x84, 0x00, 0x00, 0x00,
		0x69, 0x16,
	}
	if !bytes.Equal(request, want) {
		t.Fatalf("request = %x", request)
	}
}

func TestParseReadResponse(t *testing.T) {
	frame := []byte{
		0x68, 0x17, 0x17, 0x68, 0x00, 0x02, 0x08, 0x32, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
		0x00, 0x06, 0x00, 0x00, 0x04, 0x01, 0xff, 0x04, 0x00, 0x10, 0x01, 0x00, 0x60, 0x16,
	}
	data, err := parseReadResponse(frame, 2)
	if err != nil {
		t.Fatal(err)
	}
	address, _ := mpi.ParseAddress("DB1.DBB0", "UINT16")
	value, err := mpi.DecodeValue(data, address, &collection.Variable{DataType: "UINT16"})
	if err != nil {
		t.Fatal(err)
	}
	if value != uint16(256) {
		t.Fatalf("value = %v", value)
	}
}

func TestPPIAck(t *testing.T) {
	ack := buildAck(2)
	want := []byte{0x10, 0x02, 0x00, 0x5c, 0x5e, 0x16}
	if !bytes.Equal(ack, want) {
		t.Fatalf("ack = %x", ack)
	}
}
