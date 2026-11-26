package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/aimgr"
	"harns-platform/pkg/gateway/sqlite"
)

func TestTrimToolResultReturnsValidJSONWhenTooLarge(t *testing.T) {
	value := `{"items":["` + strings.Repeat("x", maxToolResultLen) + `"]}`
	trimmed := trimToolResult("drivers.list", value)
	require.LessOrEqual(t, len(trimmed), maxToolResultLen)
	require.True(t, json.Valid([]byte(trimmed)))
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(trimmed), &payload))
	assert.Equal(t, true, payload["truncated"])
	assert.Equal(t, "drivers.list", payload["tool"])
}

func TestTrimToolResultKeepsSmallValue(t *testing.T) {
	value := `{"ok":true}`
	assert.Equal(t, value, trimToolResult("test", value))
}

func TestToolCallSignature(t *testing.T) {
	calls := []toolCall{
		{Name: "collections.list", Arguments: json.RawMessage(`{"page":1}`)},
		{Name: "drivers.list", Arguments: json.RawMessage(`{}`)},
	}
	first := toolCallSignature(calls)
	second := toolCallSignature(calls)
	assert.Equal(t, first, second)
	assert.NotEmpty(t, first)
	assert.Empty(t, toolCallSignature(nil))
}

func TestChatFallsBackWhenToolCallsRepeat(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.Header().Set("Content-Type", "application/json")
		if _, ok := payload["tools"]; ok {
			requests++
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","type":"function","function":{"name":"collections_list","arguments":"{}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"工具调用次数过多，请补充信息后重试。"}}]}`))
	}))
	defer server.Close()

	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()
	store, err := aimgr.NewStore(db, "test-secret")
	require.NoError(t, err)
	require.NoError(t, store.Initialize(context.Background()))
	require.NoError(t, store.UpdateSettings(context.Background(), &aimgr.Settings{
		AssistantEnabled: true,
		Name:             "test",
		Model:            "test-model",
		EndpointURL:      server.URL,
		APIKey:           "test-key",
	}))
	registry, err := tool.NewRegistry()
	require.NoError(t, err)
	service := NewService(aimgr.NewService(store), registry)

	response, err := service.Chat(context.Background(), tool.Principal{UserID: "user-1"}, "zh-CN", &ChatRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	assert.Contains(t, response.Reply, "工具调用次数过多")
	assert.GreaterOrEqual(t, requests, 3)
}
