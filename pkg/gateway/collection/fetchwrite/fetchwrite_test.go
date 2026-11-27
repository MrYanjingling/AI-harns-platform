package fetchwrite

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math"
	"net"
	"strconv"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		addr      string
		dataType  string
		wantOrg   byte
		wantDB    byte
		wantStart uint16
		wantUnits uint16
		wantBit   uint8
		wantWordB bool
	}{
		{"DB1.DBW0", "INT16", OrgDB, 1, 0, 1, 0, false},
		{"DB1.DBW10", "FLOAT32", OrgDB, 1, 10, 2, 0, false},
		{"DB1.DBW20", "FLOAT64", OrgDB, 1, 20, 4, 0, false},
		{"DB1.DBW5.7", "BOOL", OrgDB, 1, 5, 1, 7, true},
		{"DB2.DBX3.4", "BOOL", OrgDB, 2, 1, 1, 4, false},
		{"M10.3", "BOOL", OrgM, 0, 10, 1, 3, false},
		{"M20", "INT32", OrgM, 0, 20, 4, 0, false},
		{"I0", "UINT16", OrgI, 0, 0, 2, 0, false},
		{"Q2.1", "BIT", OrgQ, 0, 2, 1, 1, false},
		{"Z1", "UINT16", OrgZ, 0, 1, 1, 0, false},
		{"T2", "UINT16", OrgT, 0, 2, 1, 0, false},
	}
	for _, tc := range tests {
		got, err := ParseAddress(tc.addr, tc.dataType)
		if err != nil {
			t.Fatalf("ParseAddress(%q, %q) err = %v", tc.addr, tc.dataType, err)
		}
		if got.Org != tc.wantOrg || got.DB != tc.wantDB || got.Start != tc.wantStart || got.UnitCount != tc.wantUnits || got.Bit != tc.wantBit || got.WordBit != tc.wantWordB {
			t.Fatalf("ParseAddress(%q, %q) = %+v", tc.addr, tc.dataType, got)
		}
	}
}

func TestRequestHeaderEncoding(t *testing.T) {
	hdr := buildRequestHeader(opFetch, OrgDB, 1, 0, 2)
	want := []byte{0x53, 0x35, 0x10, 0x01, 0x03, 0x05, 0x03, 0x08, 0x01, 0x01, 0x00, 0x00, 0x00, 0x02, 0xFF, 0x02}
	if !bytes.Equal(hdr[:], want) {
		t.Fatalf("hdr = %x, want %x", hdr, want)
	}
}

func TestBuildReadBatches(t *testing.T) {
	tags := make([]*tagRef, 0, 20)
	for i := 0; i < 10; i++ {
		addr, _ := ParseAddress("DB1.DBW"+strconv.Itoa(i), "INT16")
		tags = append(tags, &tagRef{variable: &collection.Variable{Name: "w", DataType: "INT16"}, address: addr})
	}
	for i := 0; i < 10; i++ {
		addr, _ := ParseAddress("M"+strconv.Itoa(i), "BOOL")
		tags = append(tags, &tagRef{variable: &collection.Variable{Name: "m", DataType: "BOOL"}, address: addr})
	}
	batches := buildReadBatches(tags)
	if len(batches) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(batches))
	}
	if batches[0].org != OrgDB || batches[0].unitCount != 10 {
		t.Fatalf("unexpected DB batch: %+v", batches[0])
	}
	if batches[1].org != OrgM || batches[1].unitCount != 10 {
		t.Fatalf("unexpected M batch: %+v", batches[1])
	}
}

func TestMockServerFetchWriteRoundTrip(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	dbMem := make([]byte, 4096)
	mMem := make([]byte, 2048)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				hdr := make([]byte, headerSize)
				for {
					if _, err := io.ReadFull(c, hdr); err != nil {
						return
					}
					op := hdr[5]
					org := hdr[8]
					start := int(binary.BigEndian.Uint16(hdr[10:12]))
					length := int(binary.BigEndian.Uint16(hdr[12:14]))
					ub := unitBytes(org)
					byteStart := start * ub
					byteLen := length * ub
					mem := dbMem
					if org == OrgM {
						mem = mMem
					}
					resp := [headerSize]byte{'S', '5', 0x10, 0x01, 0x03, op + 1, 0x0F, 0x03, 0x00, 0xFF, 0x07}
					if op == opFetch {
						_, _ = c.Write(resp[:])
						_, _ = c.Write(mem[byteStart : byteStart+byteLen])
					} else if op == opWrite {
						buf := make([]byte, byteLen)
						if _, err := io.ReadFull(c, buf); err != nil {
							return
						}
						copy(mem[byteStart:byteStart+byteLen], buf)
						_, _ = c.Write(resp[:])
					}
				}
			}(conn)
		}
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	req := &CreateReq{
		Name:             "test-fw",
		CollectorCycleMs: 200,
		ProtocolConfig: ProtocolConfig{
			Location:    host,
			Port:        port,
			TimeoutMs:   2000,
			Concurrency: 1,
		},
		Variables: []*collection.Variable{
			{Name: "b0", Address: "M0.1", DataType: "BOOL", AccessMode: "rw"},
			{Name: "b1", Address: "M0.3", DataType: "BOOL", AccessMode: "rw"},
			{Name: "w0", Address: "DB1.DBW0", DataType: "INT16", AccessMode: "rw"},
			{Name: "f64", Address: "DB1.DBW10", DataType: "FLOAT64", AccessMode: "rw"},
		},
	}
	col, err := NewCollection(req)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := col.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer col.Stop(ctx)
	if err := col.Start(ctx); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if col.GetCollectStatus() == collection.CollectStatusCollecting {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if col.GetCollectStatus() != collection.CollectStatusCollecting {
		t.Fatalf("collector not collecting: %s", col.GetCollectStatus())
	}

	actions := []collection.ActionValue{
		{Variable: req.Variables[0], Value: true},
		{Variable: req.Variables[1], Value: true},
		{Variable: req.Variables[2], Value: int16(-1234)},
		{Variable: req.Variables[3], Value: 3.14159265},
	}
	if err := col.DeliverAction(ctx, actions); err != nil {
		t.Fatalf("DeliverAction failed: %v", err)
	}
	time.Sleep(350 * time.Millisecond)

	vars := col.GetCollectionMeta().SnapshotVariables()
	got := make(map[string]any, len(vars))
	for _, v := range vars {
		got[v.Name] = v.Value
	}
	if got["b0"] != true || got["b1"] != true {
		t.Fatalf("unexpected bool values: b0=%v b1=%v", got["b0"], got["b1"])
	}
	if got["w0"] != int16(-1234) {
		t.Fatalf("unexpected w0: %#v", got["w0"])
	}
	if f, ok := got["f64"].(float64); !ok || math.Abs(f-3.14159265) > 1e-6 {
		t.Fatalf("unexpected f64: %#v", got["f64"])
	}
}
