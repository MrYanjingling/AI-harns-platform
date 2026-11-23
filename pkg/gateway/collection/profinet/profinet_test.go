package profinet

import (
	"encoding/json"
	"strings"
	"testing"

	"harns-platform/pkg/gateway/collection"
)

func TestCreateHandlerBuildsProfinetIdentity(t *testing.T) {
	handler := NewCreateHandler()
	if handler.DriverType() != DriverType || handler.DriverModel() != DriverModel {
		t.Fatalf("unexpected handler identity %s/%s", handler.DriverType(), handler.DriverModel())
	}
	body := []byte(`{
		"name": "profinet-plc",
		"collectorCycleMs": 1000,
		"protocolConfig": {"location": "192.168.1.10", "rack": 0, "slot": 1},
		"variables": [
			{"name": "temperature", "dataType": "FLOAT32", "address": "DB1.DBD0", "accessMode": "r"},
			{"name": "running", "dataType": "BOOL", "address": "DB1.DBX4.0", "accessMode": "rw"}
		]
	}`)
	request, err := handler.DecodeCreate(body)
	if err != nil {
		t.Fatal(err)
	}
	collector, raw, err := handler.BuildCreate(request)
	if err != nil {
		t.Fatal(err)
	}
	meta := collector.GetCollectionMeta()
	if meta.GetDriverType() != DriverType || meta.GetDriverModel() != DriverModel {
		t.Fatalf("unexpected collection identity %s/%s", meta.GetDriverType(), meta.GetDriverModel())
	}
	if meta.GetName() != "profinet-plc" {
		t.Fatalf("unexpected name %q", meta.GetName())
	}
	if len(meta.GetVariables()) != 2 {
		t.Fatalf("unexpected variable count %d", len(meta.GetVariables()))
	}
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	if config["location"] != "192.168.1.10" || config["port"] != float64(102) {
		t.Fatalf("unexpected config %s", raw)
	}
}

func TestValidateRejectsMissingLocation(t *testing.T) {
	var req CreateReq
	req.Name = "profinet"
	req.CollectorCycleMs = 1000
	req.ProtocolConfig.Location = ""
	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "location") && !strings.Contains(err.Error(), "validate.") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestCollectorFactoryIdentity(t *testing.T) {
	factory := NewCollectorFactory()
	if factory.DriverType() != DriverType || factory.DriverModel() != DriverModel {
		t.Fatalf("unexpected factory identity %s/%s", factory.DriverType(), factory.DriverModel())
	}
	meta := &collection.CollectionMeta{DriverType: DriverType, DriverModel: DriverModel}
	collector, err := factory.NewCollector(meta, json.RawMessage(`{"location":"192.168.1.10","port":102}`))
	if err != nil {
		t.Fatal(err)
	}
	if collector.GetCollectionMeta().GetDriverModel() != DriverModel {
		t.Fatalf("unexpected collector model %q", collector.GetCollectionMeta().GetDriverModel())
	}
}
