package yokogawalink

import "testing"

func TestParseAndFrames(t *testing.T) {
	a, err := parseAddress("D100", "INT16")
	if err != nil || a.Area != 'D' || a.AreaCode != 0x0004 || a.Offset != 100 {
		t.Fatalf("unexpected D100: %+v %v", a, err)
	}
	req := buildReadRequest(1, &readBatch{area: 'D', areaCode: 0x0004, start: 100, count: 2})
	if len(req) != 12 || req[0] != cmdWordRead || req[1] != 1 {
		t.Fatalf("unexpected read request: % X", req)
	}
	payload, err := parseResponse(cmdWordRead, []byte{0x91, 0x00, 0x00, 0x02, 0x12, 0x34})
	if err != nil || len(payload) != 2 {
		t.Fatalf("unexpected parseResponse: % X %v", payload, err)
	}
	val, err := decodeValue(a, payload, 0)
	if err != nil || val != int16(0x1234) {
		t.Fatalf("unexpected decoded value: %v %v", val, err)
	}
}
