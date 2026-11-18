package fins

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func TestProtocolDefaultsTransportTCP(t *testing.T) {
	item, err := NewCollection(&CreateReq{Name: "fins"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Transport != TransportTCP {
		t.Fatalf("unexpected transport %q", item.Transport)
	}
	var config ProtocolConfig
	raw, err := item.ConfigJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	if config.Transport != TransportTCP {
		t.Fatalf("unexpected config transport %q", config.Transport)
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		address string
		data    string
		word    byte
		bit     byte
		start   uint16
		length  uint16
		isBit   bool
	}{
		{"D100", "int16", 0x82, 0x02, 100, 1, false},
		{"CIO0", "uint32", 0xB0, 0x30, 0, 2, false},
		{"CIO0.3", "bool", 0xB0, 0x30, 0, 1, true},
		{"EM0W0", "int16", 0xA0, 0x20, 0, 1, false},
		{"E1W0", "int16", 0xA1, 0x21, 0, 1, false},
		{"F0", "bool", 0, 0x09, 0, 1, true},
	}
	for _, test := range tests {
		parsed, err := parseAddress(test.address, test.data, 0)
		if err != nil {
			t.Fatalf("%s: %v", test.address, err)
		}
		if parsed.areaWord != test.word || parsed.areaBit != test.bit || parsed.start != test.start || parsed.length != test.length || parsed.isBit != test.isBit {
			t.Fatalf("%s: unexpected parse %+v", test.address, parsed)
		}
	}
}

func TestParseAddressStringAndOrder(t *testing.T) {
	parsed, err := parseAddress("D0.10H", "string", 0)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.stringLength != 10 || parsed.length != 5 || !parsed.stringHigh {
		t.Fatalf("unexpected string parse %+v", parsed)
	}
	parsed, err = parseAddress("D0#L", "int32", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.order.reverseWords {
		t.Fatalf("expected word order")
	}
	if _, err := parseAddress("D0.10", "string", 0); err == nil {
		t.Fatal("expected missing string direction error")
	}
}

func TestBuildReadFramesMergesContiguousWords(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", DataType: "int16", Address: "D100", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "uint16", Address: "D101", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildReadFrames(variables, 64)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 {
		t.Fatalf("expected one frame, got %d", len(frames))
	}
	frame := frames[0]
	if frame.Start != 100 || frame.Length != 2 || len(frame.Variables) != 2 {
		t.Fatalf("unexpected frame %+v", frame)
	}
	request := frame.Request
	if request[0] != 0x80 || request[2] != 0x02 || request[10] != 0x01 || request[11] != 0x01 {
		t.Fatalf("unexpected request header % X", request)
	}
	if request[12] != 0x82 || request[13] != 0x00 || request[14] != 100 || request[16] != 0x00 || request[17] != 0x02 {
		t.Fatalf("unexpected read request % X", request)
	}
	response := []byte{
		0xC0, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		0x01, 0x01, 0x00, 0x00,
		0x00, 0x0A, 0x00, 0x14,
	}
	payload, err := validateReadResponse(response, frame)
	if err != nil {
		t.Fatal(err)
	}
	values, err := decodeFrame(frame, payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].Value != int16(10) || values[1].Value != uint16(20) {
		t.Fatalf("unexpected values %+v", values)
	}
}

func TestTCPWrapper(t *testing.T) {
	frame := wrapTCP(2, []byte{1, 2, 3})
	if string(frame[:4]) != "FINS" {
		t.Fatalf("unexpected magic %q", frame[:4])
	}
	if frame[7] != 11 {
		t.Fatalf("unexpected length %d", frame[7])
	}
	if frame[11] != 2 {
		t.Fatalf("unexpected command %d", frame[11])
	}
}

func TestWriteRequest(t *testing.T) {
	variable := &collection.Variable{Name: "a", DataType: "int16", Address: "D100", AccessMode: collection.AccessModeWrite}
	request, err := buildWriteRequest(variable, int16(10))
	if err != nil {
		t.Fatal(err)
	}
	if request[10] != 0x01 || request[11] != 0x02 || request[12] != 0x82 || request[18] != 0x00 || request[19] != 0x0A {
		t.Fatalf("unexpected write request % X", request)
	}
}

func TestTCPTransportHandshakeAndRead(t *testing.T) {
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
		header := make([]byte, 16)
		if _, err := io.ReadFull(conn, header); err != nil {
			done <- err
			return
		}
		length := int(binary.BigEndian.Uint32(header[4:8]))
		if length != 12 {
			done <- fmt.Errorf("unexpected handshake length %d", length)
			return
		}
		body := make([]byte, length-8)
		if _, err := io.ReadFull(conn, body); err != nil {
			done <- err
			return
		}
		if _, err := conn.Write(wrapTCP(1, []byte{0, 0, 0, 1, 0, 0, 0, 2})); err != nil {
			done <- err
			return
		}
		header = make([]byte, 16)
		if _, err := io.ReadFull(conn, header); err != nil {
			done <- err
			return
		}
		length = int(binary.BigEndian.Uint32(header[4:8]))
		body = make([]byte, length-8)
		if _, err := io.ReadFull(conn, body); err != nil {
			done <- err
			return
		}
		if body[0] != 0x80 || body[4] != 2 || body[7] != 1 {
			done <- fmt.Errorf("unexpected nodes % X", body)
			return
		}
		finsResponse := []byte{
			0xC0, 0x00, 0x02, 0x00, 0x02, 0x00, 0x00, 0x01, 0x00, body[9],
			0x01, 0x01, 0x00, 0x00, 0x00, 0x0A,
		}
		_, err = conn.Write(wrapTCP(2, finsResponse))
		done <- err
	}()
	client, err := newTCPTransport(context.Background(), listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	address, _ := parseAddress("D100", "int16", 0)
	request := newReadRequest(address.areaCode(), address.start, 0, 1)
	response, err := client.Exchange(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response) != 16 || response[14] != 0x00 || response[15] != 0x0A {
		t.Fatalf("unexpected response % X", response)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
