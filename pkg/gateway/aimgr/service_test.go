package aimgr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/aimgr/v1"
	"harns-platform/pkg/gateway/sqlite"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewStore(db, "test-ai-secret")
	require.NoError(t, err)
	require.NoError(t, store.Initialize(context.Background()))
	return NewService(store)
}

func TestModelAndSettingsLifecycle(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	settings, err := service.GetSettings(ctx)
	require.NoError(t, err)
	assert.False(t, settings.AssistantEnabled)
	assert.False(t, settings.MCPEnabled)
	assert.False(t, settings.HasAPIKey)

	updated, err := service.UpdateModel(ctx, &v1.UpdateModelReq{
		Name:        "primary",
		Remark:      "primary",
		Model:       "gpt-4.1-mini",
		EndpointURL: "https://api.openai.com/v1/chat/completions",
		APIKey:      "sk-test",
	})
	require.NoError(t, err)
	assert.True(t, updated.HasAPIKey)
	assert.Equal(t, "********", updated.APIKey)
	assert.Equal(t, "primary", updated.Name)
	assert.Equal(t, "gpt-4.1-mini", updated.Model)
	assert.Equal(t, "primary", updated.Remark)

	updated, err = service.UpdateModel(ctx, &v1.UpdateModelReq{
		Name:        "backup",
		Remark:      "backup",
		Model:       "claude-sonnet-4",
		EndpointURL: "https://api.anthropic.com/v1/messages",
	})
	require.NoError(t, err)
	assert.True(t, updated.HasAPIKey)
	assert.Equal(t, "backup", updated.Name)
	assert.Equal(t, "claude-sonnet-4", updated.Model)

	updated, err = service.UpdateSettings(ctx, &v1.UpdateSettingsReq{
		AssistantEnabled: true,
		MCPEnabled:       true,
	})
	require.NoError(t, err)
	assert.True(t, updated.AssistantEnabled)
	assert.True(t, updated.MCPEnabled)
	assert.Equal(t, "backup", updated.Name)
	assert.Equal(t, "claude-sonnet-4", updated.Model)
}

func TestMCPTokenLifecycle(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	settings, err := service.UpdateSettings(ctx, &v1.UpdateSettingsReq{
		AssistantEnabled: false,
		MCPEnabled:       true,
	})
	require.NoError(t, err)
	assert.True(t, settings.MCPEnabled)
	assert.NotEmpty(t, settings.MCPToken)

	enabledToken := settings.MCPToken
	settings, err = service.UpdateSettings(ctx, &v1.UpdateSettingsReq{
		AssistantEnabled: false,
		MCPEnabled:       false,
	})
	require.NoError(t, err)
	assert.False(t, settings.MCPEnabled)
	assert.Empty(t, settings.MCPToken)

	settings, err = service.UpdateSettings(ctx, &v1.UpdateSettingsReq{
		AssistantEnabled: false,
		MCPEnabled:       true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, settings.MCPToken)
	assert.NotEqual(t, enabledToken, settings.MCPToken)

}
