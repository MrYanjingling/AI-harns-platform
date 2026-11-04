package things

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestShadow() *thingShadow {
	return &thingShadow{
		Reported: make(map[string]json.RawMessage),
		Desired:  make(map[string]json.RawMessage),
	}
}

func TestApplyShadowUpdatesMergesReports(t *testing.T) {
	shadow := newTestShadow()
	changed, err := applyShadowUpdates(shadow, []ShadowUpdate{
		{Time: time.UnixMilli(200), Values: map[string]any{"a": 3}},
		{Time: time.UnixMilli(300), Values: map[string]any{"b": 3}},
	}, 1000)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.JSONEq(t, `3`, string(shadow.Reported["a"]))
	assert.JSONEq(t, `3`, string(shadow.Reported["b"]))
	assert.Equal(t, int64(300), shadow.Timestamp)
	assert.Equal(t, int64(1), shadow.Version)
}

func TestApplyShadowUpdatesSkipsOlderBatch(t *testing.T) {
	shadow := newTestShadow()
	shadow.Reported["a"] = json.RawMessage(`3`)
	shadow.Timestamp = 300
	shadow.Version = 1

	changed, err := applyShadowUpdates(shadow, []ShadowUpdate{
		{Time: time.UnixMilli(200), Values: map[string]any{"b": 4}},
	}, 1000)

	require.NoError(t, err)
	assert.False(t, changed)
	assert.NotContains(t, shadow.Reported, "b")
	assert.Equal(t, int64(300), shadow.Timestamp)
	assert.Equal(t, int64(1), shadow.Version)
}

func TestApplyShadowUpdatesSameTimestampLastWins(t *testing.T) {
	shadow := newTestShadow()
	changed, err := applyShadowUpdates(shadow, []ShadowUpdate{
		{Time: time.UnixMilli(200), Values: map[string]any{"a": 1}},
		{Time: time.UnixMilli(200), Values: map[string]any{"a": 2}},
	}, 1000)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.JSONEq(t, `2`, string(shadow.Reported["a"]))
	assert.Equal(t, int64(1), shadow.Version)
}

func TestApplyShadowUpdatesClearsSatisfiedDesired(t *testing.T) {
	shadow := newTestShadow()
	shadow.Desired["a"] = json.RawMessage(`1.0`)
	shadow.Desired["b"] = json.RawMessage(`4`)

	changed, err := applyShadowUpdates(shadow, []ShadowUpdate{
		{Time: time.UnixMilli(200), Values: map[string]any{"a": 1, "b": 5}},
	}, 1000)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.NotContains(t, shadow.Desired, "a")
	assert.JSONEq(t, `4`, string(shadow.Desired["b"]))
	assert.Equal(t, int64(1), shadow.Version)
}

func TestApplyShadowUpdatesClearsAllDesired(t *testing.T) {
	shadow := newTestShadow()
	shadow.Desired["temperature"] = json.RawMessage(`20`)

	changed, err := applyShadowUpdates(shadow, []ShadowUpdate{
		{Time: time.UnixMilli(200), Values: map[string]any{"temperature": 20}},
	}, 1000)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Empty(t, shadow.Desired)
}

func TestApplyShadowUpdatesZeroTimeUsesNow(t *testing.T) {
	shadow := newTestShadow()
	changed, err := applyShadowUpdates(shadow, []ShadowUpdate{
		{Values: map[string]any{"a": 1}},
	}, 12345)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, int64(12345), shadow.Timestamp)
}
