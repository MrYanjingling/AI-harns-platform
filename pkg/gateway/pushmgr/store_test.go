package pushmgr

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/gateway/pushmgr/v1"
	"harns-platform/pkg/gateway/sqlite"
)

func newPushTestService(t *testing.T) *Service {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	return NewService(store, &fakeSource{}, nil)
}

func TestPushConnectorAndGroupCRUD(t *testing.T) {
	service := newPushTestService(t)
	ctx := context.Background()

	connector, err := service.CreateConnector(ctx, &v1.CreateConnectorReq{
		Name: "remote",
		PushConfig: json.RawMessage(`{
			"brokerUrl":"tcp://127.0.0.1:1883",
			"dataTopic":"data"
		}`),
	})
	require.NoError(t, err)
	assert.Equal(t, ConnectorTypeMQTT, connector.Type)
	assert.Equal(t, int64(1), connector.Version)
	assert.Equal(t, ConnectorStatusStopped, connector.Status)
	assert.NotZero(t, connector.CreatedTime)

	list, err := service.ListConnectors(ctx, apis.PageQuery{}, "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)

	group, err := service.CreateGroup(ctx, &v1.CreatePushGroupReq{
		Name:           "group",
		ConnectorID:    connector.ID,
		PushIntervalMs: 0,
		AllCollections: true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(DefaultPushIntervalMs), group.PushIntervalMs)
	assert.Equal(t, DefaultMaxCollectionsPerMessage, group.MaxCollectionsPerPushMessage)
	assert.True(t, group.AllCollections)

	group, err = service.UpdateGroup(ctx, group.ID, &v1.UpdatePushGroupReq{
		Name:           "group-v2",
		ConnectorID:    connector.ID,
		PushIntervalMs: 2000,
		AllCollections: false,
		CollectionIDs:  []string{"c1"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), group.Version)
	assert.False(t, group.AllCollections)

	err = service.DeleteConnector(ctx, connector.ID, false)
	require.ErrorIs(t, err, ErrConnectorInUse)

	err = service.DeleteConnector(ctx, connector.ID, true)
	require.NoError(t, err)
	_, err = service.GetGroup(ctx, group.ID)
	require.ErrorIs(t, err, ErrGroupNotFound)
}
