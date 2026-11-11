package s7

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestParseAddress(t *testing.T) {
	cases := map[string]parsedAddress{
		"DB1.DBW0":   {area: areaDB, blockSize: 1, address: 0},
		"DB1.DBX0.5": {area: areaDB, blockSize: 1, address: 0, bit: 5},
		"I0.0":       {area: areaI, address: 0},
		"Q10.3":      {area: areaQ, address: 10, bit: 3},
		"M100":       {area: areaM, address: 100},
	}
	for value, expected := range cases {
		parsed, err := parseAddress(value)
		require.NoError(t, err, value)
		assert.Equal(t, expected, parsed, value)
	}
}

func TestBuildAndDecodeFrame(t *testing.T) {
	variables := []*collection.Variable{{
		Name: "speed", DataType: "int16", Address: "DB1.DBW0", AccessMode: collection.AccessModeRead,
	}}
	frames, err := buildFrames(variables, 240)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	assert.Equal(t, 6, frames[0].DataLength)

	values, err := decodeFrame(frames[0], []byte{0, 0, 0, 0, 0x00, 0x2A})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, int16(42), values[0].value)
}

func TestBuildFramesRejectsString(t *testing.T) {
	_, err := buildFrames([]*collection.Variable{{
		Name: "text", DataType: "string", Address: "DB1.DBB0", AccessMode: collection.AccessModeRead,
	}}, 240)
	require.Error(t, err)
}
