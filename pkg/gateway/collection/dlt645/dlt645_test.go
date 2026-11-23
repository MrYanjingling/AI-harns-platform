package dlt645

import (
	"bytes"
	"testing"

	"harns-platform/pkg/gateway/collection/meterlink"
)

func TestParseAddress2007(t *testing.T) {
	address, err := parseAddress("12345678#02-01-01-00", Model2007)
	if err != nil {
		t.Fatal(err)
	}
	if address.Meter != 12345678 {
		t.Fatalf("meter = %d", address.Meter)
	}
	if !bytes.Equal(address.DI, []byte{0x00, 0x01, 0x01, 0x02}) {
		t.Fatalf("DI = %x", address.DI)
	}
	if address.Value() != 0x02010100 {
		t.Fatalf("DI value = %08X", address.Value())
	}
}

func TestParseAddress1997(t *testing.T) {
	address, err := parseAddress("12345678#00-01", Model1997)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(address.DI, []byte{0x01, 0x00}) {
		t.Fatalf("DI = %x", address.DI)
	}
}

func TestDLT645ReadFrame(t *testing.T) {
	address, err := parseAddress("12345678#00-00-00-00", Model2007)
	if err != nil {
		t.Fatal(err)
	}
	request := buildReadFrame(address, 0x11)
	if len(request) != 20 {
		t.Fatalf("request length = %d", len(request))
	}
	if !bytes.Equal(request[:5], []byte{0xFE, 0xFE, 0xFE, 0xFE, 0x68}) {
		t.Fatalf("preamble = %x", request[:5])
	}
	if !bytes.Equal(request[5:11], []byte{0x78, 0x56, 0x34, 0x12, 0x00, 0x00}) {
		t.Fatalf("address = %x", request[5:11])
	}
	if meterlink.ChecksumSum8(request[4:len(request)-2]) != request[len(request)-2] {
		t.Fatalf("checksum mismatch")
	}

	payload := []byte{0x45, 0x23, 0x01}
	data := append(append([]byte{}, address.DI...), payload...)
	response := []byte{0x68}
	response = append(response, meterlink.EncodeBCDAddress(address.Meter, 6)...)
	response = append(response, 0x68, 0x91, byte(len(data)))
	response = append(response, meterlink.Mask(data)...)
	response = append(response, meterlink.ChecksumSum8(response), 0x16)
	got, err := parseReadFrame(response, address, 0x91)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %x", got)
	}
	value, err := decodeDI(address, got, "FLOAT32", Model2007)
	if err != nil {
		t.Fatal(err)
	}
	if value != 123.45 {
		t.Fatalf("value = %v", value)
	}
}

func TestDLT645RejectsBadChecksum(t *testing.T) {
	address, err := parseAddress("12345678#00-00-00-00", Model2007)
	if err != nil {
		t.Fatal(err)
	}
	response := buildReadFrame(address, 0x11)
	response[0] = 0x68
	response[len(response)-2]++
	if _, err := parseReadFrame(response[:len(response)-5], address, 0x91); err == nil {
		t.Fatal("expected invalid frame error")
	}
}
