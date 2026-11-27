package mc

import (
	"bytes"
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestBuildReadRequests(t *testing.T) {
	cases := []struct {
		family Family
		want   []byte
	}{
		{Family1E, []byte{0x01, 0xFF, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x44, 0x02, 0x00}},
		{Family3E, []byte{0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, 0x0C, 0x00, 0x00, 0x00, 0x01, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA8, 0x02, 0x00}},
		{Family4E, []byte{0x54, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, 0x0C, 0x00, 0x00, 0x00, 0x01, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA8, 0x02, 0x00}},
	}
	for _, tc := range cases {
		got := buildReadRequest(tc.family, 0, 0xA8, true, 0, 2)
		if !bytes.Equal(got, tc.want) {
			t.Fatalf("%s request mismatch\n got=% X\nwant=% X", tc.family, got, tc.want)
		}
	}
}

func TestParseResponses(t *testing.T) {
	data, err := parseResponse(Family1E, 0, []byte{0x81, 0x00, 0x34, 0x12, 0x78, 0x56})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x34, 0x12, 0x78, 0x56}) {
		t.Fatalf("1E data=% X", data)
	}
	if _, err := parseResponse(Family1E, 0, []byte{0x81, 0x5B, 0x10}); err == nil {
		t.Fatal("expected 1E error end code")
	}
	data, err = parseResponse(Family3E, 0, []byte{0xD0, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, 0x06, 0x00, 0x00, 0x00, 0x34, 0x12, 0x78, 0x56})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x34, 0x12, 0x78, 0x56}) {
		t.Fatalf("data=% X", data)
	}
	data, err = parseResponse(Family4E, 7, []byte{0xD4, 0x00, 0x07, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, 0x06, 0x00, 0x00, 0x00, 0x34, 0x12, 0x78, 0x56})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x34, 0x12, 0x78, 0x56}) {
		t.Fatalf("4E data=% X", data)
	}
	if _, err := parseResponse(Family4E, 8, []byte{0xD4, 0x00, 0x07, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, 0x06, 0x00, 0x00, 0x00, 0x34, 0x12, 0x78, 0x56}); err == nil {
		t.Fatal("expected serial mismatch")
	}
}

func TestParseAddressAndGrouping(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "d0", Address: "D0", DataType: "INT16"},
		{Name: "d1", Address: "D1", DataType: "INT16"},
		{Name: "m0", Address: "M0", DataType: "BIT"},
		{Name: "m1", Address: "M1", DataType: "BIT"},
		{Name: "d2bit", Address: "D2.3", DataType: "BIT"},
	}
	commands := buildCommands(variables)
	if len(commands) != 2 {
		t.Fatalf("commands=%d, want 2", len(commands))
	}
	if commands[0].area != 0x90 || commands[0].word || commands[0].count != 2 {
		t.Fatalf("unexpected bit command %+v", commands[0])
	}
	if commands[1].area != 0xA8 || !commands[1].word || commands[1].count != 3 {
		t.Fatalf("unexpected word command %+v", commands[1])
	}
}

func TestDecodeWordAndBit(t *testing.T) {
	item := &point{number: 0, quantity: 1, dataType: "INT16", word: true}
	cmd := &command{word: true, start: 0, count: 1}
	value, err := decodePoint(item, cmd, []byte{0x34, 0x12})
	if err != nil {
		t.Fatal(err)
	}
	if value != int16(0x1234) {
		t.Fatalf("value=%#v", value)
	}
	bit := &point{number: 0, dataType: "BIT", word: false}
	bitCmd := &command{word: false, start: 0, count: 1}
	value, err = decodePoint(bit, bitCmd, []byte{0x10})
	if err != nil {
		t.Fatal(err)
	}
	if value != true {
		t.Fatalf("bit value=%#v", value)
	}
}
