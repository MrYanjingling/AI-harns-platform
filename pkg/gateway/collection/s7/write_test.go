package s7

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestBuildWriteRequest(t *testing.T) {
	request, transportSize, err := buildWriteRequest(&collection.Variable{
		Name: "speed", DataType: "int16", Address: "DB1.DBW0", AccessMode: collection.AccessModeReadWrite,
	}, int16(42))
	require.NoError(t, err)
	assert.Equal(t, uint8(2), transportSize)
	assert.NotEmpty(t, request)
}
