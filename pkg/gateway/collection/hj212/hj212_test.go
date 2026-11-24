package hj212

import (
	"fmt"
	"strconv"
	"testing"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		address  string
		dataType string
		command  int
		float    bool
		wantErr  bool
	}{
		{"RT!a34013-Rtd", "DOUBLE", 2011, true, false},
		{"RT!a34013-Flag", "STRING", 2011, false, false},
		{"MIN!w01018-Avg", "DOUBLE", 2051, true, false},
		{"HOUR!w01018-Cou", "DOUBLE", 2061, true, false},
		{"DAY!w01018-Flag", "STRING", 2031, false, false},
		{"RT!a34013-Rtd", "STRING", 2011, true, true},
		{"RT!a34013-Flag", "DOUBLE", 2011, false, true},
		{"RT!a34013-Unknown", "DOUBLE", 2011, true, true},
		{"MIN!w01018-SampleTime", "STRING", 2051, false, true},
		{"XX!w01018-Avg", "DOUBLE", 0, false, true},
	}
	for _, test := range tests {
		address, err := ParseAddress(test.address, test.dataType)
		if (err != nil) != test.wantErr {
			t.Fatalf("%s: err=%v", test.address, err)
		}
		if err == nil && (address.Command != test.command || address.Float != test.float) {
			t.Fatalf("%s: got %+v", test.address, address)
		}
	}
}

func TestBuildAndParseResponse(t *testing.T) {
	request := string(buildRequest(2011, "32", "123456", "ABC123"))
	if request[:2] != "##" {
		t.Fatalf("request header: %q", request[:2])
	}
	length, err := strconv.Atoi(request[2:6])
	if err != nil || length != len(request)-12 {
		t.Fatalf("request length: %d %v", length, err)
	}
	if request[len(request)-2:] != "\r\n" {
		t.Fatalf("request tail: %q", request[len(request)-2:])
	}

	data := "QN=20261108120000123;ST=32;CN=2011;PW=123456;MN=ABC123;Flag=4;CP=&&a34013-Rtd=1.23,w01018-Flag=0&&"
	frame := fmt.Sprintf("##%04d%s0000\r\n", len(data), data)
	values, err := parseResponse([]byte(frame))
	if err != nil {
		t.Fatal(err)
	}
	if values["a34013-Rtd"] != "1.23" || values["w01018-Flag"] != "0" {
		t.Fatalf("values: %#v", values)
	}
}
