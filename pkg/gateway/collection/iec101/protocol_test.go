package iec101

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestFixedFrame(t *testing.T) {
	frame := buildFixedFrame(0x40, 1)
	if !bytes.Equal(frame, []byte{0x10, 0x40, 1, 0x41, 0x16}) {
		t.Fatalf("unexpected frame % X", frame)
	}
}

func TestVariableFrameRoundTrip(t *testing.T) {
	asdu := buildInterrogationASDU(1, 1, 3)
	frame := buildVariableFrame(0x53, 1, asdu)
	ctrl, got, err := readLinkFrame(bytes.NewReader(frame))
	if err != nil {
		t.Fatal(err)
	}
	if ctrl != 0x53 || !bytes.Equal(got, asdu) {
		t.Fatalf("unexpected frame ctrl=%X asdu=% X", ctrl, got)
	}
}

func TestParseFloatASDU(t *testing.T) {
	asdu := []byte{13, 1, 20, 0, 1, 1, 0, 0}
	bits := math.Float32bits(3.5)
	asdu = append(asdu, byte(bits), byte(bits>>8), byte(bits>>16), byte(bits>>24), 0)
	values, _, err := parseASDU(asdu, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := values[1].(float32); !ok || value != 3.5 {
		t.Fatalf("unexpected values %+v", values)
	}
}

func TestConfigKeepsCombinedAddress(t *testing.T) {
	raw := []byte(`{"location":"127.0.0.1:2404"}`)
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	config.Normalize()
	if config.Location != "127.0.0.1:2404" {
		t.Fatalf("unexpected location %q", config.Location)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if !strings.Contains(string(encoded), `"location":"127.0.0.1:2404"`) {
		t.Fatalf("unexpected encoded config %s", encoded)
	}
}
