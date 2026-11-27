package omroncip

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseAddress(t *testing.T) {
	parsed, err := parseAddress("D100", "int16")
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.segments) != 1 || parsed.segments[0].symbol != "D100" || parsed.typeCode != 0xC3 {
		t.Fatalf("unexpected address %+v", parsed)
	}
	parsed, err = parseAddress("MyArray[3]", "int16")
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.segments) != 1 || parsed.segments[0].symbol != "MyArray" || parsed.segments[0].index != 3 {
		t.Fatalf("unexpected array address %+v", parsed)
	}
	parsed, err = parseAddress("MyStruct.Member", "int16")
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.segments) != 2 || parsed.segments[1].symbol != "Member" {
		t.Fatalf("unexpected struct address %+v", parsed)
	}
	parsed, err = parseAddress("CIO0.0", "bool")
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.bitSet || parsed.bit != 0 || len(parsed.segments) != 1 || parsed.segments[0].symbol != "CIO0" {
		t.Fatalf("unexpected bit address %+v", parsed)
	}
}

func TestReadRequest(t *testing.T) {
	address, err := parseAddress("D100", "int16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{0x4C, 0x03, 0x91, 0x04, 'D', '1', '0', '0', 1, 0}
	if !bytes.Equal(request, expected) {
		t.Fatalf("unexpected request % X", request)
	}
}

func TestReadResponse(t *testing.T) {
	address, err := parseAddress("D100", "int16")
	if err != nil {
		t.Fatal(err)
	}
	response := []byte{0xCC, 0x00, 0x00, 0x00, 0xC3, 0x00, 0x01, 0x00, 0x0A, 0x00}
	data, _, err := parseReadResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeValue(data, "int16", address)
	if err != nil {
		t.Fatal(err)
	}
	if value != int16(10) {
		t.Fatalf("unexpected value %v", value)
	}
}

func TestReadResponseWithoutElementCount(t *testing.T) {
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
	}
	for _, tc := range cases {
		address, err := parseAddress("D0", tc.dataType)
		if err != nil {
			t.Fatal(err)
		}
		data, dataType, err := parseReadResponse(tc.response)
		if err != nil {
			t.Fatal(err)
		}
		value, err := decodeValue(data, tc.dataType, address)
		if err != nil {
			t.Fatal(err)
		}
		if dataType != address.typeCode {
			t.Fatalf("%s: unexpected data type %X", tc.dataType, dataType)
		}
		if !reflect.DeepEqual(value, tc.value) {
			t.Fatalf("%s: unexpected value %v", tc.dataType, value)
		}
	}
}

func TestWriteRequest(t *testing.T) {
	address, err := parseAddress("D100", "int16")
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
	expected := []byte{0x4D, 0x03, 0x91, 0x04, 'D', '1', '0', '0', 0xC3, 0x00, 1, 0, 0x0A, 0x00}
	if !bytes.Equal(request, expected) {
		t.Fatalf("unexpected request % X", request)
	}
}

func TestParseRRData(t *testing.T) {
	payload := []byte{
		0, 0, 0, 0,
		10, 0,
		1, 0,
		0xB2, 0x00, 4, 0,
		0xCC, 0, 0, 0,
	}
	data, err := parseRRData(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0xCC, 0, 0, 0}) {
		t.Fatalf("unexpected data % X", data)
	}
}

func TestCIPClientRegisterAndRead(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		header := make([]byte, 24)
		if _, err := io.ReadFull(conn, header); err != nil {
			done <- err
			return
		}
		if binary.LittleEndian.Uint16(header[:2]) != 0x0065 {
			done <- fmt.Errorf("unexpected register command")
			return
		}
		length := int(binary.LittleEndian.Uint16(header[2:]))
		body := make([]byte, length)
		if _, err := io.ReadFull(conn, body); err != nil {
			done <- err
			return
		}
		if _, err := conn.Write(cipResponse(0x0065, 0x1234, []byte{1, 0, 0, 0})); err != nil {
			done <- err
			return
		}
		header = make([]byte, 24)
		if _, err := io.ReadFull(conn, header); err != nil {
			done <- err
			return
		}
		if binary.LittleEndian.Uint16(header[:2]) != 0x006F || binary.LittleEndian.Uint32(header[4:8]) != 0x1234 {
			done <- fmt.Errorf("unexpected send rr data header")
			return
		}
		length = int(binary.LittleEndian.Uint16(header[2:]))
		body = make([]byte, length)
		if _, err := io.ReadFull(conn, body); err != nil {
			done <- err
			return
		}
		cip := []byte{0xCC, 0, 0, 0, 0xC3, 0, 1, 0, 0x0A, 0}
		payload := []byte{0, 0, 0, 0, 10, 0, 2, 0, 0, 0, 0, 0, 0xB2, 0, byte(len(cip)), 0}
		payload = append(payload, cip...)
		_, err = conn.Write(cipResponse(0x006F, 0x1234, payload))
		done <- err
	}()
	client, err := newCIPClient(context.Background(), listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	address, err := parseAddress("D100", "int16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address)
	if err != nil {
		t.Fatal(err)
	}
	response, _, _, err := client.Exchange(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	data, _, err := parseReadResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeValue(data, "int16", address)
	if err != nil {
		t.Fatal(err)
	}
	if value != int16(10) {
		t.Fatalf("unexpected value %v", value)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func cipResponse(command uint16, session uint32, payload []byte) []byte {
	response := make([]byte, 24+len(payload))
	binary.LittleEndian.PutUint16(response[0:], command)
	binary.LittleEndian.PutUint16(response[2:], uint16(len(payload)))
	binary.LittleEndian.PutUint32(response[4:], session)
	copy(response[24:], payload)
	return response
}

func TestMultipleRequestAndResponse(t *testing.T) {
	first := []byte{0x4C, 0x03, 0x91, 0x04, 'D', '1', '0', '0', 1, 0}
	second := []byte{0x4C, 0x03, 0x91, 0x04, 'D', '1', '0', '1', 1, 0}
	request, err := buildMultipleRequest([][]byte{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if request[0] != 0x0A || request[6] != 2 || request[7] != 0 {
		t.Fatalf("unexpected request % X", request)
	}
	if binary.LittleEndian.Uint16(request[8:10]) != 6 || binary.LittleEndian.Uint16(request[10:12]) != 16 {
		t.Fatalf("unexpected offsets % X", request[8:12])
	}
	response1 := []byte{0xCC, 0, 0, 0, 0xC3, 0, 1, 0, 0x0A, 0}
	response2 := []byte{0xCC, 0, 0, 0, 0xC3, 0, 1, 0, 0x14, 0}
	data := []byte{2, 0, 6, 0, 16, 0}
	data = append(data, response1...)
	data = append(data, response2...)
	response := append([]byte{0x8A, 0, 0, 0}, data...)
	items, err := parseMultipleResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || !bytes.Equal(items[0], response1) || !bytes.Equal(items[1], response2) {
		t.Fatalf("unexpected items % X", items)
	}
}

func TestConcurrencyDefaultsAndConfig(t *testing.T) {
	item, err := NewCollection(&CreateReq{Name: "omron-cip"})
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
	if !strings.Contains(string(raw), `"concurrency":1`) {
		t.Fatalf("unexpected config %s", raw)
	}
}

func TestEncapsulationStatusText(t *testing.T) {
	if got := encapsulationStatusText(1); got != "invalid or unsupported encapsulation command" {
		t.Fatalf("unexpected status text: %s", got)
	}
	if got := encapsulationCommandName(0x0065); got != "RegisterSession (0x0065)" {
		t.Fatalf("unexpected command text: %s", got)
	}
}
