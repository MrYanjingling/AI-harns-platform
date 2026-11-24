package snmp

import (
	"context"
	"encoding/hex"
	"net"
	"testing"
	"time"
)

func TestParseAddress(t *testing.T) {
	address, err := ParseAddress("public|1.3.6.1.2.1.1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if address.Community != "public" || address.OID != "1.3.6.1.2.1.1.1.0" {
		t.Fatalf("address=%+v", address)
	}
	if hex.EncodeToString(address.Encoded) != "2b06010201010100" {
		t.Fatalf("encoded=%X", address.Encoded)
	}
	for _, value := range []string{"", "public", "|1.3.6", "public|", "public|1", "public|3.1.2"} {
		if _, err := ParseAddress(value); err == nil {
			t.Fatalf("%q should fail", value)
		}
	}
}

func TestBuildGetRequest(t *testing.T) {
	address, _ := ParseAddress("public|1.3.6.1.2.1.1.1.0")
	request := buildGetRequest(address, 1)
	want := "302602010104067075626c6963a019020101020100020100300e300c06082b060102010101000500"
	if hex.EncodeToString(request) != want {
		t.Fatalf("request=%X want=%s", request, want)
	}
}

func TestParseResponse(t *testing.T) {
	response, _ := hex.DecodeString("302c02010104067075626c6963a21f0201010201000201003014301206082b0601020101010004064e6575726f6e")
	value, id, err := parseResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 || value != "Neuron" {
		t.Fatalf("id=%d value=%#v", id, value)
	}
}

func TestDecodeOID(t *testing.T) {
	encoded := []byte{0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x01, 0x00}
	value, err := decodeOID(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if value != "1.3.6.1.2.1.1.1.0" {
		t.Fatalf("oid=%s", value)
	}
	value, err = decodeOID([]byte{0x81, 0x34})
	if err != nil {
		t.Fatal(err)
	}
	if value != "2.100" {
		t.Fatalf("oid=%s", value)
	}
}

func TestClientRead(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	response, _ := hex.DecodeString("302c02010104067075626c6963a21f0201010201000201003014301206082b0601020101010004064e6575726f6e")
	go func() {
		buffer := make([]byte, 1500)
		n, addr, err := server.ReadFromUDP(buffer)
		if err != nil || n == 0 {
			return
		}
		_, _ = server.WriteToUDP(response, addr)
	}()
	address, _ := ParseAddress("public|1.3.6.1.2.1.1.1.0")
	client, err := openClient(context.Background(), "127.0.0.1", server.LocalAddr().(*net.UDPAddr).Port, 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	value, err := client.Read(address)
	if err != nil {
		t.Fatal(err)
	}
	if value != "Neuron" {
		t.Fatalf("value=%#v", value)
	}
}
