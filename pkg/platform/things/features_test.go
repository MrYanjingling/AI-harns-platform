package things

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeFeatureValues(t *testing.T) {
	merged := mergeFeatureValues(
		map[string]struct{}{"a": {}, "b": {}},
		map[string]json.RawMessage{
			"a": json.RawMessage(`380`),
			"c": json.RawMessage(`"legacy"`),
		},
	)

	assert.JSONEq(t, `380`, string(merged["a"]))
	assert.Nil(t, merged["b"])
	assert.JSONEq(t, `"legacy"`, string(merged["c"]))
}
