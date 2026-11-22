package iec102

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseAddress(t *testing.T) {
	ra, ioa, err := parseAddress("10.255")
	if err != nil || ra != 10 || ioa != 255 {
		t.Fatalf("unexpected address %d.%d %v", ra, ioa, err)
	}
	if _, _, err := parseAddress("1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestVariableFrameRoundTrip(t *testing.T) {
	asdu := []byte{1, 1, 3, 0, 1, 2, 3, 1}
	frame := buildVariableFrame(0x53, 1, asdu)
	got, err := readLinkFrame(bytes.NewReader(frame))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, asdu) {
		t.Fatalf("unexpected ASDU % X", got)
	}
}

func TestParseASDU(t *testing.T) {
	values, err := parseASDU([]byte{1, 1, 3, 0, 1, 2, 3, 1})
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := values["2.3"]; !ok || value != true {
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
