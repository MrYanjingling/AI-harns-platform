package cclinkie

import (
	"encoding/binary"
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestCyclicRequestAndResponse(t *testing.T) {
	rww := make([]byte, 64)
	ry := make([]byte, 8)
	binary.LittleEndian.PutUint16(rww[0:2], 0x1234)
	ry[0] = 0x05

	req := buildCyclicRequest(0x0100007F, 0x0100007F, 1, 1, 1, rww, ry)
	if binary.BigEndian.Uint16(req[0:2]) != slmpSubheaderReq {
		t.Fatalf("bad subheader %x", req[0:2])
	}
	if binary.LittleEndian.Uint16(req[11:13]) != slmpCmdCyclicData {
		t.Fatalf("bad cmd %x", req[11:13])
	}

	// Build a valid 1-station response (dl = 50 + 72 = 122, total = 131)
	resp := make([]byte, 131)
	binary.BigEndian.PutUint16(resp[0:2], slmpSubheaderResp)
	binary.LittleEndian.PutUint16(resp[7:9], 122)
	binary.LittleEndian.PutUint16(resp[11:13], protocolVersion)
	binary.LittleEndian.PutUint16(resp[59:61], 0x5678)
	resp[59+64] = 0x02 // RX1 = true

	rwr, rx, err := parseCyclicResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	pWord, _ := parseAddress(&collection.Variable{Name: "rwr0", DataType: "UINT16", Address: "RWr0"})
	vWord, err := decodeWordPoint(pWord, rwr)
	if err != nil || vWord != uint16(0x5678) {
		t.Fatalf("rwr0=%v err=%v", vWord, err)
	}
	vBit, err := decodeBit(rx, 1, "BOOL")
	if err != nil || vBit != true {
		t.Fatalf("rx1=%v err=%v", vBit, err)
	}
}
