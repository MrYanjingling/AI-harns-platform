package comli

import (
	"encoding/binary"
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestBuildReadRequestMatchesDocumentedExample(t *testing.T) {
	frame := buildReadRequest(1, 0x4000, 2, '1')
	want := []byte{0x02, '0', '1', '1', '2', '4', '0', '0', '0', '0', '2', 0x03, 0x05}
	if len(frame) != len(want) {
		t.Fatalf("frame length %d, want %d", len(frame), len(want))
	}
	for i := range want {
		if frame[i] != want[i] {
			t.Fatalf("frame[%d]=%#x, want %#x (frame=% X)", i, frame[i], want[i], frame)
		}
	}
}

func TestMirrorByte(t *testing.T) {
	if got := mirrorByte(0x31); got != 0x8C {
		t.Fatalf("mirrorByte(0x31)=%#x, want 0x8c", got)
	}
	if got := mirrorByte(mirrorByte(0xA5)); got != 0xA5 {
		t.Fatalf("double mirror=%#x, want 0xa5", got)
	}
}

func TestParseReadResponseAndDecode(t *testing.T) {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, 0x1234)
	for i := range data {
		data[i] = mirrorByte(data[i])
	}
	frame := []byte{stx, '0', '0', '1', '2', '4', '0', '0', '0', '0', '2'}
	frame = append(frame, data...)
	frame = append(frame, etx)
	frame = append(frame, checksum(frame))

	got, err := parseReadResponse(frame, 0x4000, 2, '1', messageReadRegisters)
	if err != nil {
		t.Fatal(err)
	}
	point := &comliPoint{area: areaRegisters, offset: 0, quantity: 2, dataType: "INT16"}
	command := &readCommand{area: areaRegisters, start: 0, end: 2}
	value, err := decodePointValue(point, command, got)
	if err != nil {
		t.Fatal(err)
	}
	if number, ok := value.(int16); !ok || number != 0x1234 {
		t.Fatalf("decoded value %#v, want int16(0x1234)", value)
	}
}

func TestParseReadResponseRejectsBadBCC(t *testing.T) {
	frame := []byte{stx, '0', '0', '1', '2', '4', '0', '0', '0', '0', '2', 0x00, etx, 0x00}
	if _, err := parseReadResponse(frame, 0x4000, 2, '1', messageReadRegisters); err == nil {
		t.Fatal("expected bad BCC error")
	}
}

func TestParseWriteAck(t *testing.T) {
	frame := []byte{stx, '0', '0', '1', '1', 0x06, etx}
	frame = append(frame, checksum(frame))
	if err := parseWriteAck(frame, '1'); err != nil {
		t.Fatal(err)
	}
	if err := parseWriteAck(frame, '2'); err == nil {
		t.Fatal("expected stamp mismatch")
	}
}

func TestParseAddress(t *testing.T) {
	cases := []struct {
		address string
		data    string
		area    byte
		offset  uint16
		bit     uint8
		qty     int
		wantErr bool
	}{
		{"1!0.3", "BIT", areaBits, 3, 3, 1, false},
		{"1!1.0", "INT16", areaRegisters, 0, 0, 2, false},
		{"2!1.100", "FLOAT32", areaRegisters, 100, 0, 4, false},
		{"1!1.0.3", "BIT", areaRegisters, 0, 3, 2, false},
		{"0!1.0", "INT16", areaRegisters, 0, 0, 2, true},
		{"1!0.3", "INT16", areaBits, 0, 0, 0, true},
		{"1!1.4096", "INT16", areaRegisters, 0, 0, 0, true},
	}
	for _, tc := range cases {
		point, err := parseAddress(&collection.Variable{Address: tc.address, DataType: tc.data})
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error", tc.address)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", tc.address, err)
		}
		if point.area != tc.area || point.offset != tc.offset || point.bit != tc.bit || point.quantity != tc.qty {
			t.Fatalf("%s: unexpected point %+v", tc.address, point)
		}
	}
}

func TestBuildReadCommandsMergesContiguousRegisters(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", Address: "1!1.0", DataType: "INT16"},
		{Name: "b", Address: "1!1.1", DataType: "INT16"},
		{Name: "c", Address: "1!1.2", DataType: "FLOAT32"},
		{Name: "d", Address: "2!1.0", DataType: "INT16"},
	}
	commands := buildReadCommands(variables)
	if len(commands) != 2 {
		t.Fatalf("commands=%d, want 2 (%+v)", len(commands), commands)
	}
	if commands[0].identity != 1 || commands[0].start != 0 || commands[0].end != 8 {
		t.Fatalf("unexpected first command %+v", commands[0])
	}
	if commands[1].identity != 2 || commands[1].start != 0 || commands[1].end != 2 {
		t.Fatalf("unexpected second command %+v", commands[1])
	}
}

func TestBuildWriteRequestMirrorsData(t *testing.T) {
	frame := buildWriteRequest(1, messageWriteBit, 3, '1', []byte{'1'})
	if frame[0] != stx || frame[4] != messageWriteBit || frame[11] != mirrorByte('1') {
		t.Fatalf("unexpected write frame % X", frame)
	}
	if checksum(frame[:len(frame)-1]) != frame[len(frame)-1] {
		t.Fatalf("bad BCC in % X", frame)
	}
}
