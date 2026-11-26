package assistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSystemPrompt(t *testing.T) {
	data := systemPromptData{
		Tools: []string{"collections_list", "drivers_get_schema"},
	}

	zh, err := renderSystemPrompt("zh-CN", data)
	require.NoError(t, err)
	assert.Contains(t, zh, "网关的 AI 助手")
	assert.Contains(t, zh, "collections_list")
	assert.Contains(t, zh, "**异常：具体信息**")
	assert.Contains(t, zh, "计算变量")
	assert.Contains(t, zh, "dataTopic")
	assert.Contains(t, zh, "commandTopic")
	assert.Contains(t, zh, "Markdown")
	assert.Contains(t, zh, "assistant_ask_user")
	assert.Contains(t, zh, "一次只问一个问题")

	en, err := renderSystemPrompt("en-US", data)
	require.NoError(t, err)
	assert.Contains(t, en, "industrial IoT gateway")
	assert.Contains(t, en, "**Error: details**")
	assert.Contains(t, en, "calculated variables")
	assert.Contains(t, en, "dataTopic")
	assert.Contains(t, en, "commandTopic")
	assert.Contains(t, en, "Markdown")
	assert.Contains(t, en, "assistant_ask_user")
	assert.Contains(t, en, "exactly one question")

	fallback, err := renderSystemPrompt("fr-FR", data)
	require.NoError(t, err)
	assert.Equal(t, zh, fallback)
}
