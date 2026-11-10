package modbustcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateHandlerDecodeAndBuild(t *testing.T) {
	handler := &CreateHandler{}
	body := []byte(`{
		"name":"modbus-tcp-1",
		"collectorCycleMs":1000,
		"location":"127.0.0.1:502",
		"backupLocation":"127.0.0.2:502",
		"memoryLayout":"ABCD",
		"variables":[{"name":"temperature","dataType":"float32","address":"holding:40001","accessMode":"r"}]
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
}

func TestCreateReqValidate(t *testing.T) {
	req := &CreateReq{
		Name:             "",
		CollectorCycleMs: 1000,
	}
	require.Error(t, req.Validate())
}
