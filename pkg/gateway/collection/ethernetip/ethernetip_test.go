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
		index    byte
		bit      uint8
		bitSet   bool
		typeCode uint16
	}{
		{"Tag", "int16", []string{"Tag"}, 0, 0, false, 0xC3},
		{"Program:MainProgram.Counter", "int32", []string{"Program:MainProgram", "Counter"}, 0, 0, false, 0xC4},
		{"MyArray[3]", "int16", []string{"MyArray"}, 3, 0, false, 0xC3},
		{"Flags.5", "bool", []string{"Flags"}, 0, 5, true, 0xC1},
		{"Analog", "REAL", []string{"Analog"}, 0, 0, false, 0xCA},
	}
	for _, tc := range cases {
		parsed, err := parseAddress(tc.address, tc.dataType)
		if err != nil {
			t.Fatalf("%s: %v", tc.address, err)
		}
		if len(parsed.segments) != len(tc.segments) {
			t.Fatalf("%s: unexpected segments %+v", tc.address, parsed.segments)
		}
		for index, symbol := range tc.segments {
			if parsed.segments[index].symbol != symbol {
				t.Fatalf("%s: unexpected symbol %q", tc.address, parsed.segments[index].symbol)
			}
		}
		if parsed.segments[len(parsed.segments)-1].index != tc.index {
			t.Fatalf("%s: unexpected index", tc.address)
		}
		if parsed.bit != tc.bit || parsed.bitSet != tc.bitSet {
			t.Fatalf("%s: unexpected bit %+v", tc.address, parsed)
		}
		if parsed.typeCode != tc.typeCode {
			t.Fatalf("%s: unexpected type code %X", tc.address, parsed.typeCode)
		}
	}
}

func TestParseAddressRejectsInvalid(t *testing.T) {
	for _, value := range []string{"", ".Tag", "Tag.", "Tag..Member", "Array[-1]", "Array[256]"} {
		if _, err := parseAddress(value, "int16"); err == nil {
			t.Fatalf("%q should be rejected", value)
		}
	}
}

func TestReadRequestPath(t *testing.T) {
	address, err := parseAddress("Program:MainProgram.Counter", "int32")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address)
	if err != nil {
		t.Fatal(err)
	}
	if request[0] != 0x4C {
		t.Fatalf("unexpected service %X", request[0])
	}
	if !bytes.Contains(request, []byte("Program:MainProgram")) || !bytes.Contains(request, []byte("Counter")) {
		t.Fatalf("unexpected path % X", request)
	}
}

func TestWriteRequest(t *testing.T) {
	address, err := parseAddress("Counter", "int16")
	if err != nil {
		t.Fatal(err)
	}
	data, err := encodeValue(int16(10), "int16", address)
	if err != nil {
		t.Fatal(err)
	}
	request, err := writeRequest(address, data)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{0x4D, 0x05, 0x91, 0x07, 'C', 'o', 'u', 'n', 't', 'e', 'r', 0, 0xC3, 0, 1, 0, 0x0A, 0}
	if !bytes.Equal(request, expected) {
		t.Fatalf("unexpected request % X", request)
	}
}

func TestReadResponseWithAndWithoutCount(t *testing.T) {
	cases := []struct {
		dataType string
		value    any
		response []byte
	}{
		{"bool", true, []byte{0xCC, 0, 0, 0, 0xC1, 0, 1, 0, 1}},
		{"int16", int16(75), []byte{0xCC, 0, 0, 0, 0xC3, 0, 0x4B, 0}},
		{"int32", int32(296), []byte{0xCC, 0, 0, 0, 0xC4, 0, 0x28, 1, 0, 0}},
		{"float32", float32(608.0999755859375), []byte{0xCC, 0, 0, 0, 0xCA, 0, 0x66, 6, 0x18, 0x44}},
		{"float64", float64(1000.1), []byte{0xCC, 0, 0, 0, 0xCB, 0, 0xCD, 0xCC, 0xCC, 0xCC, 0xCC, 0x40, 0x8F, 0x40}},
	}
	for _, tc := range cases {
		address, err := parseAddress("Tag", tc.dataType)
		if err != nil {
			t.Fatal(err)
		}
		data, dataType, err := parseReadResponse(tc.response)
		if err != nil {
			t.Fatalf("%s: %v", tc.dataType, err)
		}
		value, err := decodeValue(data, tc.dataType, address)
		if err != nil {
			t.Fatalf("%s: %v", tc.dataType, err)
		}
		if dataType != address.typeCode {
			t.Fatalf("%s: unexpected data type %X", tc.dataType, dataType)
		}
		if !reflect.DeepEqual(value, tc.value) {
			t.Fatalf("%s: unexpected value %v", tc.dataType, value)
		}
	}
}

func TestDecodeBitAddress(t *testing.T) {
	address, err := parseAddress("Flags.5", "bool")
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeValue([]byte{0x20, 0, 0, 0}, "bool", address)
	if err != nil {
		t.Fatal(err)
	}
	if value != true {
		t.Fatalf("unexpected value %v", value)
	}
}

func TestMultipleRequestAndResponse(t *testing.T) {
	first := []byte{0x4C, 0x01, 0x91, 0x01, 'A', 0, 1, 0}
	second := []byte{0x4C, 0x01, 0x91, 0x01, 'B', 0, 1, 0}
	request, err := buildMultipleRequest([][]byte{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if request[0] != 0x0A || request[6] != 2 || request[7] != 0 {
		t.Fatalf("unexpected request % X", request)
	}
	response1 := []byte{0xCC, 0, 0, 0, 0xC3, 0, 1, 0, 0x0A, 0}
	response2 := []byte{0xCC, 0, 0, 0, 0xC3, 0, 1, 0, 0x14, 0}
	data := []byte{2, 0, 6, 0, 16, 0}
	data = append(data, response1...)
	data = append(data, response2...)
	items, err := parseMultipleResponse(append([]byte{0x8A, 0, 0, 0}, data...))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || !bytes.Equal(items[0], response1) || !bytes.Equal(items[1], response2) {
		t.Fatalf("unexpected items % X", items)
	}
}

func TestConcurrencyDefaultsAndConfig(t *testing.T) {
	item, err := NewCollection(&CreateReq{Name: "ethernet-ip"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Concurrency != DefaultConcurrency {
		t.Fatalf("unexpected concurrency %d", item.Concurrency)
	}
	raw, err := item.ConfigJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"concurrency":1`) || !strings.Contains(string(raw), `"port":44818`) {
		t.Fatalf("unexpected config %s", raw)
	}
}
