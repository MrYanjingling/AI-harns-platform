package ethernetip5500

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

func TestParseAddressAndRequest(t *testing.T) {
	cases := []struct {
		address  string
		dataType string
		segments []string
		index    byte
		bit      uint8
		bitSet   bool
		typeCode uint16
	}{
		{"Tag", "int16", []string{"Tag"}, 0, 0, false, 0xC3},
		{"Program:MainProgram.Counter", "DINT", []string{"Program:MainProgram", "Counter"}, 0, 0, false, 0xC4},
		{"MyArray[3]", "INT16", []string{"MyArray"}, 3, 0, false, 0xC3},
		{"Flags.5", "BOOL", []string{"Flags", "5"}, 0, 0, false, 0xC1},
		{"Analog", "REAL", []string{"Analog"}, 0, 0, false, 0xCA},
	}
	for _, tc := range cases {
		parsed, err := parseAddress(tc.address, tc.dataType)
		if err != nil {
			t.Fatalf("%s: %v", tc.address, err)
		}
		if len(parsed.segments) != len(tc.segments) {
			t.Fatalf("%s: segments %+v", tc.address, parsed.segments)
		}
		for index, symbol := range tc.segments {
			if parsed.segments[index].symbol != symbol {
				t.Fatalf("%s: symbol %q", tc.address, parsed.segments[index].symbol)
			}
		}
		if parsed.bit != tc.bit || parsed.bitSet != tc.bitSet {
			t.Fatalf("%s: bit %+v", tc.address, parsed)
		}
		if parsed.typeCode != tc.typeCode {
			t.Fatalf("%s: type code %X", tc.address, parsed.typeCode)
		}
	}
	address, err := parseAddress("MyArray[3]", "INT16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x4C, 0x06, 0x91, 0x07, 'M', 'y', 'A', 'r', 'r', 'a', 'y', 0, 0x28, 0x03, 1, 0}
	if !bytes.Equal(request, want) {
		t.Fatalf("unexpected read request % X", request)
	}
}

func TestForwardOpenPacketLayout(t *testing.T) {
	body := buildForwardOpenBody(7)
	if len(body) != 42 {
		t.Fatalf("unexpected body size %d", len(body))
	}
	if body[34] != 0xA3 || body[35] != 3 || body[36] != 1 || body[37] != 7 {
		t.Fatalf("unexpected connection path % X", body[34:])
	}
	cip := buildForwardOpenCIP(7)
	if cip[0] != 0x54 || cip[1] != 2 || !bytes.Equal(cip[2:6], []byte{0x20, 0x06, 0x24, 0x01}) {
		t.Fatalf("unexpected forward open cip % X", cip[:6])
	}
}

func TestFakeConnectedServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() { done <- serveFake5500(listener) }()

	client, err := newCIPClient(context.Background(), listener.Addr().String(), 7, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if client.session != 0x12345678 || client.otConnectionID != 0x11111111 {
		t.Fatalf("unexpected session/connection %X %X", client.session, client.otConnectionID)
	}
	address, err := parseAddress("Tag", "INT16")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address)
	if err != nil {
		t.Fatal(err)
	}
	response, _, _, err := client.Exchange(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	data, _, err := parseReadResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeValue(data, "INT16", address)
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

func serveFake5500(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		command, session, payload, err := readTestEnetPacket(conn)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch command {
		case enetCommandRegisterSession:
			if err := writeTestEnetPacket(conn, command, 0x12345678, 0, []byte{1, 0, 0, 0}); err != nil {
				return err
			}
		case enetCommandSendRRData:
			_ = session
			cip, err := parseUnconnectedData(payload)
			if err != nil {
				return err
			}
			if len(cip) < 2 || cip[0] != enetForwardOpenService {
				return io.ErrUnexpectedEOF
			}
			response := []byte{0xD4, 0, 0, 0}
			body := make([]byte, 26)
			binary.LittleEndian.PutUint32(body[0:], 0x11111111)
			binary.LittleEndian.PutUint32(body[4:], 0x22222222)
			response = append(response, body...)
			reply := buildCommandHeader(1)
			reply = append(reply, buildItem(enetItemUnconnectedData, response)...)
			if err := writeTestEnetPacket(conn, command, 0x12345678, 0, reply); err != nil {
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
			replyCIP := []byte{0xCC, 0, 0, 0, 0xC3, 0, 0x2A, 0}
			replyData := make([]byte, 2+len(replyCIP))
			binary.LittleEndian.PutUint16(replyData, seq)
			copy(replyData[2:], replyCIP)
			reply := buildCommandHeader(2)
			connID := make([]byte, 4)
			binary.LittleEndian.PutUint32(connID, 0x22222222)
			reply = append(reply, buildItem(enetItemConnectionBased, connID)...)
			reply = append(reply, buildItem(enetItemConnectedData, replyData)...)
			if err := writeTestEnetPacket(conn, command, 0x12345678, 0, reply); err != nil {
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
	go func() { _ = serveFake5500HandshakeOnly(listener) }()

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
			Name: "tag", DataType: "INT16", Address: "Tag", AccessMode: collection.AccessModeRead,
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

func serveFake5500HandshakeOnly(listener net.Listener) error {
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
