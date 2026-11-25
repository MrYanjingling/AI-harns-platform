package updatemgr

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/sqlite"
)

func TestSettingsStoreDefaultsAndUpdate(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "gateway.db"))
	require.NoError(t, err)
	defer db.Close()

	store, err := newSettingsStore(db, "test-secret")
	require.NoError(t, err)
	require.NoError(t, store.Initialize(context.Background()))

	settings, err := store.Get(context.Background())
	require.NoError(t, err)
	assert.False(t, settings.Enabled)
	assert.Empty(t, settings.BrokerURL)

	require.NoError(t, store.Update(context.Background(), &Settings{
		Enabled:   true,
		BrokerURL: "tcp://mqtt.internal:1883",
		Username:  "gateway",
		Password:  "secret",
	}))
	settings, err = store.Get(context.Background())
	require.NoError(t, err)
	assert.True(t, settings.Enabled)
	assert.Equal(t, "tcp://mqtt.internal:1883", settings.BrokerURL)
	assert.Equal(t, "gateway", settings.Username)
	assert.Equal(t, "secret", settings.Password)
	assert.True(t, settings.toVo().HasPassword)
}
