package hostlinkcmode

import (
	"errors"
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestParseAddress(t *testing.T) {
	parsed, err := parseAddress("00!CIO0000", "int16")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.unit != "00" || parsed.readCommand != "RR" || parsed.writeCommand != "WR" || parsed.address != 0 || parsed.length != 1 {
		t.Fatalf("unexpected address %+v", parsed)
	}
	parsed, err = parseAddress("1!D100", "int32")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.unit != "01" || parsed.readCommand != "RD" || parsed.address != 100 || parsed.length != 2 {
		t.Fatalf("unexpected address %+v", parsed)
	}
	parsed, err = parseAddress("00!D0000.10H", "string")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.stringLength != 10 || parsed.length != 5 || !parsed.stringHigh {
		t.Fatalf("unexpected string address %+v", parsed)
	}
	if _, err := parseAddress("00!CIO0000.10", "string"); err == nil {
		t.Fatal("expected string direction error")
	}
}

func TestReadFrameAndResponse(t *testing.T) {
	address, err := parseAddress("00!CIO0000", "int16")
	if err != nil {
		t.Fatal(err)
	}
	request := string(buildReadRequest(address))
	if request != "@00RR0000000141*\r" {
		t.Fatalf("unexpected request %q", request)
	}
	response, err := parseResponse([]byte("@00RR00000A31*\r"), address)
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeResponse(response, address, "int16")
	if err != nil {
		t.Fatal(err)
	}
	if value != int16(10) {
		t.Fatalf("unexpected value %v", value)
	}
}

func TestWriteFrame(t *testing.T) {
	address, err := parseAddress("00!CIO0000", "int16")
	if err != nil {
		t.Fatal(err)
	}
	data, err := encodeResponseData(int16(255), address, "int16")
	if err != nil {
		t.Fatal(err)
	}
	request := string(buildWriteRequest(address, data))
	if request != "@00WR000000FF45*\r" {
		t.Fatalf("unexpected request %q", request)
	}
	if _, err := parseResponse([]byte("@00WR13A5*\r"), address); err == nil {
		t.Fatal("expected end code error")
	} else {
		var deviceErr *collection.DeviceError
		if !errors.As(err, &deviceErr) || deviceErr.Code != "13" {
			t.Fatalf("unexpected end code error %v", err)
		}
	}
}

func TestWriteEndCode16ReturnsDeviceError(t *testing.T) {
	address, err := parseAddress("00!TS0000", "bool")
	if err != nil {
		t.Fatal(err)
	}
	_, err = parseResponse([]byte("@00WG1657*\r"), address)
	var deviceErr *collection.DeviceError
	if !errors.As(err, &deviceErr) {
		t.Fatalf("expected device error, got %v", err)
	}
	if deviceErr.Code != "16" || deviceErr.MessageID != "error.hostlink_end_code_16" {
		t.Fatalf("unexpected device error %+v", deviceErr)
	}
}

func TestBitAddress(t *testing.T) {
	address, err := parseAddress("00!TS0000", "bool")
	if err != nil {
		t.Fatal(err)
	}
	if !address.isBit || address.readCommand != "RG" || address.writeCommand != "WG" {
		t.Fatalf("unexpected bit address %+v", address)
	}
	value, err := decodeResponse(response{command: "RG", data: "1"}, address, "bool")
	if err != nil {
		t.Fatal(err)
	}
	if value != true {
		t.Fatalf("unexpected value %v", value)
	}
}

func TestBuildReadFramesMergesContiguousWords(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", DataType: "int16", Address: "00!D0000", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "int16", Address: "00!D0001", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildReadFrames(variables, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 {
		t.Fatalf("expected one frame, got %d", len(frames))
	}
	if frames[0].Address.length != 2 || len(frames[0].Variables) != 2 {
		t.Fatalf("unexpected frame %+v", frames[0])
	}
	parsed, err := parseResponse([]byte("@00RD00000A001400*\r"), frames[0].Address)
	if err != nil {
		t.Fatal(err)
	}
	values, err := decodeFrame(frames[0], parsed)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].Value != int16(10) || values[1].Value != int16(20) {
		t.Fatalf("unexpected values %+v", values)
	}
}

func TestBuildReadFramesSplitsAtLimit(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", DataType: "int16", Address: "00!D0000", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "int16", Address: "00!D0001", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildReadFrames(variables, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("expected two frames, got %d", len(frames))
	}
}

func TestBuildReadFramesMergesBits(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", DataType: "bool", Address: "00!TS0000", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "bool", Address: "00!TS0001", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildReadFrames(variables, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 || frames[0].Address.length != 2 {
		t.Fatalf("unexpected frames %+v", frames)
	}
	values, err := decodeFrame(frames[0], response{command: "RG", data: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].Value != true || values[1].Value != false {
		t.Fatalf("unexpected values %+v", values)
	}
}
