package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/aimgr"
)

func TestAnthropicCombinesToolResultsIntoNextUserMessage(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	}))
	defer server.Close()

	provider := newProvider(&aimgr.RuntimeModel{
		Protocol:    aimgr.ProtocolAnthropicMessages,
		EndpointURL: server.URL,
		Model:       "test-model",
		APIKey:      "test-key",
	})
	_, err := provider.Chat(context.Background(), "system", []chatMessage{
		{Role: "user", Content: "query"},
		{Role: "assistant", ToolCalls: []toolCall{
			{ID: "call-1", Name: "gateway_get_meta", Arguments: json.RawMessage(`{}`)},
			{ID: "call-2", Name: "collections_list", Arguments: json.RawMessage(`{}`)},
		}},
		{Role: "tool", ToolCallID: "call-1", Content: `{"ok":true}`},
		{Role: "tool", ToolCallID: "call-2", Content: `{"ok":true}`},
	}, nil)
	require.NoError(t, err)

	messages, ok := captured["messages"].([]any)
	require.True(t, ok)
	require.Len(t, messages, 3)
	next, ok := messages[2].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "user", next["role"])
	blocks, ok := next["content"].([]any)
	require.True(t, ok)
	require.Len(t, blocks, 2)
	for index, block := range blocks {
		item, ok := block.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "tool_result", item["type"])
		assert.Equal(t, []string{"call-1", "call-2"}[index], item["tool_use_id"])
	}
}
