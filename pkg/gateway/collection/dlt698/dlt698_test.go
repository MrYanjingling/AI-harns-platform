package dlt698

import (
	"bytes"
	"strings"
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
	if len(request) != 25 || request[0] != 0x68 || request[len(request)-1] != 0x16 {
		t.Fatalf("unexpected frame envelope: %x", request)
	}
	// Verify length(0x0017), control(0x43), reversed meter address, HCS, APDU with trailing TimeTag(0x00), and FCS.
	header := []byte{0x17, 0x00, 0x43, 0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	body := appendFCS(append([]byte{}, header...), header)
	body = append(body, 0x05, 0x01, 0x01, 0x00, 0x00, 0x02, 0x00, 0x00)
	body = appendFCS(body, body)
	want := append(append([]byte{0x68}, body...), 0x16)
	if !bytes.Equal(request, want) {
		t.Fatalf("request = %x, want %x", request, want)
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
	if _, err := parseGetResponse(frame, meter, oad, 1); err == nil || !strings.Contains(err.Error(), "result 0x06") {
		t.Fatalf("expected result 0x06 error, got %v", err)
	}
}

func TestParseGetResponseStandardDAR(t *testing.T) {
	meter, err := parseMeterAddress("000000000001")
	if err != nil {
		t.Fatal(err)
	}
	oad, _ := parseOAD("00100200")
	// Standard DL/T 698.45 Get-Result CHOICE dar [0] followed by DAR=0x06, FollowReport=0x00, TimeTag=0x00.
	apdu := []byte{0x85, 0x01, 0x01, 0x00, 0x10, 0x02, 0x00, 0x00, 0x06, 0x00, 0x00}
	frame := responseFrame(meter, 0, apdu)
	if _, err := parseGetResponse(frame, meter, oad, 1); err == nil || !strings.Contains(err.Error(), "dar 0x06") {
		t.Fatalf("expected dar 0x06 error, got %v", err)
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
