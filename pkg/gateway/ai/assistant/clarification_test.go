package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/aimgr"
	"harns-platform/pkg/gateway/sqlite"
)

func TestParseClarification(t *testing.T) {
	value, err := parseClarification(json.RawMessage(`{
		"question": " 请选择设备协议 ",
		"options": ["Modbus TCP", "Modbus RTU", "Modbus RTU", " "],
		"allowCustom": false
	}`))
	require.NoError(t, err)
	assert.Equal(t, "请选择设备协议", value.Question)
	assert.Equal(t, []string{"Modbus TCP", "Modbus RTU"}, value.Options)
	assert.False(t, value.AllowCustom)
}

func TestParseClarificationDefaultsCustomInput(t *testing.T) {
	value, err := parseClarification(json.RawMessage(`{
		"question": "请选择",
		"options": ["A", "B"]
	}`))
	require.NoError(t, err)
	assert.True(t, value.AllowCustom)
}

func TestParseClarificationRejectsInvalidOptions(t *testing.T) {
	_, err := parseClarification(json.RawMessage(`{
		"question": "请选择",
		"options": ["A"]
	}`))
	require.Error(t, err)
}

func TestFindClarificationCall(t *testing.T) {
	call := findClarificationCall([]toolCall{
		{Name: "collections_list"},
		{Name: "assistant_ask_user", Arguments: json.RawMessage(`{}`)},
	}, map[string]string{"assistant_ask_user": askUserToolName})
	require.NotNil(t, call)
	assert.Equal(t, "assistant_ask_user", call.Name)
}

func TestAskUserToolSchemaIsValid(t *testing.T) {
	assert.True(t, json.Valid(askUserToolSchema))
	assert.NotEmpty(t, askUserToolName)
}

func TestChatReturnsClarification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","type":"function","function":{"name":"assistant_ask_user","arguments":"{\"question\":\"请选择设备协议\",\"options\":[\"Modbus TCP\",\"OPC UA\"],\"allowCustom\":true}"}}]}}]}`))
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
		Messages: []Message{{Role: "user", Content: "帮我创建采集"}},
	})
	require.NoError(t, err)
	require.NotNil(t, response.Clarification)
	assert.Equal(t, "请选择设备协议", response.Reply)
	assert.Equal(t, []string{"Modbus TCP", "OPC UA"}, response.Clarification.Options)
	assert.True(t, response.Clarification.AllowCustom)
}
