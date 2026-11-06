package things

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/platform/thingmodel/tsl"
	"harns-platform/pkg/platform/things/v1"
)

func TestBuildUploadFormat(t *testing.T) {
	properties := []tsl.Property{
		{Identifier: "temperature", DataType: tsl.DataTypeDetail{Type: tsl.DataTypeDouble}},
		{Identifier: "humidity", DataType: tsl.DataTypeDetail{Type: tsl.DataTypeInt}},
		{Identifier: "enabled", DataType: tsl.DataTypeDetail{Type: tsl.DataTypeBool}},
	}

	format, err := buildUploadFormat(properties, 1772343234400)
	require.NoError(t, err)
	assert.NotContains(t, format, "collectionId")

	var document struct {
		Type    string `json:"type"`
		Payload []struct {
			Time     int64          `json:"time"`
			Property map[string]any `json:"property"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal([]byte(format), &document))
	assert.Equal(t, "property", document.Type)
	require.Len(t, document.Payload, 1)
	assert.Equal(t, int64(1772343234400), document.Payload[0].Time)
	assert.Equal(t, float64(0), document.Payload[0].Property["temperature"])
	assert.Equal(t, float64(0), document.Payload[0].Property["humidity"])
	assert.Equal(t, false, document.Payload[0].Property["enabled"])
}

func TestBuildUploadFormatWithoutProperties(t *testing.T) {
	format, err := buildUploadFormat(nil, 123)
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"property","payload":[{"time":123,"property":{}}]}`, format)
}

func TestUploadSampleValue(t *testing.T) {
	assert.Equal(t, int64(0), uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeInt}))
	assert.Equal(t, int64(0), uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeLong}))
	assert.Equal(t, float64(0), uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeFloat}))
	assert.Equal(t, float64(0), uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeDouble}))
	assert.Equal(t, false, uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeBool}))
	assert.Equal(t, 2, uploadSampleValue(tsl.DataTypeDetail{
		Type:     tsl.DataTypeEnum,
		EnumSpec: &tsl.EnumSpec{Values: []tsl.EnumValue{{Value: 2}}},
	}))
	assert.Equal(t, "", uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeText}))
	assert.Equal(t, "", uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeTime}))
	assert.Equal(t, []any{}, uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeArray}))
	assert.Equal(t, map[string]any{}, uploadSampleValue(tsl.DataTypeDetail{Type: tsl.DataTypeStruct}))
}

func TestToThingBasicVo(t *testing.T) {
	item := &ThingBasic{
		Thing: Thing{
			Meta: apis.Meta{
				ID:          "thing-1",
				CreatedUser: "user-1",
				UpdatedUser: "user-2",
			},
			ThingModelID: "model-1",
		},
		ThingModelKey:       "tm_demo",
		ThingModelName:      "Demo",
		CreatedUserName:     "alice",
		CreatedUserNickName: "Alice",
		UpdatedUserName:     "bob",
		UpdatedUserNickName: "Bob",
	}

	vo := toThingBasicVo(item)
	assert.Equal(t, "tm_demo", vo.ThingModelKey)
	assert.Equal(t, "Demo", vo.ThingModelName)
	require.NotNil(t, vo.CreatedBy)
	assert.Equal(t, v1.UserBriefVo{ID: "user-1", UserName: "alice", NickName: "Alice"}, *vo.CreatedBy)
	require.NotNil(t, vo.UpdatedBy)
	assert.Equal(t, "bob", vo.UpdatedBy.UserName)
}
