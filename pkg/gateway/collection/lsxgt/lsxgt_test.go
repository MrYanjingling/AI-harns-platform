package lsxgt

import "testing"

func TestParseAddressAndFrames(t *testing.T) {
	a, err := parseAddress("%MW100", "INT16")
	if err != nil || a.Area != 'M' || a.ByteOffset != 200 || a.ByteCount != 2 {
		t.Fatalf("unexpected %%MW100: %+v %v", a, err)
	}
	b, err := parseAddress("%MX10", "BOOL")
	if err != nil || b.Area != 'M' || b.ByteOffset != 1 || b.BitIndex != 2 || b.BitOffset != 10 {
		t.Fatalf("unexpected %%MX10: %+v %v", b, err)
	}
	req := buildFEnetReadRequest(1, 'M', 200, 4)
	if string(req[:8]) != "LSIS-XGT" || len(req) != fenetHeaderSize+18 {
		t.Fatalf("unexpected FEnet read request len=%d", len(req))
	}
	cnetReq := buildCnetReadRequest(0, 'M', 200, 4)
	if cnetReq[0] != 0x05 {
		t.Fatalf("expected ENQ start, got % X", cnetReq)
	}
	payload, err := parseCnetReadResponse([]byte("\x0600RSB01023412\x03"))
	if err != nil || len(payload) != 2 || payload[0] != 0x34 || payload[1] != 0x12 {
		t.Fatalf("unexpected Cnet response: % X %v", payload, err)
	}
}
