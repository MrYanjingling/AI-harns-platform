package things

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/emqx"
)

type fakeThingRelationCleaner struct {
	called chan []string
	err    error
}

func (f *fakeThingRelationCleaner) CleanupThingRelations(_ context.Context, thingIDs []string) error {
	f.called <- append([]string(nil), thingIDs...)
	return f.err
}

func TestDispatchThingRelationCleanup(t *testing.T) {
	cleaner := &fakeThingRelationCleaner{called: make(chan []string, 1)}
	service := &Service{}
	service.SetThingRelationCleaner(cleaner)

	service.dispatchThingRelationCleanup([]DeletedThingRef{
		{ID: "thing-1"},
		{ID: "thing-2"},
		{ID: ""},
	})

	select {
	case got := <-cleaner.called:
		assert.Equal(t, []string{"thing-1", "thing-2"}, got)
	case <-time.After(time.Second):
		t.Fatal("relation cleaner was not called")
	}
}

func TestDispatchThingRelationCleanupWithoutCleaner(t *testing.T) {
	service := &Service{}
	service.dispatchThingRelationCleanup([]DeletedThingRef{{ID: "thing-1"}})
}

func TestDeletedThingCacheKeys(t *testing.T) {
	keys, ids := deletedThingCacheKeys([]DeletedThingRef{
		{ID: "thing-1", Token: "token-1"},
		{ID: "thing-2", Token: ""},
		{ID: "", Token: "token-3"},
	})

	require.Len(t, ids, 2)
	assert.Equal(t, []string{"thing-1", "thing-2"}, ids)
	assert.Equal(t, tokenCacheKey("token-1"), keys[0])
	assert.Equal(t, snapshotCachePrefix+"thing-1", keys[1])
	assert.Equal(t, shadowKeyPrefix+"thing-1", keys[2])
	assert.Equal(t, snapshotCachePrefix+"thing-2", keys[3])
	assert.Equal(t, shadowKeyPrefix+"thing-2", keys[4])
	assert.Equal(t, tokenCacheKey("token-3"), keys[5])
}

func TestDispatchMQTTKick(t *testing.T) {
	called := make(chan []string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v5/clients/kickout/bulk", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		var ids []string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&ids))
		called <- ids
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	service := &Service{emqx: emqx.NewService(emqx.Config{APIURL: server.URL, AK: "ak", SK: "sk"})}
	service.dispatchMQTTKick([]DeletedThingRef{{MQTTClientID: "client-1"}})

	select {
	case got := <-called:
		assert.Equal(t, []string{"client-1"}, got)
	case <-time.After(time.Second):
		t.Fatal("MQTT kick was not called")
	}
}
