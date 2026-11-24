package fx

import (
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestParseAddress(t *testing.T) {
	cases := []struct {
		address   string
		data      string
		bitAddr   uint16
		byteAddr  uint16
		wordStart uint16
		bytes     int
		wantErr   bool
	}{
		{"X0", "BIT", 0x400, 0x80, 0, 1, false},
		{"Y1", "BIT", 0x501, 0xA0, 0, 1, false},
		{"M0", "BIT", 0x800, 0x100, 0, 1, false},
		{"D0", "INT16", 0, 0, 0x1000, 2, false},
		{"D1", "INT32", 0, 0, 0x1002, 4, false},
		{"D0.3", "BIT", 0, 0, 0x1000, 2, false},
		{"M0", "INT16", 0, 0, 0, 0, true},
		{"Z0", "INT16", 0, 0, 0, 0, true},
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
		if point.bitAddr != tc.bitAddr || point.byteAddr != tc.byteAddr || point.wordStart != tc.wordStart || point.bytes != tc.bytes {
			t.Fatalf("%s: unexpected point %+v", tc.address, point)
		}
	}
}

func TestFXRequests(t *testing.T) {
	request := buildReadRequest(0x1000, 2)
	if request[0] != stx || request[1] != cmdRead || string(request[2:6]) != "1000" || string(request[6:8]) != "02" || request[8] != etx {
		t.Fatalf("unexpected read request % X", request)
	}
	on := buildBitRequest(true, 0x800)
	if on[1] != cmdOn || string(on[2:6]) != "0008" {
		t.Fatalf("unexpected ON request % X", on)
	}
	off := buildBitRequest(false, 0x800)
	if off[1] != cmdOff {
		t.Fatalf("unexpected OFF request % X", off)
	}
}

func TestParseReadResponse(t *testing.T) {
	data := []byte{0x02, '3', '4', '1', '2', etx}
	data = append(data, checksum(data[1:])...)
	got, err := parseReadResponse(data, 2)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string([]byte{0x34, 0x12}) {
		t.Fatalf("data=% X", got)
	}
	if err := parseWriteAck([]byte{ack}); err != nil {
		t.Fatal(err)
	}
}
