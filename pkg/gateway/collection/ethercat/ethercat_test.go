package ethercat

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func TestParseAMSNetID(t *testing.T) {
	netID, err := parseAMSNetID("192.168.1.10.1.1")
	if err != nil {
		t.Fatal(err)
	}
	expected := [6]byte{192, 168, 1, 10, 1, 1}
	if netID != expected {
		t.Fatalf("unexpected net id %v", netID)
	}
	if formatAMSNetID(netID) != "192.168.1.10.1.1" {
		t.Fatalf("unexpected formatted net id %q", formatAMSNetID(netID))
	}
	if _, err := parseAMSNetID("192.168.1.1"); err == nil {
		t.Fatal("short net id should be rejected")
	}
	if _, err := parseAMSNetID("192.168.1.10.1.999"); err == nil {
		t.Fatal("out of range octet should be rejected")
	}
}

func TestResolveTargetNetID(t *testing.T) {
	netID, err := resolveTargetNetID("192.168.1.10", "")
	if err != nil {
		t.Fatal(err)
	}
	if formatAMSNetID(netID) != "192.168.1.10.1.1" {
		t.Fatalf("unexpected derived net id %q", formatAMSNetID(netID))
	}
	netID, err = resolveTargetNetID("plc.local", "10.0.0.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if formatAMSNetID(netID) != "10.0.0.1.1.1" {
		t.Fatalf("unexpected configured net id %q", formatAMSNetID(netID))
	}
	if _, err := resolveTargetNetID("plc.local", ""); err == nil {
		t.Fatal("hostname without net id should be rejected")
	}
}

func TestParseAddress(t *testing.T) {
	direct, err := parseAddress("0xF020,0x10", "int16")
	if err != nil {
		t.Fatal(err)
	}
	if direct.IndexGroup != 0xF020 || direct.IndexOffset != 16 {
		t.Fatalf("unexpected address %+v", direct)
	}
	direct, err = parseAddress("0xF020:16", "int16")
	if err != nil {
		t.Fatal(err)
	}
	if direct.IndexOffset != 16 {
		t.Fatalf("unexpected decimal offset %+v", direct)
	}
	text, err := parseAddress("0x4020,0.80", "STRING")
	if err != nil {
		t.Fatal(err)
	}
	if text.StringLength != 80 {
		t.Fatalf("unexpected string length %+v", text)
	}
	if text.ReadOnly {
		t.Fatalf("0x4020 should be writable %+v", text)
	}
	readOnly, err := parseAddress("0x4025,0", "int16")
	if err != nil {
		t.Fatal(err)
	}
	if !readOnly.ReadOnly {
		t.Fatalf("0x4025 should be read-only %+v", readOnly)
	}
	writeOnly, err := parseAddress("0xF006,0", "int16")
	if err != nil {
		t.Fatal(err)
	}
	if !writeOnly.WriteOnly {
		t.Fatalf("0xF006 should be write-only %+v", writeOnly)
	}
}

func TestParseAddressRejectsInvalid(t *testing.T) {
	cases := []struct {
		address  string
		dataType string
	}{
		{"", "int16"},
		{"MAIN.nCounter", "int16"},
		{"0x9999,0", "int16"},
		{"0x4020,0x10000", "int16"},
		{"0x4025,1", "int16"},
		{"0x4020,0.80", "int16"},
		{"0x4020,0", "STRING"},
		{"0x4020,0.300", "STRING"},
	}
	for _, tc := range cases {
		if _, err := parseAddress(tc.address, tc.dataType); err == nil {
			t.Fatalf("%q (%s) should be rejected", tc.address, tc.dataType)
		}
	}
}

func TestTypeCodec(t *testing.T) {
	cases := []struct {
		dataType string
		value    any
		length   int
		bytes    []byte
	}{
		{"bool", true, 1, []byte{1}},
		{"int16", int16(42), 2, []byte{42, 0}},
		{"int32", int32(-2), 4, []byte{0xFE, 0xFF, 0xFF, 0xFF}},
		{"float32", float32(1.5), 4, le32(0x3FC00000)},
		{"float64", float64(2.5), 8, le64(0x4004000000000000)},
		{"string", "ab", 4, []byte{'a', 'b', 0, 0}},
	}
	for _, tc := range cases {
		address := parsedAddress{StringLength: tc.length}
		size, err := typeSize(tc.dataType, address)
		if err != nil {
			t.Fatalf("%s: %v", tc.dataType, err)
		}
		if tc.dataType != "string" && size != tc.length {
			t.Fatalf("%s: unexpected size %d", tc.dataType, size)
		}
		encoded, err := encodeValue(tc.value, tc.dataType, tc.length)
		if err != nil {
			t.Fatalf("%s: %v", tc.dataType, err)
		}
		if !reflect.DeepEqual(encoded, tc.bytes) {
			t.Fatalf("%s: unexpected bytes % X", tc.dataType, encoded)
		}
		decoded, err := decodeValue(encoded, tc.dataType)
		if err != nil {
			t.Fatalf("%s: %v", tc.dataType, err)
		}
		if !reflect.DeepEqual(decoded, tc.value) {
			t.Fatalf("%s: unexpected value %#v", tc.dataType, decoded)
		}
	}
}

func TestADSClientDirectReadWrite(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		for {
			request, err := readAMSPacket(conn)
			if err != nil {
				if err == io.EOF {
					serverErr <- nil
				} else {
					serverErr <- err
				}
				return
			}
			command := binary.LittleEndian.Uint16(request[16:])
			var data []byte
			switch command {
			case adsCommandRead:
				if binary.LittleEndian.Uint32(request[32:]) != 0xF020 ||
					binary.LittleEndian.Uint32(request[36:]) != 16 ||
					binary.LittleEndian.Uint32(request[40:]) != 2 {
					serverErr <- errUnexpectedRead
					return
				}
				data = make([]byte, 10)
				binary.LittleEndian.PutUint32(data[4:], 2)
				binary.LittleEndian.PutUint16(data[8:], 42)
			case adsCommandWrite:
				if binary.LittleEndian.Uint32(request[32:]) != 0xF020 ||
					binary.LittleEndian.Uint32(request[36:]) != 16 ||
					binary.LittleEndian.Uint32(request[40:]) != 2 ||
					binary.LittleEndian.Uint16(request[44:]) != 7 {
					serverErr <- errUnexpectedWrite
					return
				}
				data = make([]byte, 8)
			default:
				serverErr <- errUnexpectedCommand
				return
			}
			if _, err := conn.Write(amsResponse(request, data)); err != nil {
				serverErr <- err
				return
			}
		}
	}()

	client, err := newADSClient(context.Background(), listener.Addr().String(),
		[6]byte{192, 168, 1, 10, 1, 1}, 851,
		[6]byte{127, 0, 0, 1, 1, 1}, 851, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	data, _, _, err := client.read(context.Background(), 0xF020, 16, 2)
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeValue(data, "int16")
	if err != nil {
		t.Fatal(err)
	}
	if value != int16(42) {
		t.Fatalf("unexpected value %v", value)
	}
	if _, _, _, err := client.write(context.Background(), 0xF020, 16, le16(7)); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

var (
	errUnexpectedRead    = &testError{"unexpected read request"}
	errUnexpectedWrite   = &testError{"unexpected write request"}
	errUnexpectedCommand = &testError{"unexpected command"}
)

type testError struct{ text string }

func (e *testError) Error() string { return e.text }

func readAMSPacket(conn net.Conn) ([]byte, error) {
	header := make([]byte, amsTCPHeader)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := int(binary.LittleEndian.Uint32(header[2:]))
	if length < amsHeaderSize || length > 1024*1024 {
		return nil, errUnexpectedCommand
	}
	packet := make([]byte, length)
	if _, err := io.ReadFull(conn, packet); err != nil {
		return nil, err
	}
	return packet, nil
}

func amsResponse(request, data []byte) []byte {
	packet := make([]byte, amsHeaderSize+len(data))
	copy(packet[0:6], request[8:14])
	binary.LittleEndian.PutUint16(packet[6:], binary.LittleEndian.Uint16(request[14:]))
	copy(packet[8:14], request[0:6])
	binary.LittleEndian.PutUint16(packet[14:], binary.LittleEndian.Uint16(request[6:]))
	binary.LittleEndian.PutUint16(packet[16:], binary.LittleEndian.Uint16(request[16:]))
	binary.LittleEndian.PutUint16(packet[18:], 0x0005)
	binary.LittleEndian.PutUint32(packet[20:], uint32(len(data)))
	binary.LittleEndian.PutUint32(packet[28:], binary.LittleEndian.Uint32(request[28:]))
	copy(packet[amsHeaderSize:], data)
	wire := make([]byte, amsTCPHeader+len(packet))
	binary.LittleEndian.PutUint32(wire[2:], uint32(len(packet)))
	copy(wire[amsTCPHeader:], packet)
	return wire
}

func TestConfigDefaults(t *testing.T) {
	item, err := NewCollection(&CreateReq{Name: "ethercat"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Port != DefaultPort || item.AMSPort != DefaultAMSPort ||
		item.LocalAMSPort != DefaultLocalAMSPort || item.TimeoutMs != DefaultTimeoutMs {
		t.Fatalf("unexpected defaults %+v", item)
	}
	raw, err := item.ConfigJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"port":48898`) || !strings.Contains(string(raw), `"amsPort":851`) ||
		strings.Contains(string(raw), "stringLength") {
		t.Fatalf("unexpected config %s", raw)
	}
}

func TestBuildReadFrames(t *testing.T) {
	frames, err := buildReadFrames([]*collection.Variable{
		{Name: "counter", DataType: "int16", Address: "0xF020,0", AccessMode: collection.AccessModeRead},
		{Name: "text", DataType: "STRING", Address: "0x4020,0.8", AccessMode: collection.AccessModeRead},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || frames[0].Length != 2 || frames[1].Length != 8 {
		t.Fatalf("unexpected frames %+v", frames)
	}
	if _, err := buildReadFrames([]*collection.Variable{
		{Name: "release", DataType: "int16", Address: "0xF006,0", AccessMode: collection.AccessModeRead},
	}); err == nil {
		t.Fatal("write-only group should not be readable")
	}
}
