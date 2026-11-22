package pushmgr

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReadySendStore struct {
	mu      sync.Mutex
	rows    []*ReadySendBatch
	nextSeq int64
	deleted []int64
}

func (s *fakeReadySendStore) SaveReadySend(_ context.Context, batch *ReadySendBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := *batch
	copied.Payload = append([]byte(nil), batch.Payload...)
	s.nextSeq++
	copied.Seq = s.nextSeq
	s.rows = append(s.rows, &copied)
	return nil
}

func (s *fakeReadySendStore) ListReadySend(_ context.Context, groupID string, limit int) ([]*ReadySendBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*ReadySendBatch, 0)
	for _, row := range s.rows {
		if row.GroupID != groupID {
			continue
		}
		out = append(out, row)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *fakeReadySendStore) DeleteReadySend(_ context.Context, groupID string, seq int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.rows[:0]
	for _, row := range s.rows {
		if row.GroupID == groupID && row.Seq == seq {
			s.deleted = append(s.deleted, seq)
			continue
		}
		out = append(out, row)
	}
	s.rows = out
	return nil
}

func (s *fakeReadySendStore) ReadySendStats(_ context.Context, groupID string) (ReadySendStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var stats ReadySendStats
	for _, row := range s.rows {
		if row.GroupID != groupID {
			continue
		}
		stats.Messages += int64(row.PayloadSize)
		stats.Batches++
		if stats.Oldest.IsZero() || row.CreatedTime.Before(stats.Oldest) {
			stats.Oldest = row.CreatedTime
		}
	}
	return stats, nil
}

func newOutboxTestRuntime(t *testing.T, connector *fakeConnector) (*RuntimeManager, *fakeReadySendStore) {
	t.Helper()
	store := &fakeReadySendStore{}
	runtime, err := NewRuntimeManager(&fakeSource{}, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.SetReadySendStore(store)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{
		ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`),
	})
	return runtime, store
}

func outboxTestGroup() *groupRuntime {
	return &groupRuntime{
		group: &PushGroup{ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group"},
		hub:   newPayloadHub(),
	}
}

func outboxMessage(collectionID string) propertyMessage {
	return propertyMessage{Type: "property", Payload: []propertyItem{{CollectionID: collectionID}}}
}

func TestSendFreshBuffersAndSpillsAtHundred(t *testing.T) {
	connector := &fakeConnector{publishErr: context.DeadlineExceeded}
	runtime, store := newOutboxTestRuntime(t, connector)
	group := outboxTestGroup()
	memory := make([]propertyMessage, 0, 128)
	messages := make([]propertyMessage, 101)
	for i := range messages {
		messages[i] = outboxMessage("c1")
	}

	outbox := runtime.newGroupOutbox(group)
	outbox.memory = memory
	outbox.sendFresh(context.Background(), messages)

	require.Len(t, outbox.memory, 1)
	require.Len(t, store.rows, 1)
	assert.Equal(t, 100, store.rows[0].PayloadSize)
	assert.Equal(t, int64(101), group.pendingMsgs.Load())
	assert.Equal(t, int64(1), group.pendingRows.Load())
	var payload propertyMessage
	require.NoError(t, json.Unmarshal(store.rows[0].Payload, &payload))
	assert.Len(t, payload.Payload, 100)
}

func TestRecoverySendsMemoryBeforeReadySend(t *testing.T) {
	connector := &fakeConnector{}
	runtime, store := newOutboxTestRuntime(t, connector)
	group := outboxTestGroup()
	group.backlog.Store(true)
	group.pendingMsgs.Store(2)
	group.pendingRows.Store(1)
	store.rows = []*ReadySendBatch{{
		Seq: 1, GroupID: "group-1", ConnectorID: "conn-1", Topic: "data/old",
		Payload: mustMerge(t, []propertyMessage{outboxMessage("old")}), PayloadSize: 1,
		CreatedTime: time.Now().UTC(),
	}}
	memory := []propertyMessage{outboxMessage("new")}
	outbox := runtime.newGroupOutbox(group)
	outbox.memory = memory

	require.NoError(t, outbox.recover(context.Background()))

	require.Len(t, connector.payloads, 2)
	assert.Contains(t, string(connector.payloads[0]), `"new"`)
	assert.Contains(t, string(connector.payloads[1]), `"old"`)
	assert.Empty(t, outbox.memory)
	assert.Empty(t, store.rows)
	assert.False(t, group.backlog.Load())
	assert.Zero(t, group.pendingMsgs.Load())
	assert.Zero(t, group.pendingRows.Load())
}

func TestRecoveryKeepsMemoryWhenPublishFails(t *testing.T) {
	connector := &fakeConnector{publishErr: assert.AnError}
	runtime, _ := newOutboxTestRuntime(t, connector)
	group := outboxTestGroup()
	group.backlog.Store(true)
	memory := []propertyMessage{outboxMessage("new")}
	outbox := runtime.newGroupOutbox(group)
	outbox.memory = memory

	err := outbox.recover(context.Background())
	require.Error(t, err)
	require.Len(t, outbox.memory, 1)
	assert.True(t, group.backlog.Load())
}

func TestRecoveryDeletesReadySendWhenPublishFails(t *testing.T) {
	connector := &fakeConnector{publishErr: assert.AnError}
	runtime, store := newOutboxTestRuntime(t, connector)
	group := outboxTestGroup()
	group.backlog.Store(true)
	row := &ReadySendBatch{
		Seq: 1, GroupID: "group-1", ConnectorID: "conn-1", Topic: "data/old",
		Payload: mustMerge(t, []propertyMessage{outboxMessage("old")}), PayloadSize: 1,
		CreatedTime: time.Now().UTC(),
	}
	store.rows = []*ReadySendBatch{row}
	outbox := runtime.newGroupOutbox(group)

	require.NoError(t, outbox.recover(context.Background()))
	assert.Empty(t, store.rows)
	assert.Equal(t, []int64{1}, store.deleted)
}

func TestRecoveryDeletesReadySendWhenConnectorMissing(t *testing.T) {
	connector := &fakeConnector{}
	runtime, store := newOutboxTestRuntime(t, connector)
	group := outboxTestGroup()
	group.backlog.Store(true)
	store.rows = []*ReadySendBatch{{
		Seq: 1, GroupID: "group-1", ConnectorID: "missing", Topic: "data/old",
		Payload: mustMerge(t, []propertyMessage{outboxMessage("old")}), PayloadSize: 1,
		CreatedTime: time.Now().UTC(),
	}}
	outbox := runtime.newGroupOutbox(group)

	require.NoError(t, outbox.recover(context.Background()))
	assert.Empty(t, store.rows)
	assert.Empty(t, connector.payloads)
}

func mustMerge(t *testing.T, messages []propertyMessage) []byte {
	t.Helper()
	payload, err := mergePropertyMessages(messages)
	require.NoError(t, err)
	return payload
}

func TestGroupOutboxFlushPersistsMemory(t *testing.T) {
	connector := &fakeConnector{}
	runtime, store := newOutboxTestRuntime(t, connector)
	group := outboxTestGroup()
	outbox := runtime.newGroupOutbox(group)
	outbox.memory = []propertyMessage{outboxMessage("a"), outboxMessage("b")}

	require.NoError(t, outbox.flush(context.Background()))
	assert.Empty(t, outbox.memory)
	require.Len(t, store.rows, 1)
	assert.Equal(t, 2, store.rows[0].PayloadSize)
}

func TestStopGroupFlushesMemoryToReadySend(t *testing.T) {
	source := &fakeSource{snapshots: []CollectionSnapshot{{
		CollectionID: "c1", Status: "collecting", Values: map[string]any{"a": 1},
	}}}
	connector := &fakeConnector{publishErr: assert.AnError}
	store := &fakeReadySendStore{}
	runtime, err := NewRuntimeManager(source, "gateway-1", &fakeFactory{connector: connector})
	require.NoError(t, err)
	runtime.SetReadySendStore(store)
	runtime.UpsertConnector(context.Background(), &ConnectorRecord{
		ID: "conn-1", Type: ConnectorTypeMQTT, PushConfig: json.RawMessage(`{}`),
	})
	runtime.UpsertGroup(&PushGroup{
		ID: "group-1", ConnectorID: "conn-1", DataTopic: "data/group",
		RunState: ConnectorRunStateRunning, AllCollections: true, PushIntervalMs: 10000,
	})
	runtime.enqueueGroupSnapshot(context.Background(), "group-1")
	require.Eventually(t, func() bool {
		msgs, _, _ := runtime.GroupPending("group-1")
		return msgs > 0
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, runtime.StopGroup("group-1"))
	require.Len(t, store.rows, 1)
	assert.Equal(t, 1, store.rows[0].PayloadSize)
}
