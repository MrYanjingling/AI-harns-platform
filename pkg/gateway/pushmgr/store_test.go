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

	list, err := service.ListConnectors(ctx, apis.PageQuery{}, "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
	assert.Equal(t, int64(0), list.List[0].GroupCount)

	filtered, err := service.ListConnectors(ctx, apis.PageQuery{}, "", ConnectorTypeKafka)
	require.NoError(t, err)
	assert.Equal(t, int64(0), filtered.Total)

	group, err := service.CreateGroup(ctx, &v1.CreatePushGroupReq{
		Name:           "group",
		ConnectorID:    connector.ID,
		DataTopic:      "data/group",
		PushIntervalMs: 0,
		AllCollections: true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(DefaultPushIntervalMs), group.PushIntervalMs)
	assert.Equal(t, DefaultMaxCollectionsPerMessage, group.MaxCollectionsPerPushMessage)
	assert.True(t, group.AllCollections)

	list, err = service.ListConnectors(ctx, apis.PageQuery{}, "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.List[0].GroupCount)

	groups, err := service.ListGroups(ctx, apis.PageQuery{}, "", connector.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), groups.Total)

	group, err = service.UpdateGroup(ctx, group.ID, &v1.UpdatePushGroupReq{
		Name:           "group-v2",
		ConnectorID:    connector.ID,
		DataTopic:      "data/group-v2",
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

func TestStoreMigratesGroupTopicsFromConnector(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE push_connectors (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			push_config TEXT NOT NULL DEFAULT '{}',
			version INTEGER NOT NULL DEFAULT 1,
			created_time TEXT NOT NULL,
			updated_time TEXT NOT NULL
		);
		CREATE TABLE push_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			connector_id TEXT NOT NULL,
			push_interval_ms INTEGER NOT NULL,
			all_collections INTEGER NOT NULL DEFAULT 0,
			collection_ids TEXT NOT NULL DEFAULT '[]',
			max_collections_per_push_message INTEGER NOT NULL DEFAULT 10,
			version INTEGER NOT NULL DEFAULT 1,
			created_time TEXT NOT NULL,
			updated_time TEXT NOT NULL
		);
		INSERT INTO push_connectors (id, type, name, push_config, version, created_time, updated_time)
		VALUES ('c1', 'mqtt', 'legacy', '{"brokerUrl":"tcp://127.0.0.1:1883","dataTopic":"legacy/data","commandTopic":"legacy/cmd"}', 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO push_groups (id, name, connector_id, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time)
		VALUES ('g1', 'legacy-group', 'c1', 1000, 1, '[]', 10, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`)
	require.NoError(t, err)
	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	group, err := store.GetGroup(context.Background(), "g1")
	require.NoError(t, err)
	assert.Equal(t, "legacy/data", group.DataTopic)
	assert.Equal(t, "legacy/cmd", group.CommandTopic)
	assert.Equal(t, ConnectorRunStateRunning, group.RunState)
}

func TestRunStateLifecycle(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	source := &fakeSource{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: &fakeConnector{}})
	require.NoError(t, err)
	service := NewService(store, source, runtime)
	ctx := context.Background()

	connector, err := service.CreateConnector(ctx, &v1.CreateConnectorReq{
		Name: "run-state",
		PushConfig: json.RawMessage(`{
			"brokerUrl":"tcp://127.0.0.1:1883"
		}`),
	})
	require.NoError(t, err)
	assert.Equal(t, ConnectorRunStateRunning, connector.RunState)
	assert.Equal(t, ConnectorStatusRunning, connector.Status)

	_, err = service.StopConnector(ctx, connector.ID)
	require.NoError(t, err)
	stored, err := store.GetConnector(ctx, connector.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorRunStateStopped, stored.RunState)

	status, err := service.StartConnector(ctx, connector.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorStatusRunning, status)
	stored, err = store.GetConnector(ctx, connector.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorRunStateRunning, stored.RunState)

	status, err = service.RestartConnector(ctx, connector.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorStatusRunning, status)

	group, err := service.CreateGroup(ctx, &v1.CreatePushGroupReq{
		Name: "run-state-group", ConnectorID: connector.ID, DataTopic: "data/run-state",
		PushIntervalMs: 10000, AllCollections: true,
	})
	require.NoError(t, err)
	assert.Equal(t, ConnectorRunStateRunning, group.RunState)
	assert.Equal(t, ConnectorStatusRunning, group.Status)

	_, err = service.StopGroup(ctx, group.ID)
	require.NoError(t, err)
	storedGroup, err := store.GetGroup(ctx, group.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorRunStateStopped, storedGroup.RunState)

	status, err = service.StartGroup(ctx, group.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorStatusRunning, status)

	status, err = service.RestartGroup(group.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorStatusRunning, status)
	runtime.Close(ctx)
}

func TestStopAllStopsGroupsAndConnectors(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	source := &fakeSource{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: &fakeConnector{}})
	require.NoError(t, err)
	service := NewService(store, source, runtime)
	ctx := context.Background()

	connector, err := service.CreateConnector(ctx, &v1.CreateConnectorReq{
		Name:       "stop-all",
		PushConfig: json.RawMessage(`{"brokerUrl":"tcp://127.0.0.1:1883"}`),
	})
	require.NoError(t, err)
	group, err := service.CreateGroup(ctx, &v1.CreatePushGroupReq{
		Name: "stop-all-group", ConnectorID: connector.ID, DataTopic: "data/stop-all",
		PushIntervalMs: 10000, AllCollections: true,
	})
	require.NoError(t, err)

	require.NoError(t, service.StopAll(ctx))
	storedConnector, err := store.GetConnector(ctx, connector.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorRunStateStopped, storedConnector.RunState)
	storedGroup, err := store.GetGroup(ctx, group.ID)
	require.NoError(t, err)
	assert.Equal(t, ConnectorRunStateStopped, storedGroup.RunState)
	runtime.Close(ctx)
}
