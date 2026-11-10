package modbustcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestCreateReqJSONIsFlat(t *testing.T) {
	req := &CreateReq{
		Name:             "modbus-tcp-1",
		CollectorCycleMs: 1000,
		Location:         "127.0.0.1:502",
		BackupLocation:   "127.0.0.2:502",
		MemoryLayout:     "ABCD",
		Variables: []*collection.Variable{{
			Name:       "temperature",
			Type:       "telemetry",
			DataType:   "float32",
			Address:    "holding:40001",
			AccessMode: collection.AccessModeRead,
			Rate:       1,
			Precision:  2,
		}},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	encoded := string(data)
	assert.Contains(t, encoded, `"location":"127.0.0.1:502"`)
	assert.Contains(t, encoded, `"backupLocation":"127.0.0.2:502"`)
	assert.Contains(t, encoded, `"variables"`)
	assert.NotContains(t, encoded, `"address":{"`)
	assert.NotContains(t, encoded, `"config"`)
	assert.NotContains(t, encoded, `"variableInterval"`)
}

func TestNewCollection(t *testing.T) {
	req := &CreateReq{
		Name:             "modbus-tcp-1",
		CollectorCycleMs: 500,
		Location:         "127.0.0.1:502",
		BackupLocation:   "127.0.0.2:502",
		MemoryLayout:     "ABCD",
		Variables: []*collection.Variable{{
			Name:       "temperature",
			DataType:   "float32",
			Address:    "holding:40001",
			AccessMode: collection.AccessModeRead,
		}},
	}

	created, err := NewCollection(req)
	require.NoError(t, err)
	assert.Equal(t, DriverType, created.DriverType)
	assert.Equal(t, DriverModel, created.DriverModel)
	assert.Equal(t, collection.CollectStatusStopped, created.CollectStatus)
	assert.Equal(t, uint64(500), created.CollectorCycleMs)
	assert.Equal(t, "127.0.0.1:502", created.Location)
	require.Len(t, created.Variables, 1)
	assert.Equal(t, float64(1), created.Variables[0].Rate)
}

func TestConfigJSONRoundTrip(t *testing.T) {
	original := &ModbusTcpCollection{
		Location:        "127.0.0.1:502",
		BackupLocation:  "127.0.0.2:502",
		MemoryLayout:    "ABCD",
		PositionAddress: 10,
	}
	data, err := original.ConfigJSON()
	require.NoError(t, err)
	assert.NotContains(t, string(data), "variables")

	var restored ModbusTcpCollection
	require.NoError(t, restored.ApplyConfig(data))
	assert.Equal(t, original.Location, restored.Location)
	assert.Equal(t, original.BackupLocation, restored.BackupLocation)
	assert.Equal(t, original.MemoryLayout, restored.MemoryLayout)
	assert.Equal(t, original.PositionAddress, restored.PositionAddress)
}
