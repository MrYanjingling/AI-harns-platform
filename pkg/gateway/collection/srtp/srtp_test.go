package srtp

import (
	"encoding/binary"
	"testing"
)

func TestParseAddress(t *testing.T) {
	cases := []struct {
		address  string
		dataType string
		memory   uint8
		start    uint16
		quantity uint16
		length   uint16
		bit      int
		wantErr  bool
	}{
		{"%R1", "INT16", 0x08, 0, 1, 2, 0, false},
		{"%R2", "INT32", 0x08, 1, 2, 4, 0, false},
		{"%AI1", "FLOAT32", 0x0A, 0, 2, 4, 0, false},
		{"%I1", "UINT8", 0x10, 0, 1, 1, 0, false},
		{"%I1", "INT16", 0x10, 0, 2, 2, 0, false},
		{"%I1", "BIT", 0x46, 0, 1, 1, 0, false},
		{"%R1.3", "BIT", 0x08, 0, 1, 2, 3, false},
		{"%R1#8", "STRING", 0x08, 0, 4, 8, 0, false},
		{"%R0", "INT16", 0, 0, 0, 0, 0, true},
		{"%R1.16", "BIT", 0, 0, 0, 0, 0, true},
		{"%R1", "STRING", 0, 0, 0, 0, 0, true},
		{"%R1#8", "INT16", 0, 0, 0, 0, 0, true},
	}
	for _, test := range cases {
		address, err := ParseAddress(test.address, test.dataType)
		if (err != nil) != test.wantErr {
			t.Fatalf("%s: err=%v", test.address, err)
		}
		if err == nil {
			if address.Memory != test.memory || address.Start != test.start || address.Quantity != test.quantity || address.ByteLength != test.length || address.Bit != test.bit {
				t.Fatalf("%s: got %+v", test.address, address)
			}
		}
	}
}

func TestBuildRequest(t *testing.T) {
	address, err := ParseAddress("%R1", "INT16")
	if err != nil {
		t.Fatal(err)
	}
	frame := buildRequest(serviceRead, address, nil, address.ByteLength)
	if len(frame) != headerSize {
		t.Fatalf("length=%d", len(frame))
	}
	if frame[0x2a] != serviceRead || frame[0x2b] != memoryRWord {
		t.Fatalf("header=% X", frame[:0x30])
	}
	if binary.LittleEndian.Uint16(frame[0x2c:]) != 0 || binary.LittleEndian.Uint16(frame[0x2e:]) != 1 {
		t.Fatalf("address=% X", frame[0x2c:0x30])
	}
}

func TestValueCodec(t *testing.T) {
	address, _ := ParseAddress("%R1", "INT16")
	data, err := encodeValue(int16(1234), address)
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeValue(data, address)
	if err != nil {
		t.Fatal(err)
	}
	if value != int16(1234) {
		t.Fatalf("value=%v", value)
	}

	bit, _ := ParseAddress("%R1.3", "BIT")
	word := make([]byte, 2)
	binary.LittleEndian.PutUint16(word, 8)
	value, err = decodeValue(word, bit)
	if err != nil {
		t.Fatal(err)
	}
	if value != true {
		t.Fatalf("bit=%v", value)
	}
}
