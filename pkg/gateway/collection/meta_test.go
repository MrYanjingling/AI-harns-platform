package collection

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionMetaJSON(t *testing.T) {
	modTime := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	meta := &CollectionMeta{
		Meta: Meta{
			Name:    "collector-1",
			ID:      "01M7...",
			Version: "1",
			ModTime: modTime,
		},
		DriverType:       "modbus",
		DriverModel:      "modbusTcp",
		CollectStatus:    CollectStatusStopped,
		CollectorCycleMs: 1000,
		Variables: []*Variable{{
			Name:       "temperature",
			Type:       "telemetry",
			DataType:   "float32",
			Address:    "holding:40001",
			AccessMode: AccessModeRead,
			Rate:       1,
			Precision:  2,
		}},
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)
	encoded := string(data)
	assert.Contains(t, encoded, `"name":"collector-1"`)
	assert.Contains(t, encoded, `"id":"01M7..."`)
	assert.Contains(t, encoded, `"version":"1"`)
	assert.Contains(t, encoded, `"driverType":"modbus"`)
	assert.Contains(t, encoded, `"driverModel":"modbusTcp"`)
	assert.Contains(t, encoded, `"collectStatus":"stopped"`)
	assert.Contains(t, encoded, `"collectorCycleMs":1000`)
	assert.Contains(t, encoded, `"variables":[{"name":"temperature"`)
	assert.NotContains(t, encoded, `"eTag"`)
	assert.NotContains(t, encoded, `"deviceCode"`)
}

func TestCollectionMetaAccessors(t *testing.T) {
	now := time.Now().UTC()
	meta := &CollectionMeta{}
	meta.SetName("collector")
	meta.SetID("id-1")
	meta.SetVersion("2")
	meta.SetModTime(now)
	meta.SetDriverType("s7")
	meta.SetDriverModel("s71500")
	meta.SetCollectStatus(CollectStatusUnconnected)
	meta.SetCollectorCycleMs(500)
	meta.IndexCollection()

	assert.Equal(t, "collector", meta.GetName())
	assert.Equal(t, "id-1", meta.GetID())
	assert.Equal(t, "2", meta.GetVersion())
	assert.Equal(t, now, meta.GetModTime())
	assert.Equal(t, "s7", meta.GetDriverType())
	assert.Equal(t, "s71500", meta.GetDriverModel())
	assert.Equal(t, CollectStatusUnconnected, meta.GetCollectStatus())
	assert.Equal(t, uint64(500), meta.GetCollectorCycleMs())
	assert.Same(t, meta, meta.GetCollectionMeta())

	copied := &CollectionMeta{}
	copied.SetCollectionMeta(meta)
	assert.Equal(t, meta, copied)
}
