package things

import (
	"context"
	"errors"
	"testing"
	"time"

	"harns-platform/pkg/apis/response"

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
	points      []timeseries.PropertyPoint
	logs        []timeseries.PropertyLog
	gatewayLogs []timeseries.GatewayPropertyLog
	debugValues []bool
	writeErr    error
	logErr      error
	order       *[]string
}

type fakeThingSnapshotResolver struct {
	snapshots map[string]*ThingSnapshot
	err       error
}

func (f *fakeThingSnapshotResolver) GetThingSnapshot(ctx context.Context, thingID string) (*ThingSnapshot, error) {
	if f.err != nil {
		return nil, f.err
	}
	snapshot, ok := f.snapshots[thingID]
	if !ok {
		return nil, response.ErrThingNotFound(thingID)
	}
	return snapshot, nil
}

func (f *fakePropertyTimeseries) WriteProperty(ctx context.Context, modelKey string, points []timeseries.PropertyPoint, debug bool) error {
	if f.order != nil {
		*f.order = append(*f.order, "timeseries")
	}
	f.points = append(f.points, points...)
	f.debugValues = append(f.debugValues, debug)
	return f.writeErr
}

func (f *fakePropertyTimeseries) WritePropertyLog(ctx context.Context, modelKey string, log timeseries.PropertyLog) error {
	f.logs = append(f.logs, log)
	return f.logErr
}

func (f *fakePropertyTimeseries) WriteGatewayPropertyLog(ctx context.Context, log timeseries.GatewayPropertyLog) error {
	f.gatewayLogs = append(f.gatewayLogs, log)
	return f.logErr
}

type fakePropertyShadow struct {
	updates  []ShadowUpdate
	thingIDs []string
	thingID  string
	calls    int
	err      error
	order    *[]string
}

func (f *fakePropertyShadow) UpdateReported(ctx context.Context, thingID string, updates []ShadowUpdate) error {
	f.calls++
	f.thingID = thingID
	f.thingIDs = append(f.thingIDs, thingID)
	f.updates = append(f.updates, updates...)
	if f.order != nil {
		*f.order = append(*f.order, "shadow")
	}
	return f.err
}

type fakePropertyTriggerSink struct {
	triggers []PropertyTrigger
	order    *[]string
}

func (f *fakePropertyTriggerSink) Dispatch(ctx context.Context, trigger PropertyTrigger) {
	f.triggers = append(f.triggers, trigger)
	if f.order != nil {
		*f.order = append(*f.order, "trigger")
	}
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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

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
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo"}
	before := time.Now().Add(-time.Second)
	after := time.Now().Add(time.Second)

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	require.Len(t, ts.points, 1)
	assert.True(t, ts.points[0].Time.After(before))
	assert.True(t, ts.points[0].Time.Before(after))
}

func TestDefaultPropertyProcessorUpdatesShadowBeforeTimeseries(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	order := make([]string, 0, 3)
	ts := &fakePropertyTimeseries{order: &order}
	shadow := &fakePropertyShadow{order: &order}
	trigger := &fakePropertyTriggerSink{order: &order}
	processor := NewPropertyProcessor(resolver, ts, shadow, trigger, nil, nil, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1773827321000,"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{
		ThingModelKey: "tm_demo",
		ThingModelID:  "model-1",
		ThingID:       "thing-1",
		DeviceKey:     "device-1",
		DeviceName:    "demo",
	}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Equal(t, "thing-1", shadow.thingID)
	require.Len(t, shadow.updates, 1)
	assert.Equal(t, map[string]any{"temperature": float64(20)}, shadow.updates[0].Values)
	require.Len(t, trigger.triggers, 1)
	assert.Equal(t, int64(1773827321000), trigger.triggers[0].Time)
	assert.Equal(t, "thing-1", trigger.triggers[0].DeviceID)
	assert.Equal(t, "device-1", trigger.triggers[0].DeviceKey)
	assert.Equal(t, "demo", trigger.triggers[0].DeviceName)
	assert.Equal(t, "tm_demo", trigger.triggers[0].ModelKey)
	assert.Equal(t, map[string]any{"temperature": float64(20)}, trigger.triggers[0].Values)
	assert.Equal(t, []string{"shadow", "trigger", "timeseries"}, order)
}

func TestDefaultPropertyProcessorSkipsShadowOnValidationFailure(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	shadow := &fakePropertyShadow{}
	processor := NewPropertyProcessor(resolver, ts, shadow, nil, nil, nil, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1,"property":{"temperature":"abc"}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo", ThingID: "thing-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Zero(t, shadow.calls)
	assert.Empty(t, ts.points)
}

func TestDefaultPropertyProcessorShadowErrorContinues(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	shadow := &fakePropertyShadow{err: errors.New("redis down")}
	processor := NewPropertyProcessor(resolver, ts, shadow, nil, nil, nil, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":1,"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo", ThingID: "thing-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Equal(t, 1, shadow.calls)
	require.Len(t, ts.points, 1)
	require.Len(t, ts.logs, 1)
	assert.True(t, ts.logs[0].Success)
}

func TestDefaultPropertyProcessorDispatchesOneTriggerPerItem(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	trigger := &fakePropertyTriggerSink{}
	processor := NewPropertyProcessor(resolver, ts, nil, trigger, nil, nil, ts)

	report := decodeProperty(t, `{
		"type":"property",
		"payload":[
			{"time":100,"property":{"temperature":20,"status":"on"}},
			{"time":200,"property":{"temperature":21}}
		]
	}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo", ThingID: "thing-1", DeviceKey: "device-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	require.Len(t, trigger.triggers, 2)
	assert.Equal(t, int64(100), trigger.triggers[0].Time)
	assert.Equal(t, map[string]any{"temperature": float64(20), "status": "on"}, trigger.triggers[0].Values)
	assert.Equal(t, int64(200), trigger.triggers[1].Time)
	assert.Equal(t, map[string]any{"temperature": float64(21)}, trigger.triggers[1].Values)
}

func TestDefaultPropertyProcessorDispatchesWhenShadowFails(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	shadow := &fakePropertyShadow{err: errors.New("redis down")}
	trigger := &fakePropertyTriggerSink{}
	processor := NewPropertyProcessor(resolver, ts, shadow, trigger, nil, nil, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":100,"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo", ThingID: "thing-1", DeviceKey: "device-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Len(t, trigger.triggers, 1)
	assert.Len(t, ts.points, 1)
}

func TestDefaultPropertyProcessorDoesNotDispatchOnValidationFailure(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	trigger := &fakePropertyTriggerSink{}
	processor := NewPropertyProcessor(resolver, ts, nil, trigger, nil, nil, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":100,"property":{"temperature":"bad"}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo", ThingID: "thing-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Empty(t, trigger.triggers)
}

func TestDefaultPropertyProcessorDebugSkipsTriggersAndWritesDebugSeries(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	trigger := &fakePropertyTriggerSink{}
	snapshots := &fakeThingSnapshotResolver{snapshots: map[string]*ThingSnapshot{
		"thing-1": {ID: "thing-1", Debug: 1},
	}}
	processor := NewPropertyProcessor(resolver, ts, nil, trigger, nil, snapshots, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":100,"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo", ThingID: "thing-1", DeviceKey: "device-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Empty(t, trigger.triggers)
	require.Len(t, ts.points, 1)
	assert.Equal(t, []bool{true}, ts.debugValues)
}

func TestDefaultPropertyProcessorDropsWhenSnapshotMissing(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	shadow := &fakePropertyShadow{}
	trigger := &fakePropertyTriggerSink{}
	snapshots := &fakeThingSnapshotResolver{snapshots: map[string]*ThingSnapshot{}}
	processor := NewPropertyProcessor(resolver, ts, shadow, trigger, nil, snapshots, ts)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":100,"property":{"temperature":20}}]}`)
	msg := &DeviceMessage{ThingModelKey: "tm_demo", ThingID: "deleted-thing", DeviceKey: "device-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Zero(t, shadow.calls)
	assert.Empty(t, trigger.triggers)
	assert.Empty(t, ts.points)
	assert.Empty(t, ts.logs)
}
