package memobus

import "testing"

func TestParseAndFrames(t *testing.T) {
	a, err := parseAddress("MW100", "INT16")
	if err != nil || a.Reg != 100 || a.Words != 1 {
		t.Fatalf("unexpected MW100: %+v %v", a, err)
	}
	req := buildReadRequest(1, 0, 100, 2)
	if len(req) != headerSize+10 || req[0] != msgTypeCmd || req[headerSize+3] != sfcReadHoldReg {
		t.Fatalf("unexpected read request: % X", req)
	}
	// Construct mock response: 12B header + 8B PDU header + 4B data (2 regs LE)
	resp := make([]byte, headerSize+12)
	resp[0] = msgTypeResp
	resp[6] = byte(len(resp))
	resp[headerSize+2] = mfcExtended
	resp[headerSize+3] = sfcReadHoldReg
	resp[headerSize+6] = 2
	resp[headerSize+8] = 0x34
	resp[headerSize+9] = 0x12
	payload, err := parseReadResponse(resp, 2)
	if err != nil || len(payload) != 4 {
		t.Fatalf("unexpected parseReadResponse: % X %v", payload, err)
	}
	val, err := decodeValue(a, payload, 0)
	if err != nil || val != int16(0x1234) {
		t.Fatalf("unexpected decoded value: %v %v", val, err)
	}
}
