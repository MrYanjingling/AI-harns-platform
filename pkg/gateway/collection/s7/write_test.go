package s7

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestBuildWriteRequest(t *testing.T) {
	request, err := buildWriteRequest(&collection.Variable{
		Name: "speed", DataType: "int16", Address: "DB1.DBW0", AccessMode: collection.AccessModeReadWrite,
	}, int16(42))
	require.NoError(t, err)
	assert.NotEmpty(t, request)
}

func TestBuildWriteRequestBitAndBoolTransport(t *testing.T) {
	cases := []struct {
		name      string
		variable  *collection.Variable
		value     any
		dataBytes []byte
	}{
		{
			name:      "bit true",
			variable:  &collection.Variable{Name: "b", DataType: "BIT", Address: "DB1.DBX0.2"},
			value:     1,
			dataBytes: []byte{0x00, 0x03, 0x00, 0x01, 0x01},
		},
		{
			name:      "bit false",
			variable:  &collection.Variable{Name: "b", DataType: "BIT", Address: "DB1.DBX0.2"},
			value:     0,
			dataBytes: []byte{0x00, 0x03, 0x00, 0x01, 0x00},
		},
		{
			name:      "bool true",
			variable:  &collection.Variable{Name: "b", DataType: "BOOL", Address: "DB1.DBX0.2"},
			value:     true,
			dataBytes: []byte{0x00, 0x03, 0x00, 0x01, 0x01},
		},
		{
			name:      "bool false",
			variable:  &collection.Variable{Name: "b", DataType: "BOOL", Address: "DB1.DBX0.2"},
			value:     false,
			dataBytes: []byte{0x00, 0x03, 0x00, 0x01, 0x00},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := buildWriteRequest(tc.variable, tc.value)
			require.NoError(t, err)
			require.Greater(t, len(request), 23)
			assert.Equal(t, uint8(0x01), request[22])
			assert.Equal(t, tc.dataBytes, request[len(request)-len(tc.dataBytes):])
		})
	}
}

func TestBuildWriteRequestNumericParameterTransport(t *testing.T) {
	request, err := buildWriteRequest(&collection.Variable{
		Name: "speed", DataType: "int16", Address: "DB1.DBW0",
	}, int16(42))
	require.NoError(t, err)
	require.Greater(t, len(request), 23)
	assert.Equal(t, uint8(0x02), request[22])
	assert.Equal(t, []byte{0x00, 0x04, 0x00, 0x10, 0x00, 0x2A}, request[len(request)-6:])
}
