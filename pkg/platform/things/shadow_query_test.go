package things

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/platform/thingmodel/tsl"
)

func TestBuildShadowProperties(t *testing.T) {
	properties := []tsl.Property{
		{Identifier: "temperature", Name: "温度", Required: true, Sort: 0},
		{Identifier: "mode", Name: "模式", Sort: 1},
		{Identifier: "running", Name: "运行", Sort: 2},
	}
	reported := map[string]json.RawMessage{
		"temperature": json.RawMessage(`26.0`),
		"unknown":     json.RawMessage(`true`),
	}
	desired := map[string]json.RawMessage{
		"mode": json.RawMessage(`2`),
	}

	got := buildShadowProperties(properties, reported, desired)
	require.Len(t, got, 3)
	assert.Equal(t, "temperature", got[0].Identifier)
	assert.JSONEq(t, `26.0`, string(got[0].Value))
	assert.Nil(t, got[0].DesiredValue)

	assert.Equal(t, "mode", got[1].Identifier)
	assert.Nil(t, got[1].Value)
	assert.JSONEq(t, `2`, string(got[1].DesiredValue))

	assert.Equal(t, "running", got[2].Identifier)
	assert.Nil(t, got[2].Value)
	assert.Nil(t, got[2].DesiredValue)
}
