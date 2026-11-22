package iec103

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseAddress(t *testing.T) {
	ca, fun, num, err := parseAddress("1.16.1")
	if err != nil || ca != 1 || fun != 16 || num != 1 {
		t.Fatalf("unexpected address %d.%d.%d %v", ca, fun, num, err)
	}
	if _, _, _, err := parseAddress("1.2"); err == nil {
		t.Fatal("expected error")
	}
}

func TestVariableFrameRoundTrip(t *testing.T) {
	asdu := []byte{3, 1, 3, 1, 16, 1, 0x08, 0x00, 0x00}
	frame := buildVariableFrame(0x53, 1, asdu)
	got, err := readLinkFrame(bytes.NewReader(frame))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, asdu) {
		t.Fatalf("unexpected ASDU % X", got)
	}
}

func TestParseMeasurementASDU(t *testing.T) {
	values, err := parseASDU([]byte{3, 1, 3, 1, 16, 1, 0x08, 0x00, 0x00})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := values["1.16.1"]; !ok {
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
