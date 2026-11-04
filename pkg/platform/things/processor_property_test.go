package things

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/platform/thingmodel/tsl"
	"harns-platform/pkg/timeseries"
)

func strPtr(v string) *string { return &v }

type fakePropertyResolver struct {
	properties map[string]tsl.Property
	err        error
	modelKey   string
}

func (r *fakePropertyResolver) GetPropertiesByKey(ctx context.Context, modelKey string) (map[string]tsl.Property, error) {
	r.modelKey = modelKey
	return r.properties, r.err
}

type fakePropertyTimeseries struct {
	points   []timeseries.PropertyPoint
	logs     []timeseries.PropertyLog
	writeErr error
	logErr   error
}

func (f *fakePropertyTimeseries) WriteProperty(ctx context.Context, modelKey string, points []timeseries.PropertyPoint) error {
	f.points = append(f.points, points...)
	return f.writeErr
}

func (f *fakePropertyTimeseries) WritePropertyLog(ctx context.Context, modelKey string, log timeseries.PropertyLog) error {
	f.logs = append(f.logs, log)
	return f.logErr
}

func propertyDefinitions() map[string]tsl.Property {
	return map[string]tsl.Property{
		"temperature": {
			Identifier: "temperature",
			DataType:   tsl.DataTypeDetail{Type: tsl.DataTypeDouble, Min: strPtr("0"), Max: strPtr("100")},
			Required:   true,
		},
		"status": {
			Identifier: "status",
			DataType:   tsl.DataTypeDetail{Type: tsl.DataTypeText},
		},
	}
}

func decodeProperty(t *testing.T, payload string) *PropertyReport {
	t.Helper()
	report, err := DecodePropertyReport([]byte(payload))
	require.NoError(t, err)
	return report
}

func TestDefaultPropertyProcessorSuccess(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{
		"type":"property",
		"payload":[{"time":1773827321000,"property":{"temperature":"25.5","status":"running","unknown":"ignored"}}]
	}`)
	msg := &DeviceMessage{
		ThingModelKey: "tm_demo",
		ThingModelID:  "model-1",
		ThingID:       "thing-1",
		DeviceKey:     "device-1",
		Payload:       `{"type":"property"}`,
	}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Equal(t, "tm_demo", resolver.modelKey)
	require.Len(t, ts.points, 1)
	assert.Equal(t, map[string]any{"temperature": 25.5, "status": "running"}, ts.points[0].Values)
	assert.Equal(t, time.UnixMilli(1773827321000), ts.points[0].Time)
	require.Len(t, ts.logs, 1)
	assert.True(t, ts.logs[0].Success)
}

func TestDefaultPropertyProcessorRequiredMissing(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1,"property":{"status":"running"}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo", DeviceKey: "device-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Empty(t, ts.points)
	require.Len(t, ts.logs, 1)
	assert.False(t, ts.logs[0].Success)
	assert.Contains(t, ts.logs[0].ErrorMsg, "required property is missing")
}

func TestDefaultPropertyProcessorRequiredNull(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1,"property":{"temperature":null}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Empty(t, ts.points)
	require.Len(t, ts.logs, 1)
	assert.False(t, ts.logs[0].Success)
}

func TestDefaultPropertyProcessorInvalidValue(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1,"property":{"temperature":"abc"}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Empty(t, ts.points)
	require.Len(t, ts.logs, 1)
	assert.False(t, ts.logs[0].Success)
	assert.Contains(t, ts.logs[0].ErrorMsg, "temperature")
}

func TestDefaultPropertyProcessorOptionalNullIsSkipped(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1,"property":{"temperature":20,"status":null}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	require.Len(t, ts.points, 1)
	assert.Equal(t, map[string]any{"temperature": float64(20)}, ts.points[0].Values)
	assert.True(t, ts.logs[0].Success)
}

func TestDefaultPropertyProcessorResolverError(t *testing.T) {
	resolver := &fakePropertyResolver{err: errors.New("db down")}
	ts := &fakePropertyTimeseries{}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1,"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Empty(t, ts.points)
	require.Len(t, ts.logs, 1)
	assert.False(t, ts.logs[0].Success)
	assert.Contains(t, ts.logs[0].ErrorMsg, "db down")
}

func TestDefaultPropertyProcessorWriteError(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{writeErr: errors.New("queue closed")}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1,"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	require.Len(t, ts.logs, 1)
	assert.False(t, ts.logs[0].Success)
	assert.Contains(t, ts.logs[0].ErrorMsg, "queue closed")
}

func TestDefaultPropertyProcessorEmptyPayload(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{"type":"property","payload":[]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Empty(t, ts.points)
	require.Len(t, ts.logs, 1)
	assert.True(t, ts.logs[0].Success)
}

func TestDefaultPropertyProcessorZeroTimeUsesNow(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	processor := NewPropertyProcessor(resolver, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo"}
	before := time.Now().Add(-time.Second)
	after := time.Now().Add(time.Second)

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	require.Len(t, ts.points, 1)
	assert.True(t, ts.points[0].Time.After(before))
	assert.True(t, ts.points[0].Time.Before(after))
}
