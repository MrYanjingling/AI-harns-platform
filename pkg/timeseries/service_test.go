package timeseries

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableNames(t *testing.T) {
	assert.Equal(t, "device_property_tm_1", PropertyTable("tm_1"))
	assert.Equal(t, "device_property_log_tm_1", PropertyLogTable("tm_1"))
	assert.Equal(t, "device_event_tm_1", EventTable("tm_1"))
	assert.Equal(t, "device_event_log_tm_1", EventLogTable("tm_1"))
	assert.Equal(t, "device_command_record_tm_1", CommandRecordTable("tm_1"))
}

func TestNormalizeValue(t *testing.T) {
	assert.Equal(t, int64(1), normalizeValue(int(1)))
	assert.Equal(t, int64(2), normalizeValue(int32(2)))
	assert.Equal(t, int64(3), normalizeValue(int64(3)))
	assert.Equal(t, uint64(4), normalizeValue(uint32(4)))
	assert.Equal(t, float64(5.5), normalizeValue(float32(5.5)))
	assert.Equal(t, "s", normalizeValue("s"))
	assert.Equal(t, true, normalizeValue(true))
	assert.Nil(t, normalizeValue(nil))

	raw := json.RawMessage(`9007199254740993`)
	assert.Equal(t, int64(9007199254740993), normalizeValue(raw))

	rawObject := json.RawMessage(`{"a":1}`)
	assert.JSONEq(t, `{"a":1}`, normalizeValue(rawObject).(string))

	assert.JSONEq(t, `{"a":1}`, normalizeValue(map[string]any{"a": 1}).(string))
	assert.JSONEq(t, `[1,2]`, normalizeValue([]any{1, 2}).(string))
}

func TestNormalizeFieldsSkipsNil(t *testing.T) {
	fields := normalizeFields(map[string]any{"a": 1, "b": nil, "c": int32(2)})
	assert.Equal(t, map[string]any{"a": int64(1), "c": int64(2)}, fields)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "harns_timeseries", cfg.Database)
	assert.Equal(t, "harns_timeseries_log", cfg.LogDatabase)
	assert.Equal(t, 10000, cfg.BatchSize)
	assert.Equal(t, 20000, cfg.QueueSize)
	assert.Equal(t, 3000, cfg.FlushIntervalMs)
}
