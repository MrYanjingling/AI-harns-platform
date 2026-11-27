package knx

import (
	"bytes"
	"net"
	"testing"
)

func TestParseAddress(t *testing.T) {
	address, err := ParseAddress("1/2/3,1.2.3", "UINT8")
	if err != nil {
		t.Fatal(err)
	}
	if address.Group != 0x0A03 || address.Individual != 0x1203 || address.NBit != 0 {
		t.Fatalf("address = %+v", address)
	}
	address, err = ParseAddress("31/7/255,15.15.255,4", "UINT8")
	if err != nil {
		t.Fatal(err)
	}
	if address.Group != 0xFFFF || address.Individual != 0xFFFF || address.NBit != 4 {
		t.Fatalf("address = %+v", address)
	}
	if _, err := ParseAddress("1/2/3,1.2.3,2", "UINT16"); err == nil {
		t.Fatal("expected nbit type error")
	}
}

func TestBuildTunnelRead(t *testing.T) {
	client := &knxClient{channel: 1}
	packet := client.buildTunnelRead(0x0A03)
	want := []byte{
		0x06, 0x10, 0x04, 0x20, 0x00, 0x15,
		0x04, 0x01, 0x00, 0x00,
		0x11, 0x00, 0xbc, 0xe0, 0x00, 0x00, 0x0a, 0x03, 0x01, 0x00, 0x00,
	}
	if !bytes.Equal(packet, want) {
		t.Fatalf("packet = %x", packet)
	}
}

func TestParseResponse(t *testing.T) {
	packet := []byte{
		0x06, 0x10, 0x04, 0x20, 0x00, 0x13,
		0x04, 0x01, 0x00, 0x00,
		0x29, 0x00, 0xbc, 0xe0, 0x00, 0x00, 0x0a, 0x03, 0x02, 0x00, 0x40, 0x2a,
	}
	group, data, code, apci, err := parseCEMI(packet)
	if err != nil {
		t.Fatal(err)
	}
	if group != 0x0A03 || code != 0x29 || apci != 1 {
		t.Fatalf("group=%04X code=%02X apci=%d", group, code, apci)
	}
	address, _ := ParseAddress("1/2/3,1.2.3", "UINT8")
	value, err := decodeValue(data, address, "UINT8")
	if err != nil {
		t.Fatal(err)
	}
	if value != uint8(42) {
		t.Fatalf("value = %v", value)
	}
}

func TestDecodeBool(t *testing.T) {
	address, _ := ParseAddress("1/2/3,1.2.3", "BOOL")
	value, err := decodeValue([]byte{0x41}, address, "BOOL")
	if err != nil {
		t.Fatal(err)
	}
	if value != true {
		t.Fatalf("value = %v", value)
	}
}

func TestBuildTunnelWrite(t *testing.T) {
	packet := buildTunnelWrite(1, 0, 0x0A03, []byte{0x81})
	want := []byte{
		0x06, 0x10, 0x04, 0x20, 0x00, 0x15,
		0x04, 0x01, 0x00, 0x00,
		0x11, 0x00, 0xbc, 0xe0, 0x00, 0x00, 0x0a, 0x03, 0x01, 0x00, 0x81,
	}
	if !bytes.Equal(packet, want) {
		t.Fatalf("packet = %x", packet)
	}
}

func TestEncodeWriteValue(t *testing.T) {
	address, _ := ParseAddress("1/2/3,1.2.3,2", "UINT8")
	data, err := encodeWriteValue(address, "UINT8", uint8(3))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x83}) {
		t.Fatalf("uint8 2-bit = %x", data)
	}
	data, err = encodeWriteValue(address, "INT16", int16(-1))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x80, 0xff, 0xff}) {
		t.Fatalf("int16 = %x", data)
	}
	data, err = encodeWriteValue(address, "BOOL", true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x81}) {
		t.Fatalf("bool = %x", data)
	}
}

func TestLocalAddressForResolvesNonZeroIPv4(t *testing.T) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	client := &knxClient{conn: conn}
	dest := &net.UDPAddr{IP: net.IPv4(224, 0, 23, 12), Port: 3671}
	local := client.localAddressFor(dest)
	if local.IP == nil || local.IP.IsUnspecified() {
		t.Fatalf("expected non-zero IPv4 for discovery HPAI, got %v", local.IP)
	}
	if local.Port == 0 {
		t.Fatalf("expected non-zero UDP port, got %d", local.Port)
	}
	hpai := encodeHPAI(local)
	if bytes.Equal(hpai[2:6], []byte{0, 0, 0, 0}) {
		t.Fatalf("encoded HPAI still has 0.0.0.0: %x", hpai)
	}
}
