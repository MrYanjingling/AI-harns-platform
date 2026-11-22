package pushmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/pushmgr/v1"
)

func TestReadySendStoreLifecycle(t *testing.T) {
	service, store := newPushTestStore(t)
	ctx := context.Background()
	group := createReadySendGroup(t, service)
	created := time.Now().UTC()
	batch := &ReadySendBatch{
		GroupID:     group.ID,
		ConnectorID: group.ConnectorID,
		Topic:       "data/g1",
		Payload:     []byte(`{"type":"property","payload":[]}`),
		PayloadSize: 100,
		CreatedTime: created,
	}
	require.NoError(t, store.SaveReadySend(ctx, batch))
	assert.NotZero(t, batch.Seq)

	stats, err := store.ReadySendStats(ctx, group.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), stats.Messages)
	assert.Equal(t, int64(1), stats.Batches)
	assert.WithinDuration(t, created, stats.Oldest, time.Second)

	rows, err := store.ListReadySend(ctx, group.ID, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, group.ConnectorID, rows[0].ConnectorID)
	assert.Equal(t, "data/g1", rows[0].Topic)
	assert.Equal(t, 100, rows[0].PayloadSize)

	require.NoError(t, store.DeleteReadySend(ctx, group.ID, batch.Seq))
	stats, err = store.ReadySendStats(ctx, group.ID)
	require.NoError(t, err)
	assert.Zero(t, stats.Messages)
	assert.Zero(t, stats.Batches)
}

func TestReadySendTrimByMessageCap(t *testing.T) {
	service, store := newPushTestStore(t)
	ctx := context.Background()
	group := createReadySendGroup(t, service)
	for i := 0; i < 101; i++ {
		require.NoError(t, store.SaveReadySend(ctx, &ReadySendBatch{
			GroupID:     group.ID,
			ConnectorID: group.ConnectorID,
			Topic:       "data/g1",
			Payload:     []byte(fmt.Sprintf(`{"type":"property","payload":[%d]}`, i)),
			PayloadSize: 100,
			CreatedTime: time.Now().UTC(),
		}))
	}
	stats, err := store.ReadySendStats(ctx, group.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(DefaultReadySendMaxMessages), stats.Messages)
	assert.Equal(t, int64(100), stats.Batches)

	rows, err := store.ListReadySend(ctx, group.ID, 10)
	require.NoError(t, err)
	require.Len(t, rows, 10)
	assert.Equal(t, int64(2), rows[0].Seq)
}

func TestMergePropertyMessages(t *testing.T) {
	payload, err := mergePropertyMessages([]propertyMessage{
		{Type: "property", Payload: []propertyItem{{CollectionID: "a", Time: 1}}},
		{Type: "property", Payload: []propertyItem{{CollectionID: "b", Time: 2}}},
	})
	require.NoError(t, err)
	var merged propertyMessage
	require.NoError(t, json.Unmarshal(payload, &merged))
	require.Len(t, merged.Payload, 2)
	assert.Equal(t, "a", merged.Payload[0].CollectionID)
	assert.Equal(t, "b", merged.Payload[1].CollectionID)
}

func TestReadySendCascadeOnGroupDelete(t *testing.T) {
	service, store := newPushTestStore(t)
	ctx := context.Background()
	connector, err := service.CreateConnector(ctx, &v1.CreateConnectorReq{
		Name: "c", PushConfig: json.RawMessage(`{"brokerUrl":"tcp://127.0.0.1:1883"}`),
	})
	require.NoError(t, err)
	group, err := service.CreateGroup(ctx, &v1.CreatePushGroupReq{
		Name: "g", ConnectorID: connector.ID, DataTopic: "data", PushIntervalMs: 1000, AllCollections: true,
	})
	require.NoError(t, err)
	require.NoError(t, store.SaveReadySend(ctx, &ReadySendBatch{
		GroupID: group.ID, ConnectorID: connector.ID, Topic: "data", Payload: []byte(`{}`),
		PayloadSize: 1, CreatedTime: time.Now().UTC(),
	}))
	require.NoError(t, service.DeleteGroup(ctx, group.ID))
	stats, err := store.ReadySendStats(ctx, group.ID)
	require.NoError(t, err)
	assert.Zero(t, stats.Batches)
}

func createReadySendGroup(t *testing.T, service *Service) *v1.PushGroupVo {
	t.Helper()
	ctx := context.Background()
	connector, err := service.CreateConnector(ctx, &v1.CreateConnectorReq{
		Name: "ready-send-connector", PushConfig: json.RawMessage(`{"brokerUrl":"tcp://127.0.0.1:1883"}`),
	})
	require.NoError(t, err)
	group, err := service.CreateGroup(ctx, &v1.CreatePushGroupReq{
		Name: "ready-send-group", ConnectorID: connector.ID, DataTopic: "data",
		PushIntervalMs: 1000, AllCollections: true,
	})
	require.NoError(t, err)
	return group
}

func TestReadySendCascadeOnConnectorDelete(t *testing.T) {
	service, store := newPushTestStore(t)
	ctx := context.Background()
	group := createReadySendGroup(t, service)
	require.NoError(t, store.SaveReadySend(ctx, &ReadySendBatch{
		GroupID: group.ID, ConnectorID: group.ConnectorID, Topic: "data", Payload: []byte(`{}`),
		PayloadSize: 1, CreatedTime: time.Now().UTC(),
	}))

	require.ErrorIs(t, service.DeleteConnector(ctx, group.ConnectorID, false), ErrConnectorInUse)
	stats, err := store.ReadySendStats(ctx, group.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Batches)

	require.NoError(t, service.DeleteConnector(ctx, group.ConnectorID, true))
	stats, err = store.ReadySendStats(ctx, group.ID)
	require.NoError(t, err)
	assert.Zero(t, stats.Batches)
}
