package collectionmgr

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestVariableVoOptionalFields(t *testing.T) {
	unset, err := json.Marshal(toVariableVo(&collection.Variable{Name: "a"}))
	require.NoError(t, err)
	assert.Contains(t, string(unset), `"bits":null`)
	assert.Contains(t, string(unset), `"rate":null`)
	assert.Contains(t, string(unset), `"offset":null`)
	assert.Contains(t, string(unset), `"precision":null`)

	zeroUint8 := uint8(0)
	zeroFloat := float64(0)
	zeroInt := 0
	value, err := json.Marshal(toVariableVo(&collection.Variable{
		Name:      "a",
		Bits:      &zeroUint8,
		Rate:      &zeroFloat,
		Offset:    &zeroFloat,
		Precision: &zeroInt,
	}))
	require.NoError(t, err)
	assert.Contains(t, string(value), `"bits":0`)
	assert.Contains(t, string(value), `"rate":0`)
	assert.Contains(t, string(value), `"offset":0`)
	assert.Contains(t, string(value), `"precision":0`)
}
