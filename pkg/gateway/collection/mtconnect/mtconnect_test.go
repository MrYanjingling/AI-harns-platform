package mtconnect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

const sampleXML = `<MTConnectStreams xmlns="urn:mtconnect.org:MTConnectStreams:1.7"><Streams><DeviceStream name="Mill"><ComponentStream component="Device"><Samples><Position dataItemId="Xpos">123.456</Position></Samples></ComponentStream></DeviceStream></Streams></MTConnectStreams>`

func TestFindXPath(t *testing.T) {
	root, err := parseXML([]byte(sampleXML))
	if err != nil {
		t.Fatal(err)
	}
	text, err := findXPath(root, `//DeviceStream/ComponentStream/Samples/Position[@dataItemId="Xpos"]`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "123.456" {
		t.Fatalf("text = %q", text)
	}
	value, err := convertValue(text, "DOUBLE")
	if err != nil || value != 123.456 {
		t.Fatalf("value=%v err=%v", value, err)
	}
}

func TestHTTPRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/current" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(sampleXML))
	}))
	defer server.Close()
	parsed, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(parsed.Port())
	client, err := openMTCClient(parsed.Hostname(), port, "m", "urn:mtconnect.org:MTConnectStreams:1.7", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	value, err := client.Read(context.Background(), `//Position[@dataItemId="Xpos"]`, "DOUBLE")
	if err != nil || value != 123.456 {
		t.Fatalf("value=%v err=%v", value, err)
	}
}

func TestConvertValues(t *testing.T) {
	if value, err := convertValue("true", "BOOL"); err != nil || value != true {
		t.Fatalf("bool=%v err=%v", value, err)
	}
	if value, err := convertValue("42", "INT32"); err != nil || value != int64(42) {
		t.Fatalf("int=%v err=%v", value, err)
	}
}
