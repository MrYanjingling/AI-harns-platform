package cjt188

import (
	"bytes"
	"testing"

	"harns-platform/pkg/gateway/collection/meterlink"
)

func TestParseAddress(t *testing.T) {
	address, err := parseAddress("10#0000000001#901F")
	if err != nil {
		t.Fatal(err)
	}
	if address.Type != 0x10 || address.Meter != 1 {
		t.Fatalf("type=%X meter=%d", address.Type, address.Meter)
	}
	if !bytes.Equal(address.DI, []byte{0x1F, 0x90}) {
		t.Fatalf("DI = %x", address.DI)
	}
}

func TestCJT188ReadFrame(t *testing.T) {
	address, err := parseAddress("10#0000000001#901F")
	if err != nil {
		t.Fatal(err)
	}
	request := buildReadFrame(address, 0x01, true)
	if len(request) != 16 {
		t.Fatalf("request length = %d", len(request))
	}
	if !bytes.Equal(request[2:9], []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) {
		t.Fatalf("address = %x", request[2:9])
	}
	if meterlink.ChecksumSum8(request[:len(request)-2]) != request[len(request)-2] {
		t.Fatalf("checksum mismatch")
	}

	payload := []byte{0x34, 0x12}
	data := append(append([]byte{}, address.DI...), 0x00)
	data = append(data, payload...)
	response := []byte{0x68, address.Type}
	response = append(response, meterlink.EncodeBCDAddress(address.Meter, 7)...)
	response = append(response, 0x81, byte(len(data)))
	response = append(response, data...)
	response = append(response, meterlink.ChecksumSum8(response), 0x16)
	got, err := parseReadFrame(response, address, 0x81, true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %x", got)
	}
	value, err := decodeDI(address, got, "FLOAT32")
	if err != nil {
		t.Fatal(err)
	}
	if value != 12.34 {
		t.Fatalf("value = %v", value)
	}
}

func TestCJT188RejectsAddressMismatch(t *testing.T) {
	address, err := parseAddress("10#0000000001#901F")
	if err != nil {
		t.Fatal(err)
	}
	response := buildReadFrame(address, 0x01, true)
	response[9] = 0x81
	if _, err := parseReadFrame(response, address, 0x81, true); err == nil {
		t.Fatal("expected length error")
	}
}
func TestCJT1882004WithoutSER(t *testing.T) {
	address, err := parseAddress("10#0000000001#901F")
	if err != nil {
		t.Fatal(err)
	}
	request := buildReadFrame(address, 0x01, false)
	if len(request) != 15 {
		t.Fatalf("request length = %d", len(request))
	}
	payload := []byte{0x34, 0x12}
	data := append(append([]byte{}, address.DI...), payload...)
	response := []byte{0x68, address.Type}
	response = append(response, meterlink.EncodeBCDAddress(address.Meter, 7)...)
	response = append(response, 0x81, byte(len(data)))
	response = append(response, data...)
	response = append(response, meterlink.ChecksumSum8(response), 0x16)
	got, err := parseReadFrame(response, address, 0x81, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %x", got)
	}
}
