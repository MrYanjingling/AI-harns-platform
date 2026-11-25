package drivermgr

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func normalizedSchema(t *testing.T, raw, model string) *Schema {
	t.Helper()
	text, err := normalizeBuiltinSchema(raw, model)
	require.NoError(t, err)
	schema, err := ParseSchema(json.RawMessage(text))
	require.NoError(t, err)
	return schema
}

func TestNormalizeEndpointSchema(t *testing.T) {
	schema := normalizedSchema(t, s71500Schema, "s7-1500")
	require.NotNil(t, schema.Endpoint)
	assert.Nil(t, findField(schema.Fields, "port"))
	assert.Nil(t, findField(schema.Fields, "host"))
	location := findField(schema.Fields, "location")
	require.NotNil(t, location)
	assert.Contains(t, location.PlaceholderZh, ":102")
	assert.Equal(t, "location", schema.Endpoint.Source)

	opc := normalizedSchema(t, opcUaSchema, "opcUa")
	location = findField(opc.Fields, "location")
	require.NotNil(t, location)
	assert.True(t, strings.HasPrefix(location.PlaceholderZh, "opc.tcp://"))
	assert.Equal(t, 4840, opc.Endpoint.DefaultPort)

	link := normalizedSchema(t, hj212Schema, "HJ212")
	require.NotNil(t, link.Endpoint)
	assert.Equal(t, "link", link.Endpoint.Link)
	location = findField(link.Fields, "location")
	require.NotNil(t, location)
	require.Len(t, location.Variants, 2)
	assert.Contains(t, location.Variants[0].PlaceholderZh, "ttyUSB0")
	assert.Contains(t, location.Variants[1].PlaceholderZh, ":9000")
}

func TestSplitAndMergeEndpoint(t *testing.T) {
	schema := normalizedSchema(t, finsSchema, "fins")
	body, err := schema.SplitEndpointBody([]byte(`{"driverType":"omron","driverModel":"fins","protocolConfig":{"location":"127.0.0.1:9600","transport":"udp"}}`))
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	protocol, ok := payload["protocolConfig"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "127.0.0.1", protocol["location"])
	assert.Equal(t, "127.0.0.1", protocol["host"])
	assert.Equal(t, float64(9600), protocol["port"])

	merged, err := schema.MergeEndpointConfig([]byte(`{"location":"127.0.0.1","port":9600,"transport":"udp"}`))
	require.NoError(t, err)
	var config map[string]any
	require.NoError(t, json.Unmarshal(merged, &config))
	assert.Equal(t, "127.0.0.1:9600", config["location"])
	assert.NotContains(t, config, "port")

	serial := normalizedSchema(t, dlt645Schema, "dlt645_2007")
	serialBody, err := serial.SplitEndpointBody([]byte(`{"protocolConfig":{"link":0,"location":"/dev/ttyUSB0"}}`))
	require.NoError(t, err)
	serialMerged, err := serial.MergeEndpointConfig([]byte(`{"link":0,"location":"/dev/ttyUSB0","port":2010}`))
	require.NoError(t, err)
	var serialConfig map[string]any
	require.NoError(t, json.Unmarshal(serialMerged, &serialConfig))
	assert.Equal(t, "/dev/ttyUSB0", serialConfig["location"])
	assert.NotContains(t, serialConfig, "port")
	assert.Contains(t, string(serialBody), "/dev/ttyUSB0")
}

func TestStoreBuiltinSchemaUsesMergedEndpoint(t *testing.T) {
	service, _ := newTestService(t)
	items, err := service.List(t.Context(), "")
	require.NoError(t, err)
	found := false
	for _, item := range items {
		if item.Model != "s7-1500" {
			continue
		}
		found = true
		schema, err := ParseSchema(item.Schema)
		require.NoError(t, err)
		assert.Nil(t, findField(schema.Fields, "port"))
		location := findField(schema.Fields, "location")
		require.NotNil(t, location)
		assert.Contains(t, location.PlaceholderZh, ":102")
		assert.Equal(t, "location", schema.Endpoint.Source)
	}
	assert.True(t, found)
}
