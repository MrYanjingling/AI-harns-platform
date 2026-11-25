package knd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"

	"harns-platform/pkg/gateway/collection"
)

func TestParseAddress(t *testing.T) {
	address, err := ParseAddress("R100", "UINT16")
	if err != nil || address.Kind != kindPLC || address.Area != "R" || address.Number != 100 || address.Size != 2 {
		t.Fatalf("address = %+v err=%v", address, err)
	}
	address, err = ParseAddress("T5.conf", "INT32")
	if err != nil || address.Area != "T" || address.Suffix != "conf" || address.Size != 4 {
		t.Fatalf("timer = %+v err=%v", address, err)
	}
	address, err = ParseAddress("X0.3", "BIT")
	if err != nil || address.Bit != 3 || address.Size != 1 {
		t.Fatalf("bit = %+v err=%v", address, err)
	}
	address, err = ParseAddress("g54.X", "FLOAT")
	if err != nil || address.Kind != kindWorkCoors || address.Number != 54 || address.Axis != 0 {
		t.Fatalf("coor = %+v err=%v", address, err)
	}
	address, err = ParseAddress("systemStatus.run-status", "UINT32")
	if err != nil || address.Kind != kindSystemStatus || address.Key != "run-status" {
		t.Fatalf("status = %+v err=%v", address, err)
	}
}

func TestAddressPath(t *testing.T) {
	address, _ := ParseAddress("R100", "UINT16")
	if address.path() != "/v1/plc/vm/R100?size=2" {
		t.Fatalf("path = %s", address.path())
	}
	address, _ = ParseAddress("vars.3", "INT32")
	if address.path() != "/v1/vars/3" {
		t.Fatalf("path = %s", address.path())
	}
	address, _ = ParseAddress("absolute.X", "FLOAT")
	if address.path() != "/v1/coors/absolute" {
		t.Fatalf("path = %s", address.path())
	}
}

func TestDecodePLCResponse(t *testing.T) {
	address, _ := ParseAddress("R100", "UINT16")
	value, err := decodePLCResponse([]any{float64(0x34), float64(0x12)}, address, "UINT16")
	if err != nil {
		t.Fatal(err)
	}
	if value != uint16(0x1234) {
		t.Fatalf("value = %v", value)
	}
}

func TestHTTPReadWrite(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/status":
			body, _ := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(`{"run-status":3}`))
			_, _ = w.Write(body)
		case "/v1/plc/vm/R100":
			body, _ := io.ReadAll(r.Body)
			decoded, _ := simplifiedchinese.GBK.NewDecoder().Bytes(body)
			bodies = append(bodies, string(decoded))
			_, _ = w.Write([]byte(`{"error":0}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	parsed, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(parsed.Port())
	client, err := openKNDClient(parsed.Hostname(), port, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	address, _ := ParseAddress("systemStatus.run-status", "UINT32")
	value, err := client.Read(context.Background(), address, &collection.Variable{DataType: "UINT32"})
	if err != nil || fmt.Sprint(value) != "3" {
		t.Fatalf("value=%v err=%v", value, err)
	}

	address, _ = ParseAddress("R100", "UINT16")
	err = client.Write(context.Background(), address, &collection.Variable{DataType: "UINT16"}, uint16(513))
	if err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 1 || strings.TrimSpace(bodies[0]) != "[1,2]" {
		t.Fatalf("body = %v", bodies)
	}
}

func TestEncodeWriteValue(t *testing.T) {
	data, err := encodePLCBytes(uint16(0x1234), "UINT16", 2)
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := json.Marshal([]int{0x34, 0x12})
	if string(data) != string([]byte{0x34, 0x12}) {
		t.Fatalf("data=%v expected=%s", data, expected)
	}
}

func TestEndpointConfiguration(t *testing.T) {
	item, err := NewCollection(&CreateReq{
		Name: "knd", CollectorCycleMs: 1000,
		ProtocolConfig: ProtocolConfig{Location: "http://192.168.1.10:8000"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Host != "192.168.1.10" || item.Port != 8000 || item.Location != "http://192.168.1.10:8000" {
		t.Fatalf("endpoint = %+v", item)
	}
	raw, err := item.ConfigJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"location":"http://192.168.1.10:8000","timeoutMs":5000,"intervalMs":20}` {
		t.Fatalf("config = %s", raw)
	}

	legacy := &Collection{}
	if err := legacy.ApplyConfig([]byte(`{"host":"10.0.0.2","port":9000}`)); err != nil {
		t.Fatal(err)
	}
	if legacy.Host != "10.0.0.2" || legacy.Port != 9000 || legacy.Location != "http://10.0.0.2:9000" {
		t.Fatalf("legacy endpoint = %+v", legacy)
	}
	if _, err := NewCollection(&CreateReq{Name: "bad", CollectorCycleMs: 1, ProtocolConfig: ProtocolConfig{Location: "https://10.0.0.2:8000"}}); err == nil {
		t.Fatal("https endpoint should be rejected")
	}
}
