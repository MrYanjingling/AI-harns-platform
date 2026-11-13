package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/runtime"
	"harns-platform/pkg/gateway/sqlite"
)

func TestGatewayServiceLoadOrCreate(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	store := NewStore(db)
	require.NoError(t, store.Initialize(ctx))
	require.NoError(t, store.Initialize(ctx))

	_, err = store.GetMeta(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)

	service := NewService(store)
	first, err := service.LoadOrCreate(ctx)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.NotEmpty(t, first.ID)
	assert.Equal(t, "harnsgateway", first.Name)
	assert.NotEmpty(t, first.Version)
	assert.Len(t, first.Secret, 64)
	assert.Regexp(t, "^[0-9a-f]{64}$", first.Secret)
	assert.False(t, first.InitTime.IsZero())
	assert.False(t, first.ModTime.IsZero())

	time.Sleep(2 * time.Millisecond)
	second, err := service.LoadOrCreate(ctx)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, first.Version, second.Version)
	assert.Equal(t, first.Name, second.Name)
	assert.Equal(t, first.Secret, second.Secret)
	assert.Equal(t, first.InitTime, second.InitTime)
	assert.True(t, second.ModTime.After(first.ModTime))
	assert.Equal(t, first.Secret, second.Secret)

	_, err = store.db.ExecContext(ctx, `UPDATE gateway SET secret = '' WHERE id = ?`, first.ID)
	require.NoError(t, err)
	third, err := service.LoadOrCreate(ctx)
	require.NoError(t, err)
	assert.Len(t, third.Secret, 64)
	assert.NotEqual(t, first.Secret, third.Secret)
}

func TestGatewayMetaVoUptimeAndJSON(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	meta := &GatewayMeta{
		Secret:   "secret",
		InitTime: base,
		ObjectMeta: runtime.ObjectMeta{
			Name:    "harnsgateway",
			ID:      "gateway-id",
			Version: "commit-id",
			ModTime: base.Add(time.Hour),
		},
	}
	vo := meta.ToVo(base.Add(24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second))
	assert.Equal(t, "1d2h3m4s", vo.TotalUptime)
	assert.Equal(t, "1d1h3m4s", vo.CurrentUptime)
	assert.Equal(t, "commit-id", vo.GetVersion())

	payload, err := json.Marshal(vo)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	for _, key := range []string{"secret", "name", "id", "eTag", "modTime", "initTime", "totalUptime", "currentUptime"} {
		assert.Contains(t, decoded, key)
	}
}
