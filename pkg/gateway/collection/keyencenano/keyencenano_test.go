package keyencenano

import (
	"math"
	"testing"
)

func TestParseAddressAndCommands(t *testing.T) {
	a, err := parseAddress("DM100", "INT16")
	if err != nil || a.Area != "DM" || a.Offset != 100 || a.Words != 1 || a.IsBitArea {
		t.Fatalf("unexpected DM100: %+v %v", a, err)
	}
	b, err := parseAddress("R10", "BOOL")
	if err != nil || b.Area != "R" || b.Offset != 10 || !b.IsBitArea {
		t.Fatalf("unexpected R10: %+v %v", b, err)
	}
	cmd, err := buildWriteCommand(b, true)
	if err != nil || string(cmd) != "WR R10 1\r" {
		t.Fatalf("unexpected WR bit cmd %q %v", cmd, err)
	}
	f, _ := parseAddress("DM200", "FLOAT32")
	cmd, err = buildWriteCommand(f, float64(10.5))
	if err != nil {
		t.Fatal(err)
	}
	u := math.Float32bits(10.5)
	words, err := parseReadResponse("0 16680\r\n", 2)
	if err != nil {
		t.Fatal(err)
	}
	if words[0] != uint16(u) || words[1] != uint16(u>>16) {
		t.Fatalf("words mismatch: %v", words)
	}
	val, err := decodeWords(f, words)
	if err != nil || val != float32(10.5) {
		t.Fatalf("unexpected decoded float32: %v %v", val, err)
	}
}
