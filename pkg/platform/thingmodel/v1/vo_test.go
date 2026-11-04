package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/platform/thingmodel/tsl"
)

func TestThingModelDetailVoJSONFlattensEmbedded(t *testing.T) {
	vo := ThingModelDetailVo{
		ThingModelVo: ThingModelVo{
			ID:            "model-1",
			ThingModelKey: "tm_1",
			Name:          "温控器",
			PropertyCount: 1,
		},
		Properties: []PropertyVo{{
			ID:       "property-1",
			Version:  2,
			Property: tsl.Property{Identifier: "temperature", Name: "温度", RWFlag: tsl.RWRead},
		}},
	}

	data, err := json.Marshal(vo)
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &payload))
	assert.Equal(t, "model-1", payload["id"])
	assert.Equal(t, "tm_1", payload["thingModelKey"])
	assert.Equal(t, "温控器", payload["name"])

	properties, ok := payload["properties"].([]interface{})
	require.True(t, ok)
	require.Len(t, properties, 1)
	property := properties[0].(map[string]interface{})
	assert.Equal(t, "property-1", property["id"])
	assert.Equal(t, "temperature", property["identifier"])
	assert.Equal(t, "温度", property["name"])
	assert.Equal(t, "r", property["rwFlag"])
}
