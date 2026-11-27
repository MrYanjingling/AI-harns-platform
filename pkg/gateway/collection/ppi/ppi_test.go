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

func TestBuildWriteRequestBitAndBytes(t *testing.T) {
	bitAddr, err := mpi.ParseAddress("V0.1", "BOOL")
	if err != nil {
		t.Fatal(err)
	}
	bitFrame, err := buildWriteRequest(2, bitAddr, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	if bitFrame[6] != 0x7c || bitFrame[17] != 0x05 || bitFrame[22] != 0x01 || bitFrame[32] != 0x03 || bitFrame[35] != 0x01 {
		t.Fatalf("unexpected bit write frame: %x", bitFrame)
	}

	f64Addr, err := mpi.ParseAddress("VD2500", "FLOAT64")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := encodeWritePayload(f64Addr, &collection.Variable{DataType: "FLOAT64"}, 3.14159)
	if err != nil {
		t.Fatal(err)
	}
	f64Frame, err := buildWriteRequest(2, f64Addr, payload)
	if err != nil {
		t.Fatal(err)
	}
	if f64Frame[6] != 0x7c || f64Frame[17] != 0x05 || f64Frame[22] != 0x02 || f64Frame[24] != 0x08 || f64Frame[32] != 0x04 {
		t.Fatalf("unexpected float64 write frame: %x", f64Frame)
	}
}

func TestParseWriteResponse(t *testing.T) {
	okFrame := []byte{
		0x68, 0x12, 0x12, 0x68, 0x00, 0x02, 0x08, 0x32, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
		0x00, 0x01, 0x00, 0x00, 0x05, 0x01, 0xff, 0x47, 0x16,
	}
	if err := parseWriteResponse(okFrame, 2); err != nil {
		t.Fatalf("expected ok write response, got %v", err)
	}
	errFrame := append([]byte(nil), okFrame...)
	errFrame[21] = 0x05
	if err := parseWriteResponse(errFrame, 2); err == nil {
		t.Fatal("expected error for return code 0x05")
	}
}

func TestBuildReadBatches(t *testing.T) {
	tags := make([]*tagRef, 0, 24)
	for i := 0; i < 16; i++ {
		addr, _ := mpi.ParseAddress("V"+string(rune('0'+i/8))+"."+string(rune('0'+i%8)), "BOOL")
		tags = append(tags, &tagRef{variable: &collection.Variable{Name: "b", DataType: "BOOL"}, address: addr})
	}
	for i := 0; i < 4; i++ {
		addr, _ := mpi.ParseAddress("VW100", "INT16")
		addr.Start = uint16(100 + i*2)
		tags = append(tags, &tagRef{variable: &collection.Variable{Name: "w", DataType: "INT16"}, address: addr})
	}
	tAddr, _ := mpi.ParseAddress("T1", "UINT16")
	tags = append(tags, &tagRef{variable: &collection.Variable{Name: "t1", DataType: "UINT16"}, address: tAddr})

	batches := buildReadBatches(tags)
	if len(batches) != 3 {
		t.Fatalf("expected 3 batches (T1, V0..V1, VW100..VW106), got %d", len(batches))
	}
	for _, b := range batches {
		if b.merged && b.address.Type != 2 {
			t.Fatalf("merged batch must use byte transport type 2, got %d", b.address.Type)
		}
	}
}
