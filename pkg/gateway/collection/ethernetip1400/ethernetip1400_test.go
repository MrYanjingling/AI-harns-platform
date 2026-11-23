package ethernetip1400

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func TestParseAddress(t *testing.T) {
	cases := []struct {
		address string
		data    string
		file    byte
		num     byte
		element byte
		bit     byte
		bitSet  bool
	}{
		{"N7:0", "INT16", fileTypeInteger, 7, 0, 0, false},
		{"F8:3", "FLOAT32", fileTypeFloat, 8, 3, 0, false},
		{"B3:0/5", "BIT", fileTypeBinary, 3, 0, 5, true},
		{"B3/17", "BOOL", fileTypeBinary, 3, 1, 1, true},
		{"T4:2", "INT16", fileTypeTimer, 4, 2, 0, false},
		{"C5:0", "INT16", fileTypeCounter, 5, 0, 0, false},
		{"I:0/3", "BIT", fileTypeInput, 1, 0, 3, true},
		{"O:1", "INT16", fileTypeOutput, 0, 1, 0, false},
		{"S:0", "INT16", fileTypeStatus, 2, 0, 0, false},
	}
	for _, tc := range cases {
		address, err := parseAddress(tc.address, tc.data)
		if err != nil {
			t.Fatalf("%s: %v", tc.address, err)
		}
		if address.FileType != tc.file || address.FileNum != tc.num || address.Element != tc.element ||
			address.Bit != tc.bit || address.BitSet != tc.bitSet {
			t.Fatalf("%s: unexpected address %+v", tc.address, address)
		}
	}
	pre, err := parseAddress("T4:0.PRE", "INT16")
	if err != nil || pre.SubElement != subElementPRE || pre.Element != 0 {
		t.Fatalf("unexpected PRE address %+v err=%v", pre, err)
	}
	acc, err := parseAddress("C5:2.ACC", "INT16")
	if err != nil || acc.SubElement != subElementACC || acc.Element != 2 || acc.FileType != fileTypeCounter {
		t.Fatalf("unexpected ACC address %+v err=%v", acc, err)
	}
	for _, value := range []string{"", "X1:0", "N7", "N7:0/1", "B3:0/16", "T4:0.STATUS"} {
		if _, err := parseAddress(value, "INT16"); err == nil && value != "B3:0/16" {
			t.Fatalf("%q should be rejected", value)
		}
	}
	if _, err := parseAddress("B3:0/16", "BIT"); err == nil {
		t.Fatal("bit 16 should be rejected")
	}
	if _, err := parseAddress("N7:0.PRE", "INT16"); err == nil {
		t.Fatal("N file sub-element should be rejected")
	}
}

func TestReadRequestLayout(t *testing.T) {
	address, err := parseAddress("N7:0", "INT16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address, "INT16")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x4B, 0x02, 0x20, 0x67, 0x24, 0x01, 7, 0x1B, 0x05, 0x6B, 0xC5, 0xBC, 0x1A, 0x0F, 0x00, 0x00, 0x00, 0xA2, 0x02, 0x07, 0x89, 0x00, 0x00}
	if !bytes.Equal(request, want) {
		t.Fatalf("unexpected read request % X", request)
	}
}

func TestTimerCounterReadAndWrite(t *testing.T) {
	address, err := parseAddress("T4:0.ACC", "INT16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address, "INT16")
	if err != nil {
		t.Fatal(err)
	}
	if request[18] != 0x06 {
		t.Fatalf("unexpected timer read byte size %d", request[18])
	}
	if _, err := writeRequest(address, int16(1), "INT16"); err == nil {
		t.Fatal("ACC write should require read-modify-write")
	}
	response := []byte{0xCB, 0, 0, 0}
	response = append(response, pcccRequestorID...)
	response = append(response, 0x4F, 0, 0, 0)
	response = append(response, 0x10, 0, 0x20, 0, 0x30, 0)
	value, err := parseReadResponse(response, parseMustAddress(t, "T4:0.PRE", "INT16"), "INT16")
	if err != nil || value != int16(0x10) {
		t.Fatalf("unexpected PRE value %v err=%v", value, err)
	}
	value, err = parseReadResponse(response, address, "INT16")
	if err != nil || value != int16(0x20) {
		t.Fatalf("unexpected ACC value %v err=%v", value, err)
	}
}

func parseMustAddress(t *testing.T, value, dataType string) pcccAddress {
	t.Helper()
	address, err := parseAddress(value, dataType)
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func TestWriteRequestLayout(t *testing.T) {
	address, err := parseAddress("N7:0", "INT16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := writeRequest(address, int16(0x1234), "INT16")
	if err != nil {
		t.Fatal(err)
	}
	wantTail := []byte{0xAB, 0x02, 0x07, 0x89, 0x00, 0x00, 0x34, 0x12, 0xFF, 0xFF}
	if !bytes.HasSuffix(request, wantTail) {
		t.Fatalf("unexpected write request % X", request)
	}
}

func TestFakePCCCRead(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() { done <- serveFake1400(listener) }()

	client, err := newCIPClient(context.Background(), listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	address, err := parseAddress("N7:0", "INT16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address, "INT16")
	if err != nil {
		t.Fatal(err)
	}
	response, _, _, err := client.Exchange(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	value, err := parseReadResponse(response, address, "INT16")
	if err != nil {
		t.Fatal(err)
	}
	if value != int16(42) {
		t.Fatalf("unexpected value %v", value)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("fake server did not exit")
	}
}

func serveFake1400(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	session := uint32(0)
	for {
		command, _, payload, err := readTestEnetPacket(conn)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch command {
		case enetCommandRegisterSession:
			session = 0x12345678
			if err := writeTestEnetPacket(conn, command, session, 0, []byte{1, 0, 0, 0}); err != nil {
				return err
			}
		case enetCommandSendRRData:
			if _, err := parseUnconnectedData(payload); err != nil {
				return err
			}
			response := []byte{0xD4, 0, 0, 0}
			body := make([]byte, 26)
			binary.LittleEndian.PutUint32(body[0:], 0x11111111)
			binary.LittleEndian.PutUint32(body[4:], 0x22222222)
			response = append(response, body...)
			reply := buildCommandHeader(1)
			reply = append(reply, buildItem(enetItemUnconnectedData, response)...)
			if err := writeTestEnetPacket(conn, command, session, 0, reply); err != nil {
				return err
			}
		case enetCommandSendUnitData:
			data, err := parseItemData(payload, enetItemConnectedData)
			if err != nil {
				return err
			}
			if len(data) < 3 {
				return io.ErrUnexpectedEOF
			}
			seq := binary.LittleEndian.Uint16(data)
			cip := []byte{0xCB, 0, 0, 0}
			cip = append(cip, pcccRequestorID...)
			cip = append(cip, 0x4F, 0x00, 0x00, 0x00)
			cip = append(cip, 0x2A, 0x00)
			replyData := make([]byte, 2+len(cip))
			binary.LittleEndian.PutUint16(replyData, seq)
			copy(replyData[2:], cip)
			reply := buildCommandHeader(2)
			connID := make([]byte, 4)
			binary.LittleEndian.PutUint32(connID, 0x22222222)
			reply = append(reply, buildItem(enetItemConnectionBased, connID)...)
			reply = append(reply, buildItem(enetItemConnectedData, replyData)...)
			if err := writeTestEnetPacket(conn, command, session, 0, reply); err != nil {
				return err
			}
		default:
			return io.ErrUnexpectedEOF
		}
	}
}

func readTestEnetPacket(conn net.Conn) (uint16, uint32, []byte, error) {
	header := make([]byte, enetHeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, 0, nil, err
	}
	length := int(binary.LittleEndian.Uint16(header[2:4]))
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, 0, nil, err
	}
	return binary.LittleEndian.Uint16(header), binary.LittleEndian.Uint32(header[4:8]), payload, nil
}

func writeTestEnetPacket(conn net.Conn, command uint16, session uint32, context uint64, payload []byte) error {
	_, err := conn.Write(buildEnetPacket(command, session, context, payload))
	return err
}

func TestCollectionMarksUnconnectedWhenConnectionDrops(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() { _ = serveFake1400HandshakeOnly(listener) }()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port := 0
	for _, ch := range portText {
		port = port*10 + int(ch-'0')
	}
	item, err := NewCollection(&CreateReq{
		Name:             "drop-test",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Location: host, Port: port},
		Variables: []*collection.Variable{{
			Name: "tag", DataType: "INT16", Address: "N7:0", AccessMode: collection.AccessModeRead,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := item.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if item.GetCollectStatus() == collection.CollectStatusUnconnected {
			_ = item.Stop(context.Background())
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	status := item.GetCollectStatus()
	_ = item.Stop(context.Background())
	t.Fatalf("collection status is %s, want unconnected", status)
}

func serveFake1400HandshakeOnly(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	session := uint32(0)
	for {
		command, _, payload, err := readTestEnetPacket(conn)
		if err != nil {
			return err
		}
		switch command {
		case enetCommandRegisterSession:
			session = 0x12345678
			if err := writeTestEnetPacket(conn, command, session, 0, []byte{1, 0, 0, 0}); err != nil {
				return err
			}
		case enetCommandSendRRData:
			if _, err := parseUnconnectedData(payload); err != nil {
				return err
			}
			response := []byte{0xD4, 0, 0, 0}
			body := make([]byte, 26)
			binary.LittleEndian.PutUint32(body[0:], 0x11111111)
			binary.LittleEndian.PutUint32(body[4:], 0x22222222)
			response = append(response, body...)
			reply := buildCommandHeader(1)
			reply = append(reply, buildItem(enetItemUnconnectedData, response)...)
			_ = writeTestEnetPacket(conn, command, session, 0, reply)
			return nil
		default:
			return nil
		}
	}
}

func TestTimerCounterACCRMWRequest(t *testing.T) {
	address, err := parseAddress("T4:0.ACC", "INT16")
	if err != nil {
		t.Fatal(err)
	}
	current := []byte{0x10, 0x00, 0x20, 0x00, 0x30, 0x00}
	request, err := timerCounterWriteRequest(address, current, int16(0x1234), "INT16")
	if err != nil {
		t.Fatal(err)
	}
	if request[18] != 6 {
		t.Fatalf("unexpected byte size %d", request[18])
	}
	if !bytes.Equal(request[23:29], []byte{0x10, 0x00, 0x34, 0x12, 0x30, 0x00}) {
		t.Fatalf("unexpected element data % X", request[23:29])
	}
	if !bytes.Equal(request[29:35], []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) {
		t.Fatalf("unexpected mask % X", request[29:35])
	}
}

func TestTimerCounterACCRMWAction(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	writes := make(chan []byte, 1)
	go func() { _ = serveFake1400RMW(listener, writes) }()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port := 0
	for _, ch := range portText {
		port = port*10 + int(ch-'0')
	}
	item, err := NewCollection(&CreateReq{
		Name:             "rmw-test",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Location: host, Port: port},
		Variables: []*collection.Variable{{
			Name: "timer_acc", DataType: "INT16", Address: "T4:0.ACC", AccessMode: collection.AccessModeReadWrite,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	variable := item.GetCollectionMeta().GetVariables()[0]
	err = item.DeliverAction(context.Background(), []collection.ActionValue{{
		Index: 0, Variable: variable, Value: int16(0x1234),
	}})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case request := <-writes:
		if request[18] != 6 {
			t.Fatalf("unexpected byte size %d", request[18])
		}
		if !bytes.Equal(request[23:29], []byte{0x10, 0x00, 0x34, 0x12, 0x30, 0x00}) {
			t.Fatalf("unexpected RMW data % X", request[23:29])
		}
		if !bytes.Equal(request[29:35], []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) {
			t.Fatalf("unexpected RMW mask % X", request[29:35])
		}
	case <-time.After(time.Second):
		t.Fatal("expected RMW write request")
	}
	if err := item.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func serveFake1400RMW(listener net.Listener, writes chan<- []byte) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	session := uint32(0)
	for {
		command, _, payload, err := readTestEnetPacket(conn)
		if err != nil {
			return nil
		}
		switch command {
		case enetCommandRegisterSession:
			session = 0x12345678
			if err := writeTestEnetPacket(conn, command, session, 0, []byte{1, 0, 0, 0}); err != nil {
				return err
			}
		case enetCommandSendRRData:
			if _, err := parseUnconnectedData(payload); err != nil {
				return err
			}
			response := []byte{0xD4, 0, 0, 0}
			body := make([]byte, 26)
			binary.LittleEndian.PutUint32(body[0:], 0x11111111)
			binary.LittleEndian.PutUint32(body[4:], 0x22222222)
			response = append(response, body...)
			reply := buildCommandHeader(1)
			reply = append(reply, buildItem(enetItemUnconnectedData, response)...)
			if err := writeTestEnetPacket(conn, command, session, 0, reply); err != nil {
				return err
			}
		case enetCommandSendUnitData:
			data, err := parseItemData(payload, enetItemConnectedData)
			if err != nil {
				return err
			}
			if len(data) < 20 {
				return io.ErrUnexpectedEOF
			}
			seq := binary.LittleEndian.Uint16(data)
			cip := data[2:]
			function := cip[17]
			var cipResponse []byte
			if function == pcccServiceWrite {
				select {
				case writes <- append([]byte(nil), cip...):
				default:
				}
				cipResponse = []byte{0xCB, 0, 0, 0}
				cipResponse = append(cipResponse, pcccRequestorID...)
				cipResponse = append(cipResponse, 0x4F, 0x00, 0x00, 0x00)
			} else {
				cipResponse = []byte{0xCB, 0, 0, 0}
				cipResponse = append(cipResponse, pcccRequestorID...)
				cipResponse = append(cipResponse, 0x4F, 0x00, 0x00, 0x00)
				cipResponse = append(cipResponse, 0x10, 0, 0x20, 0, 0x30, 0)
			}
			replyData := make([]byte, 2+len(cipResponse))
			binary.LittleEndian.PutUint16(replyData, seq)
			copy(replyData[2:], cipResponse)
			reply := buildCommandHeader(2)
			connID := make([]byte, 4)
			binary.LittleEndian.PutUint32(connID, 0x22222222)
			reply = append(reply, buildItem(enetItemConnectionBased, connID)...)
			reply = append(reply, buildItem(enetItemConnectedData, replyData)...)
			if err := writeTestEnetPacket(conn, command, session, 0, reply); err != nil {
				return err
			}
		default:
			return io.ErrUnexpectedEOF
		}
	}
}
