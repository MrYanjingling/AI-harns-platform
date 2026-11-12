package gateway

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	assert.False(t, first.ModTime.IsZero())

	second, err := service.LoadOrCreate(ctx)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, first.Version, second.Version)
	assert.Equal(t, first.Name, second.Name)
	assert.Equal(t, first.Secret, second.Secret)
}
