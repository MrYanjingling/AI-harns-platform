package hsms

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func TestSECS2CodecAndSML(t *testing.T) {
	orig := &Item{
		Type: ItemList,
		Children: []*Item{
			{Type: ItemASCII, Raw: []byte("MODEL-A")},
			{Type: ItemUint16, Raw: []byte{0x12, 0x34}},
			{Type: ItemBoolean, Raw: []byte{0x01}},
			{Type: ItemBinary, Raw: []byte{0x00, 0xAB}},
			{Type: ItemFloat32, Raw: func() []byte { b := make([]byte, 4); binary.BigEndian.PutUint32(b, math.Float32bits(36.5)); return b }()},
		},
	}
	wire, err := orig.Encode()
	if err != nil {
		t.Fatal(err)
	}
	decoded, n, err := DecodeItem(wire)
	if err != nil || n != len(wire) {
		t.Fatalf("DecodeItem err=%v n=%d len=%d", err, n, len(wire))
	}
	sml := decoded.FormatSML()
	wantSML := "<L[5] <A[7] MODEL-A> <U2 4660> <Boolean 1> <B[2] 00AB> <F4 36.5> >."
	if sml != wantSML {
		t.Fatalf("FormatSML = %q, want %q", sml, wantSML)
	}
	reparsed, err := ParseSML(sml)
	if err != nil {
		t.Fatalf("ParseSML failed: %v", err)
	}
	wire2, err := reparsed.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if string(wire2) != string(wire) {
		t.Fatalf("SML round-trip mismatch: got %x want %x", wire2, wire)
	}
}

func TestParseAddress(t *testing.T) {
	cases := []struct {
		addr   string
		dt     string
		stream uint8
		fn     uint8
		kind   indexKind
	}{
		{"S1F1", "STRING", 1, 1, indexNone},
		{"S1F1[1]", "STRING", 1, 1, indexList},
		{"S6F11[3][1][2]", "UINT32", 6, 11, indexList},
		{"S1F3.1001", "FLOAT32", 1, 3, indexSVID},
		{"S1F12.MEASURE1", "STRING", 1, 12, indexSVID},
	}
	for _, tc := range cases {
		p, err := parseAddress(&collection.Variable{Name: "v", Address: tc.addr, DataType: tc.dt})
		if err != nil {
			t.Fatalf("parseAddress(%q) unexpected err: %v", tc.addr, err)
		}
		if p.stream != tc.stream || p.function != tc.fn || p.kind != tc.kind {
			t.Fatalf("parseAddress(%q) = S%dF%d kind=%d", tc.addr, p.stream, p.function, p.kind)
		}
	}
	if _, err := parseAddress(&collection.Variable{Name: "bad", Address: "D100", DataType: "INT16"}); err == nil {
		t.Fatal("expected error for non-HSMS address")
	}
}

// TestHSMSCollectorEndToEnd spins up a compliant SEMI E37/E5/E30 mock equipment server
// verifying Select, S1F13/S1F14, S1F1 list indexing, S1F3 SVID polling, unsolicited S6F11
// event push with auto S6F12 ACK, and DeliverAction write (scalar + SML list).
func TestHSMSCollectorEndToEnd(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var (
		mu          sync.Mutex
		s6f12Acked  bool
		lastWritten string
		sv1001      float32 = 24.5
		sv1002      uint32  = 77
	)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var writeMu sync.Mutex
		sendMsg := func(m *Message) {
			writeMu.Lock()
			defer writeMu.Unlock()
			_, _ = conn.Write(m.Encode())
		}

		pushedEvent := false
		for {
			msg, err := ReadMessage(conn)
			if err != nil {
				return
			}
			switch msg.Header.SType {
			case STypeSelectReq:
				sendMsg(buildControlMessage(STypeSelectRsp, 0, msg.Header.SystemBytes))
			case STypeLinktestReq:
				sendMsg(buildControlMessage(STypeLinktestRsp, 0, msg.Header.SystemBytes))
			case STypeSeparateReq:
				return
			case STypeData:
				switch {
				case msg.Header.Stream == 1 && msg.Header.Function == 13:
					body, _ := (&Item{
						Type: ItemList,
						Children: []*Item{
							{Type: ItemBinary, Raw: []byte{0x00}},
							{Type: ItemList, Children: []*Item{
								{Type: ItemASCII, Raw: []byte("HARNS-EQ")},
								{Type: ItemASCII, Raw: []byte("1.0.0")},
							}},
						},
					}).Encode()
					sendMsg(buildDataMessage(msg.Header.SessionID, 1, 14, false, msg.Header.SystemBytes, body))
				case msg.Header.Stream == 1 && msg.Header.Function == 1:
					body, _ := (&Item{
						Type: ItemList,
						Children: []*Item{
							{Type: ItemASCII, Raw: []byte("ETCHER-01")},
							{Type: ItemASCII, Raw: []byte("REV-2.3")},
						},
					}).Encode()
					sendMsg(buildDataMessage(msg.Header.SessionID, 1, 2, false, msg.Header.SystemBytes, body))
					if !pushedEvent {
						pushedEvent = true
						go func(devID uint16) {
							time.Sleep(30 * time.Millisecond)
							u4 := make([]byte, 4)
							binary.BigEndian.PutUint32(u4, 4201)
							evBody, _ := (&Item{
								Type: ItemList,
								Children: []*Item{
									{Type: ItemUint32, Raw: u4},
									{Type: ItemASCII, Raw: []byte("LOT-99")},
								},
							}).Encode()
							sendMsg(buildDataMessage(devID, 6, 11, true, 0x00090001, evBody))
						}(msg.Header.SessionID)
					}
				case msg.Header.Stream == 1 && msg.Header.Function == 3:
					reqRoot, _, err := DecodeItem(msg.Body)
					if err != nil {
						return
					}
					mu.Lock()
					f32 := make([]byte, 4)
					binary.BigEndian.PutUint32(f32, math.Float32bits(sv1001))
					u32 := make([]byte, 4)
					binary.BigEndian.PutUint32(u32, sv1002)
					mu.Unlock()
					var vals []*Item
					for _, ch := range reqRoot.Children {
						id, _ := itemToUint64(ch)
						if id == 1001 {
							vals = append(vals, &Item{Type: ItemFloat32, Raw: f32})
						} else {
							vals = append(vals, &Item{Type: ItemUint32, Raw: u32})
						}
					}
					body, _ := (&Item{Type: ItemList, Children: vals}).Encode()
					sendMsg(buildDataMessage(msg.Header.SessionID, 1, 4, false, msg.Header.SystemBytes, body))
				case msg.Header.Stream == 6 && msg.Header.Function == 12:
					if msg.Header.SystemBytes == 0x00090001 {
						mu.Lock()
						s6f12Acked = true
						mu.Unlock()
					}
				case msg.Header.Stream == 2 && msg.Header.Function == 15:
					it, _, _ := DecodeItem(msg.Body)
					mu.Lock()
					if it != nil {
						lastWritten = it.FormatSML()
					}
					mu.Unlock()
					ack, _ := (&Item{Type: ItemBinary, Raw: []byte{0x00}}).Encode()
					sendMsg(buildDataMessage(msg.Header.SessionID, 2, 16, false, msg.Header.SystemBytes, ack))
				}
			}
		}
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	ec := 1
	vars := []*collection.Variable{
		{Name: "mdln", Address: "S1F1[1]", DataType: "STRING", AccessMode: "r"},
		{Name: "softrev", Address: "S1F1[2]", DataType: "STRING", AccessMode: "r"},
		{Name: "temp", Address: "S1F3.1001", DataType: "FLOAT32", AccessMode: "r"},
		{Name: "waferCount", Address: "S1F3.1002", DataType: "UINT32", AccessMode: "r"},
		{Name: "eventCeid", Address: "S6F11[1]", DataType: "UINT32", AccessMode: "r"},
		{Name: "eventLot", Address: "S6F11[2]", DataType: "STRING", AccessMode: "r"},
		{Name: "setEc", Address: "S2F15", DataType: "STRING", AccessMode: "rw"},
	}
	c, err := NewCollection(&CreateReq{
		Name:             "hsms-test",
		CollectorCycleMs: 100,
		Variables:        vars,
		ProtocolConfig: ProtocolConfig{
			Host:               host,
			Port:               port,
			DeviceID:           1,
			EstablishComm:      &ec,
			TimeoutMs:          2000,
			LinktestIntervalMs: 500,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Stop(ctx)
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c.GetCollectStatus() == collection.CollectStatusCollecting {
			if v, ok := c.GetCollectionMeta().GetVariableValue("eventLot"); ok && v != nil {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if c.GetCollectStatus() != collection.CollectStatusCollecting {
		t.Fatalf("status=%s", c.GetCollectStatus())
	}

	checkVal := func(name, want string) {
		t.Helper()
		v, ok := c.GetCollectionMeta().GetVariableValue(name)
		if !ok || fmt.Sprint(v) != want {
			t.Fatalf("%s = (%v, %v), want %s", name, v, ok, want)
		}
	}
	checkVal("mdln", "ETCHER-01")
	checkVal("softrev", "REV-2.3")
	checkVal("temp", "24.5")
	checkVal("waferCount", "77")
	checkVal("eventCeid", "4201")
	checkVal("eventLot", "LOT-99")

	// Verify auto S6F12 ACK echoed the equipment's SystemBytes
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	acked := s6f12Acked
	mu.Unlock()
	if !acked {
		t.Fatal("expected S6F12 ACK with echoed SystemBytes")
	}

	// Test DeliverAction with SML List payload on S2F15
	smlPayload := "<L[1] <L[2] <U4 1002> <U4 88> > >."
	if err := c.DeliverAction(ctx, []collection.ActionValue{
		{Index: 0, Variable: vars[6], Value: smlPayload},
	}); err != nil {
		t.Fatalf("DeliverAction failed: %v", err)
	}
	mu.Lock()
	written := lastWritten
	mu.Unlock()
	if written != smlPayload {
		t.Fatalf("written SML = %q, want %q", written, smlPayload)
	}
}
