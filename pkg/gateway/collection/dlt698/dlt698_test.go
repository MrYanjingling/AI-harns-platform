package dlt698

import (
	"bytes"
	"testing"
)

func responseFrame(meter []byte, client byte, apdu []byte) []byte {
	address := buildAddressArea(meter, client)
	total := 1 + 2 + 1 + len(address) + 2 + len(apdu) + 2 + 1
	length := uint16(total - 2)
	header := []byte{byte(length), byte(length >> 8), controlGetResponse}
	header = append(header, address...)
	body := appendFCS(append([]byte{}, header...), header)
	body = append(body, apdu...)
	body = appendFCS(body, body)
	frame := append([]byte{0x68}, body...)
	return append(frame, 0x16)
}

func TestBuildGetRequest(t *testing.T) {
	meter, err := parseMeterAddress("000000000001")
	if err != nil {
		t.Fatal(err)
	}
	oad, err := parseOAD("00000200")
	if err != nil {
		t.Fatal(err)
	}
	request := buildGetRequest(meter, 0, oad, 1)
	want := []byte{0x68, 0x16, 0x00, 0x43, 0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x36, 0xB3, 0x05, 0x01, 0x01, 0x00, 0x00, 0x02, 0x00, 0x9B, 0x7D, 0x16}
	if !bytes.Equal(request, want) {
		t.Fatalf("request = %x", request)
	}
}

func TestParseGetResponse(t *testing.T) {
	meter, err := parseMeterAddress("000000000001")
	if err != nil {
		t.Fatal(err)
	}
	oad, err := parseOAD("00000200")
	if err != nil {
		t.Fatal(err)
	}
	apdu := []byte{0x85, 0x01, 0x01, 0x00, 0x00, 0x02, 0x00, 0x01, 0x12, 0x00, 0x2A}
	frame := responseFrame(meter, 0, apdu)
	data, err := parseGetResponse(frame, meter, oad, 1)
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeData(data, "UINT32")
	if err != nil {
		t.Fatal(err)
	}
	if value != uint64(42) {
		t.Fatalf("value = %v", value)
	}
}

func TestParseGetResponseError(t *testing.T) {
	meter, err := parseMeterAddress("000000000001")
	if err != nil {
		t.Fatal(err)
	}
	oad, _ := parseOAD("00000200")
	apdu := []byte{0x85, 0x01, 0x01, 0x00, 0x00, 0x02, 0x00, 0x06}
	frame := responseFrame(meter, 0, apdu)
	if _, err := parseGetResponse(frame, meter, oad, 1); err == nil {
		t.Fatal("expected result error")
	}
}

func TestParseRealSimulatorErrorFrame(t *testing.T) {
	meter, _ := parseMeterAddress("000000000001")
	oad, _ := parseOAD("00000200")
	frame := []byte{0x68, 0x17, 0x00, 0x83, 0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x92, 0xD4, 0x85, 0x01, 0x01, 0x00, 0x00, 0x02, 0x00, 0x06, 0x8B, 0x74, 0x16}
	if _, err := parseGetResponse(frame, meter, oad, 1); err == nil {
		t.Fatal("expected result error")
	}
}

func TestDecodeArray(t *testing.T) {
	value, err := decodeData([]byte{0x01, 0x02, 0x12, 0x00, 0x01, 0x12, 0x00, 0x02}, "UINT32")
	if err != nil {
		t.Fatal(err)
	}
	if value != uint64(1) {
		t.Fatalf("value = %v", value)
	}
}
