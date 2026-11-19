package gatewaymgr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/runtime"
	"harns-platform/pkg/gateway/sqlite"
	"harns-platform/pkg/gateway/validation"
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
	assert.Len(t, first.Version, 32)
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

	stored, err := store.GetMeta(ctx)
	require.NoError(t, err)
	stored.Secret = ""
	require.NoError(t, store.UpdateMeta(ctx, stored))
	third, err := service.LoadOrCreate(ctx)
	require.NoError(t, err)
	assert.Len(t, third.Secret, 64)
	assert.NotEqual(t, first.Secret, third.Secret)

	stored, err = store.GetMeta(ctx)
	require.NoError(t, err)
	stored.Version = strings.Repeat("a", 40)
	require.NoError(t, store.UpdateMeta(ctx, stored))
	fourth, err := service.LoadOrCreate(ctx)
	require.NoError(t, err)
	assert.Len(t, fourth.Version, 32)

	var payload []byte
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT payload FROM gateway_meta WHERE singleton = 1`).Scan(&payload))
	assert.NotContains(t, string(payload), fourth.ID)
	assert.NotContains(t, string(payload), fourth.Secret)
	assert.NotContains(t, string(payload), fourth.Name)
	assert.NotContains(t, string(payload), fourth.InitTime.Format(time.RFC3339Nano))
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

func TestGatewayServiceRename(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	store := NewStore(db)
	require.NoError(t, store.Initialize(ctx))
	service := NewService(store)
	first, err := service.LoadOrCreate(ctx)
	require.NoError(t, err)

	eTag, err := service.Rename(ctx, first.ID, "  plant gateway  ")
	require.NoError(t, err)
	updated, err := store.GetMeta(ctx)
	require.NoError(t, err)
	assert.Equal(t, "plant gateway", updated.Name)
	assert.Equal(t, first.ID, updated.ID)
	assert.Equal(t, first.Secret, updated.Secret)
	assert.NotEqual(t, first.Version, eTag)
	assert.Equal(t, eTag, updated.Version)
	assert.Equal(t, first.InitTime, updated.InitTime)
	assert.Equal(t, first.ModTime, updated.ModTime)

	for _, invalid := range []string{"", "   ", "bad/name", strings.Repeat("a", 65)} {
		_, err = service.Rename(ctx, first.ID, invalid)
		require.Error(t, err, invalid)
		var validationErr *validation.ValidationError
		require.True(t, errors.As(err, &validationErr), invalid)
		assert.Equal(t, "name", validationErr.Issues[0].Field)
	}
}

func TestGatewayMetaIgnoresLegacyGatewayTable(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE gateway (id TEXT PRIMARY KEY, name TEXT NOT NULL, version TEXT NOT NULL, secret TEXT NOT NULL DEFAULT '', init_time TEXT NOT NULL, mod_time TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO gateway (id, name, version, secret, init_time, mod_time) VALUES ('old', 'old', 'old', 'old', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)

	store := NewStore(db)
	require.NoError(t, store.Initialize(ctx))
	exists, err := tableExists(ctx, db, "gateway")
	require.NoError(t, err)
	assert.False(t, exists)
	_, err = store.GetMeta(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGatewayMetaDecryptFailsForCorruptPayload(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	store := NewStore(db)
	require.NoError(t, store.Initialize(ctx))
	service := NewService(store)
	_, err = service.LoadOrCreate(ctx)
	require.NoError(t, err)

	_, err = store.db.ExecContext(ctx, `UPDATE gateway_meta SET payload = ? WHERE singleton = 1`, []byte("corrupt"))
	require.NoError(t, err)
	_, err = store.GetMeta(ctx)
	require.Error(t, err)
}
