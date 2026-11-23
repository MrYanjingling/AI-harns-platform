package ethernetip

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestParseAddress(t *testing.T) {
	cases := []struct {
		address  string
		dataType string
		segments []string
		typeCode uint16
	}{
		{"Tag", "int16", []string{"Tag"}, 0xC3},
		{"Program:MainProgram.Counter", "int32", []string{"Program:MainProgram", "Counter"}, 0xC4},
		{"Flags", "BOOL", []string{"Flags"}, 0xC1},
		{"Analog", "REAL", []string{"Analog"}, 0xCA},
		{"Text", "STRING", []string{"Text"}, 0xD0},
	}
	for _, tc := range cases {
		parsed, err := parseAddress(tc.address, tc.dataType)
		if err != nil {
			t.Fatalf("%s: %v", tc.address, err)
		}
		if !reflect.DeepEqual(parsed.segments, tc.segments) {
			t.Fatalf("%s: unexpected segments %+v", tc.address, parsed.segments)
		}
		if parsed.typeCode != tc.typeCode {
			t.Fatalf("%s: unexpected type code %X", tc.address, parsed.typeCode)
		}
	}
}

func TestParseAddressRejectsInvalid(t *testing.T) {
	for _, value := range []string{"", ".Tag", "Tag.", "Tag..Member"} {
		if _, err := parseAddress(value, "int16"); err == nil {
			t.Fatalf("%q should be rejected", value)
		}
	}
	if _, err := parseAddress("Tag", "BYTE_ARRAY"); err == nil {
		t.Fatal("BYTE_ARRAY should be rejected")
	}
	long := strings.Repeat("A", 256)
	if _, err := parseAddress(long, "int16"); err == nil {
		t.Fatal("overlong symbol should be rejected")
	}
}

func TestReadRequest(t *testing.T) {
	address, err := parseAddress("Program:MainProgram.Counter", "int32")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(0x1234, address)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(request[:2], []byte{0x34, 0x12}) {
		t.Fatalf("unexpected sequence % X", request[:2])
	}
	if request[2] != 0x4C {
		t.Fatalf("unexpected service %X", request[2])
	}
	if !bytes.Contains(request, []byte("Program:MainProgram")) || !bytes.Contains(request, []byte("Counter")) {
		t.Fatalf("unexpected path % X", request)
	}
	if !bytes.Equal(request[len(request)-2:], []byte{1, 0}) {
		t.Fatalf("unexpected element count % X", request[len(request)-2:])
	}
}

func TestWriteRequest(t *testing.T) {
	address, err := parseAddress("Counter", "int16")
	if err != nil {
		t.Fatal(err)
	}
	data, err := encodeValue(int16(10), "int16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := writeRequest(0x1234, address, data)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{0x34, 0x12, 0x4D, 0x05, 0x91, 0x07, 'C', 'o', 'u', 'n', 't', 'e', 'r', 0, 0xC3, 0, 1, 0, 0x0A, 0}
	if !bytes.Equal(request, expected) {
		t.Fatalf("unexpected request % X", request)
	}
}

func TestWriteRequestStringAndBoolPadding(t *testing.T) {
	address, err := parseAddress("Text", "string")
	if err != nil {
		t.Fatal(err)
	}
	request, err := writeRequest(1, address, []byte("abc"))
	if err != nil {
		t.Fatal(err)
	}
	tail := request[len(request)-10:]
	if !bytes.Equal(tail[:8], []byte{0xD0, 0, 1, 0, 4, 0, 'a', 'b'}) {
		t.Fatalf("unexpected string request % X", request)
	}
	if !bytes.Equal(tail[8:], []byte{'c', 0}) {
		t.Fatalf("unexpected string padding % X", tail)
	}

	boolAddress, err := parseAddress("Flags", "bool")
	if err != nil {
		t.Fatal(err)
	}
	request, err = writeRequest(1, boolAddress, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(request[len(request)-6:], []byte{0xC1, 0, 1, 0, 1, 0}) {
		t.Fatalf("unexpected bool request % X", request[len(request)-6:])
	}
}

func TestParseReadResponse(t *testing.T) {
	cases := []struct {
		dataType string
		value    any
		response []byte
	}{
		{"bool", true, []byte{0xCC, 0, 0, 0, 0xC1, 0, 1}},
		{"int16", int16(75), []byte{0xCC, 0, 0, 0, 0xC3, 0, 0x4B, 0}},
		{"int32", int32(296), []byte{0xCC, 0, 0, 0, 0xC4, 0, 0x28, 1, 0, 0}},
		{"float32", float32(608.0999755859375), []byte{0xCC, 0, 0, 0, 0xCA, 0, 0x66, 6, 0x18, 0x44}},
		{"float64", float64(1000.1), []byte{0xCC, 0, 0, 0, 0xCB, 0, 0xCD, 0xCC, 0xCC, 0xCC, 0xCC, 0x40, 0x8F, 0x40}},
		{"string", "abc", []byte{0xCC, 0, 0, 0, 0xD0, 0, 3, 0, 'a', 'b', 'c', 0}},
	}
	for _, tc := range cases {
		address, err := parseAddress("Tag", tc.dataType)
		if err != nil {
			t.Fatal(err)
		}
		data, err := parseReadResponse(tc.response, address)
		if err != nil {
			t.Fatalf("%s: %v", tc.dataType, err)
		}
		value, err := decodeValue(data, tc.dataType)
		if err != nil {
			t.Fatalf("%s: %v", tc.dataType, err)
		}
		if !reflect.DeepEqual(value, tc.value) {
			t.Fatalf("%s: unexpected value %v", tc.dataType, value)
		}
	}
}

func TestParseReadResponseRejectsTypeMismatch(t *testing.T) {
	address, err := parseAddress("Tag", "int16")
	if err != nil {
		t.Fatal(err)
	}
	response := []byte{0xCC, 0, 0, 0, 0xCA, 0, 0x66, 6, 0x18, 0x44}
	if _, err := parseReadResponse(response, address); err == nil {
		t.Fatal("expected type mismatch")
	}
	if err := parseWriteResponse([]byte{0xCD, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
}

func TestForwardOpenBody(t *testing.T) {
	body := forwardOpenBody()
	if len(body) != 40 {
		t.Fatalf("unexpected body length %d", len(body))
	}
	if body[0] != 0x05 || body[1] != 0xF7 || body[18] != 0x02 {
		t.Fatalf("unexpected time parameters % X", body[:20])
	}
	if !bytes.Equal(body[22:26], []byte{0x80, 0x84, 0x1E, 0}) {
		t.Fatalf("unexpected RPI % X", body[22:26])
	}
	if !bytes.Equal(body[34:], []byte{0xA3, 2, 0x20, 2, 0x24, 1}) {
		t.Fatalf("unexpected transport/path % X", body[34:])
	}
}

func TestConfigDefaults(t *testing.T) {
	item, err := NewCollection(&CreateReq{Name: "ethernet-ip"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Port != DefaultPort {
		t.Fatalf("unexpected port %d", item.Port)
	}
	raw, err := item.ConfigJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"port":44818`) || !strings.Contains(string(raw), `"slot":0`) {
		t.Fatalf("unexpected config %s", raw)
	}
}
