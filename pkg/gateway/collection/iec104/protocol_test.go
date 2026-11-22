package iec104

import (
	"encoding/json"
	"math"
	"net"
	"strings"
	"testing"
	"time"
)

func TestInterrogationASDU(t *testing.T) {
	asdu := buildInterrogationASDU(1)
	if len(asdu) != 10 || asdu[0] != typeInterrogation || asdu[9] != 20 {
		t.Fatalf("unexpected ASDU % X", asdu)
	}
}

func TestParseSinglePointASDU(t *testing.T) {
	asdu := []byte{1, 0x81, 20, 0, 1, 0, 0xD0, 0x07, 0, 1}
	values, termination, err := parseASDU(asdu, 1)
	if err != nil || termination {
		t.Fatal(err)
	}
	if value, ok := values[2000]; !ok || value != true {
		t.Fatalf("unexpected values %+v", values)
	}
}

func TestParseFloatASDU(t *testing.T) {
	data := make([]byte, 8)
	data[0], data[1], data[2] = 1, 0, 0
	bits := math.Float32bits(12.5)
	data[3], data[4], data[5], data[6] = byte(bits), byte(bits>>8), byte(bits>>16), byte(bits>>24)
	asdu := append([]byte{13, 1, 20, 0, 1, 0}, data...)
	values, _, err := parseASDU(asdu, 1)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := values[1].(float32); !ok || value != 12.5 {
		t.Fatalf("unexpected values %+v", values)
	}
}

func TestParseSequentialSinglePointASDU(t *testing.T) {
	asdu := []byte{1, 0x83, 20, 0, 1, 0, 0xD0, 0x07, 0, 0, 1, 1}
	values, termination, err := parseASDU(asdu, 1)
	if err != nil || termination {
		t.Fatal(err)
	}
	if len(values) != 3 || values[2000] != false || values[2001] != true || values[2002] != true {
		t.Fatalf("unexpected values %+v", values)
	}
}

func TestParseSequentialFloatASDU(t *testing.T) {
	asdu := []byte{13, 0x82, 20, 0, 1, 0, 1, 0, 0}
	for _, value := range []float32{12.5, -3.25} {
		bits := math.Float32bits(value)
		asdu = append(asdu, byte(bits), byte(bits>>8), byte(bits>>16), byte(bits>>24), 0)
	}
	values, _, err := parseASDU(asdu, 1)
	if err != nil {
		t.Fatal(err)
	}
	if values[1] != float32(12.5) || values[2] != float32(-3.25) {
		t.Fatalf("unexpected values %+v", values)
	}
}

func TestSetpointASDUIncludesQOS(t *testing.T) {
	asdu := buildSetpointASDU(1, 1, 11.5)
	if len(asdu) != 14 || asdu[13] != 0 {
		t.Fatalf("unexpected setpoint ASDU % X", asdu)
	}
}

func TestInterrogateSkipsSupervisoryFrames(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	defer clientConn.Close()

	client := newClient(clientConn, 1, time.Second)
	done := make(chan struct{})
	go func() {
		defer close(done)
		request := make([]byte, 64)
		if _, err := server.Read(request); err != nil {
			return
		}
		if _, err := server.Write(buildSFrame(0)); err != nil {
			return
		}
		termination := []byte{typeInterrogation, 1, 10, 0, 1, 0, 0, 0, 0, 20}
		if _, err := server.Write(buildIFrame(termination, 0, 0)); err != nil {
			return
		}
		drain := make([]byte, 64)
		for {
			if _, err := server.Read(drain); err != nil {
				return
			}
		}
	}()

	values, err := client.interrogate()
	if err != nil {
		t.Fatalf("interrogate returned error on supervisory frame: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("unexpected values %+v", values)
	}
	_ = client.close()
	<-done
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
