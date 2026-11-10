package collectionmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection/modbustcp"
)

func TestCreateRegistry(t *testing.T) {
	handler := &modbustcp.CreateHandler{}
	registry, err := newCreateRegistry(handler)
	require.NoError(t, err)

	resolved, ok := registry.resolve(modbustcp.DriverType, modbustcp.DriverModel)
	require.True(t, ok)
	assert.Same(t, handler, resolved)

	_, ok = registry.resolve(modbustcp.DriverType, "unknown")
	assert.False(t, ok)

	_, err = newCreateRegistry(handler, handler)
	require.Error(t, err)
}
