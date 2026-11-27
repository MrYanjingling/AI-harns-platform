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

func TestParseSequentialASDU(t *testing.T) {
	// Type 13 (M_ME_NC_1), SQ=1 | count=2 => VSQ=0x82, COT=20, CA=1 (2 bytes), base IOA=100 (3 bytes).
	asdu := []byte{typeFloatMeasure, 0x82, 20, 0, 1, 0, 100, 0, 0}
	for _, f := range []float32{1.25, -4.5} {
		bits := math.Float32bits(f)
		asdu = append(asdu, byte(bits), byte(bits>>8), byte(bits>>16), byte(bits>>24), 0)
	}
	values, _, err := parseASDU(asdu, 2, 3)
	if err != nil {
		t.Fatalf("parse sequential float ASDU: %v", err)
	}
	if v, ok := values[100].(float32); !ok || v != 1.25 {
		t.Fatalf("unexpected IOA 100: %#v", values[100])
	}
	if v, ok := values[101].(float32); !ok || v != -4.5 {
		t.Fatalf("unexpected IOA 101: %#v", values[101])
	}

	// Type 1 (M_SP_NA_1), SQ=1 | count=3 => VSQ=0x83, COT=20, CA=1 (2 bytes), base IOA=2000 (0x07D0, 3 bytes).
	sp := []byte{typeSinglePoint, 0x83, 20, 0, 1, 0, 0xD0, 0x07, 0x00, 0x01, 0x00, 0x01}
	spValues, _, err := parseASDU(sp, 2, 3)
	if err != nil {
		t.Fatalf("parse sequential single point ASDU: %v", err)
	}
	if spValues[2000] != true || spValues[2001] != false || spValues[2002] != true {
		t.Fatalf("unexpected single point values: %+v", spValues)
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
