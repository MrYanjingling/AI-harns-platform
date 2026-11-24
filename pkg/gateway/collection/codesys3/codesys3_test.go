package codesys3

import (
	"bytes"
	"testing"
)

func TestTagRoundTrip(t *testing.T) {
	items := []*tag{
		u32Tag(0x22, 1),
		containerTag(0x81, cstrTag(0x10, "admin"), u32Tag(0x13, 7)),
	}
	encoded := encodeTags(items)
	decoded, err := parseTags(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 || decoded[0].ID != 0x22 || decoded[1].ID != 0x81 {
		t.Fatalf("decoded = %+v", decoded)
	}
	session := findTag(decoded, 0x13)
	if session == nil {
		t.Fatal("missing nested tag")
	}
	value, _ := tagU32(session)
	if value != 7 {
		t.Fatalf("value = %d", value)
	}
}

func TestBlockFrameRoundTrip(t *testing.T) {
	payload := []byte{1, 2, 3, 4}
	frame := blockFrame(payload)
	got, err := parseBlockFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %x", got)
	}
}

func TestRouterRoundTrip(t *testing.T) {
	src := nodeAddress("127.0.0.1", 11740, 0x2ddf)
	dst := nodeAddress("127.0.0.1", 11740, 0)
	packet := buildRouterDatagram([]byte{0xAA}, src, dst, serviceChannel, 1)
	service, payload, gotSrc, err := parseRouterDatagram(packet)
	if err != nil {
		t.Fatal(err)
	}
	if service != serviceChannel || !bytes.Equal(payload, []byte{0xAA}) || !bytes.Equal(gotSrc, src) {
		t.Fatalf("service=%d payload=%x src=%x", service, payload, gotSrc)
	}
}

func TestPasswordHashLength(t *testing.T) {
	if got := hashPassword(""); len(got) != 32 {
		t.Fatalf("empty hash length = %d", len(got))
	}
	if got := hashPassword("password"); len(got) != 32 {
		t.Fatalf("hash length = %d", len(got))
	}
}
