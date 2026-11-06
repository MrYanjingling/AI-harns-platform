package things

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/timeseries"
)

type fakeGatewayResolver struct {
	bindings map[string][]GatewayPropertyBinding
	err      error
	calls    int
}

func (r *fakeGatewayResolver) Resolve(ctx context.Context, gatewayType, gatewayID, collectionID string) ([]GatewayPropertyBinding, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return r.bindings[collectionID], nil
}

type fakeGatewayLogWriter struct {
	logs       []timeseries.GatewayPropertyLog
	deviceLogs []timeseries.PropertyLog
}

func (w *fakeGatewayLogWriter) WritePropertyLog(ctx context.Context, modelKey string, log timeseries.PropertyLog) error {
	w.deviceLogs = append(w.deviceLogs, log)
	return nil
}

func (w *fakeGatewayLogWriter) WriteGatewayPropertyLog(ctx context.Context, log timeseries.GatewayPropertyLog) error {
	w.logs = append(w.logs, log)
	return nil
}

func gatewayBindings() map[string][]GatewayPropertyBinding {
	return map[string][]GatewayPropertyBinding{
		"c1": {
			{
				ThingID:         "thing-1",
				ThingModelID:    "model-1",
				ThingModelKey:   "tm_demo",
				DeviceKey:       "device-1",
				DeviceName:      "meter-1",
				PropertyMapping: map[string]string{"t1": "temperature", "s": "status"},
			},
			{
				ThingID:         "thing-2",
				ThingModelID:    "model-1",
				ThingModelKey:   "tm_demo",
				DeviceKey:       "device-2",
				DeviceName:      "meter-2",
				PropertyMapping: map[string]string{"t2": "temperature"},
			},
		},
	}
}

func TestDefaultPropertyProcessorGatewaySuccess(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	shadow := &fakePropertyShadow{}
	trigger := &fakePropertyTriggerSink{}
	gatewayResolver := &fakeGatewayResolver{bindings: gatewayBindings()}
	gatewayLogs := &fakeGatewayLogWriter{}
	processor := NewPropertyProcessor(resolver, ts, shadow, trigger, gatewayResolver, nil, gatewayLogs)

	report := decodeProperty(t, `{
		"type":"property",
		"payload":[{"collectionId":"c1","time":1773827321000,"property":{"t1":20,"t2":21,"s":"on"}}]
	}`)
	msg := &DeviceMessage{
		GatewayType: "modbusTcp",
		GatewayID:   "gw-1",
		DeviceKey:   "",
		Payload:     `{"type":"property"}`,
	}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Equal(t, 1, gatewayResolver.calls)
	require.Len(t, ts.points, 2)
	assert.Equal(t, map[string]any{"temperature": float64(20), "status": "on"}, ts.points[0].Values)
	assert.Equal(t, map[string]any{"temperature": float64(21)}, ts.points[1].Values)
	assert.ElementsMatch(t, []string{"thing-1", "thing-2"}, shadow.thingIDs)
	require.Len(t, trigger.triggers, 2)
	assert.Equal(t, "thing-1", trigger.triggers[0].DeviceID)
	assert.Equal(t, "tm_demo", trigger.triggers[0].ModelKey)
	assert.Equal(t, int64(1773827321000), trigger.triggers[0].Time)
	require.Len(t, gatewayLogs.logs, 1)
	assert.True(t, gatewayLogs.logs[0].Success)
	assert.Equal(t, "gw-1", gatewayLogs.logs[0].GatewayID)
	assert.Empty(t, gatewayLogs.logs[0].ErrorMsg)
}

func TestDefaultPropertyProcessorGatewayPartialFailure(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	shadow := &fakePropertyShadow{}
	trigger := &fakePropertyTriggerSink{}
	gatewayResolver := &fakeGatewayResolver{bindings: gatewayBindings()}
	gatewayLogs := &fakeGatewayLogWriter{}
	processor := NewPropertyProcessor(resolver, ts, shadow, trigger, gatewayResolver, nil, gatewayLogs)

	// t2 is missing, so the second binding fails required validation.
	report := decodeProperty(t, `{
		"type":"property",
		"payload":[{"collectionId":"c1","time":100,"property":{"t1":20,"s":"on"}}]
	}`)
	msg := &DeviceMessage{GatewayType: "modbusTcp", GatewayID: "gw-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	require.Len(t, ts.points, 1)
	assert.Equal(t, "thing-1", ts.points[0].DeviceID)
	assert.Equal(t, []string{"thing-1"}, shadow.thingIDs)
	require.Len(t, trigger.triggers, 1)
	require.Len(t, gatewayLogs.logs, 1)
	assert.False(t, gatewayLogs.logs[0].Success)
	assert.Contains(t, gatewayLogs.logs[0].ErrorMsg, "required property is missing: temperature")
}

func TestDefaultPropertyProcessorGatewayResolverError(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	gatewayResolver := &fakeGatewayResolver{err: errors.New("mapping not found")}
	gatewayLogs := &fakeGatewayLogWriter{}
	processor := NewPropertyProcessor(resolver, ts, nil, nil, gatewayResolver, nil, gatewayLogs)

	report := decodeProperty(t, `{"type":"property","payload":[{"collectionId":"c1","time":100,"property":{"t1":20}}]}`)
	msg := &DeviceMessage{GatewayID: "gw-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Empty(t, ts.points)
	require.Len(t, gatewayLogs.logs, 1)
	assert.False(t, gatewayLogs.logs[0].Success)
	assert.Contains(t, gatewayLogs.logs[0].ErrorMsg, "mapping not found")
}

func TestDefaultPropertyProcessorGatewayMissingCollectionID(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	gatewayResolver := &fakeGatewayResolver{bindings: gatewayBindings()}
	gatewayLogs := &fakeGatewayLogWriter{}
	processor := NewPropertyProcessor(resolver, ts, nil, nil, gatewayResolver, nil, gatewayLogs)

	report := decodeProperty(t, `{"type":"property","payload":[{"time":100,"property":{"t1":20}}]}`)
	msg := &DeviceMessage{GatewayID: "gw-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	assert.Zero(t, gatewayResolver.calls)
	require.Len(t, gatewayLogs.logs, 1)
	assert.False(t, gatewayLogs.logs[0].Success)
	assert.Contains(t, gatewayLogs.logs[0].ErrorMsg, "collectionId is required")
}

func TestDefaultPropertyProcessorGatewayResolverNotConfigured(t *testing.T) {
	resolver := &fakePropertyResolver{properties: propertyDefinitions()}
	ts := &fakePropertyTimeseries{}
	gatewayLogs := &fakeGatewayLogWriter{}
	processor := NewPropertyProcessor(resolver, ts, nil, nil, nil, nil, gatewayLogs)

	report := decodeProperty(t, `{"type":"property","payload":[{"collectionId":"c1","time":100,"property":{"t1":20}}]}`)
	msg := &DeviceMessage{GatewayID: "gw-1"}

	require.NoError(t, processor.ProcessProperty(context.Background(), msg, report))
	require.Len(t, gatewayLogs.logs, 1)
	assert.False(t, gatewayLogs.logs[0].Success)
	assert.Contains(t, gatewayLogs.logs[0].ErrorMsg, "resolver is not configured")
}
