package modbustcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

func TestCreateHandlerDecodeAndBuild(t *testing.T) {
	handler := &CreateHandler{}
	body := []byte(`{
		"name":"modbus-tcp-1",
		"collectorCycleMs":1000,
		"protocolConfig":{"location":"127.0.0.1:502","backupLocation":"127.0.0.2:502","memoryLayout":"ABCD"},
		"variables":[{"name":"temperature","dataType":"float32","address":"1!40001","accessMode":"r"}]
	}`)

	request, err := handler.DecodeCreate(body)
	require.NoError(t, err)
	req, ok := request.(*CreateReq)
	require.True(t, ok)
	require.NoError(t, req.Validate())

	created, config, err := handler.BuildCreate(req)
	require.NoError(t, err)
	item, ok := created.(*ModbusTcpCollection)
	require.True(t, ok)
	assert.Equal(t, DriverType, item.DriverType)
	assert.Equal(t, DriverModel, item.DriverModel)
	assert.Equal(t, "127.0.0.1:502", item.Location)
	assert.Equal(t, "127.0.0.2:502", item.BackupLocation)

	var configMap map[string]any
	require.NoError(t, json.Unmarshal(config, &configMap))
	assert.Equal(t, "127.0.0.1:502", configMap["location"])
	assert.Equal(t, float64(0), configMap["positionAddress"])
	assert.Equal(t, float64(1), configMap["concurrency"])
}

func TestCreateReqValidate(t *testing.T) {
	req := &CreateReq{
		Name:             "",
		CollectorCycleMs: 1000,
	}
	require.Error(t, req.Validate())
}

func TestCreateReqValidatePositionAddress(t *testing.T) {
	req := &CreateReq{
		Name:             "modbus-tcp-position",
		CollectorCycleMs: 1000,
		ProtocolConfig: ProtocolConfig{
			Location:        "127.0.0.1:502",
			MemoryLayout:    "ABCD",
			PositionAddress: 1,
		},
		Variables: []*collection.Variable{{
			Name:       "temperature",
			DataType:   "float32",
			Address:    "1!0",
			AccessMode: collection.AccessModeRead,
		}},
	}
	err := req.Validate()
	require.Error(t, err)
	var validationErr *validation.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "variables[0].address", validationErr.Issues[0].Field)
}
