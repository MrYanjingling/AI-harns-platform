package timeseries

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableNames(t *testing.T) {
	assert.Equal(t, "device_property_tm_1", PropertyTable("tm_1"))
	assert.Equal(t, "device_property_log_tm_1", PropertyLogTable("tm_1"))
	assert.Equal(t, "device_event_tm_1", EventTable("tm_1"))
	assert.Equal(t, "device_event_log_tm_1", EventLogTable("tm_1"))
	assert.Equal(t, "device_command_record_tm_1", CommandRecordTable("tm_1"))
	assert.Equal(t, "gateway_property_log", GatewayPropertyLogMeasurement)
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
	assert.Contains(t, cfg.QueryAggregates, "count")
	assert.Contains(t, cfg.QueryAggregates, "sum")
}

type capturePointWriter struct {
	points []*Point
}

func (w *capturePointWriter) WritePoints(ctx context.Context, points []*Point) error {
	w.points = append(w.points, points...)
	return nil
}

func TestWriteGatewayPropertyLog(t *testing.T) {
	writer := &capturePointWriter{}
	store := &Store{log: newBatcher(writer, 1, time.Millisecond, time.Second, 10)}
	store.log.start()
	service := &Service{store: store}
	t.Cleanup(func() { _ = store.Close() })

	err := service.WriteGatewayPropertyLog(context.Background(), GatewayPropertyLog{
		GatewayType: "modbusTcp",
		GatewayID:   "gw-1",
		Time:        time.UnixMilli(1000),
		Success:     false,
		Payload:     `{"type":"property"}`,
		ErrorMsg:    "boom",
	})
	require.NoError(t, err)
	require.NoError(t, store.Close())
	require.Len(t, writer.points, 1)
	assert.Equal(t, GatewayPropertyLogMeasurement, writer.points[0].Measurement)
	assert.Equal(t, "modbusTcp", writer.points[0].Tags[TagGatewayType])
	assert.Equal(t, "gw-1", writer.points[0].Tags[TagGatewayID])
	assert.Equal(t, false, writer.points[0].Fields[FieldSuccess])
	assert.Equal(t, "boom", writer.points[0].Fields[FieldErrorMsg])
}
