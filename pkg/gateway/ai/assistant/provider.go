package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/aimgr"
)

type chatMessage struct {
	Role       string
	Content    string
	ToolCalls  []toolCall
	ToolCallID string
}

type toolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type chatResult struct {
	Content   string
	ToolCalls []toolCall
}

type provider interface {
	Chat(ctx context.Context, system string, messages []chatMessage, tools []tool.Tool) (*chatResult, error)
}

func newProvider(model *aimgr.RuntimeModel) provider {
	if model.Protocol == aimgr.ProtocolAnthropicMessages {
		return &anthropicProvider{model: model}
	}
	return &openAIProvider{model: model}
}

type openAIProvider struct {
	model *aimgr.RuntimeModel
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

func (p *openAIProvider) Chat(ctx context.Context, system string, messages []chatMessage, tools []tool.Tool) (*chatResult, error) {
	converted := make([]openAIMessage, 0, len(messages)+1)
	if strings.TrimSpace(system) != "" {
		converted = append(converted, openAIMessage{Role: "system", Content: system})
	}
	for _, item := range messages {
		message := openAIMessage{Role: item.Role, Content: item.Content, ToolCallID: item.ToolCallID}
		for _, call := range item.ToolCalls {
			convertedCall := openAIToolCall{ID: call.ID, Type: "function"}
			convertedCall.Function.Name = call.Name
			convertedCall.Function.Arguments = string(call.Arguments)
			message.ToolCalls = append(message.ToolCalls, convertedCall)
		}
		converted = append(converted, message)
	}
	request := map[string]any{
		"model":       p.model.Model,
		"messages":    converted,
		"temperature": 0,
	}
	if len(tools) > 0 {
		request["tools"] = openAITools(tools)
		request["tool_choice"] = "auto"
	}
	var response struct {
		Choices []struct {
			Message struct {
				Content   string           `json:"content"`
				ToolCalls []openAIToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := postJSON(ctx, p.model.EndpointURL, map[string]string{
		"Authorization": "Bearer " + p.model.APIKey,
	}, request, &response); err != nil {
		return nil, err
	}
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("model returned no choices")
	}
	message := response.Choices[0].Message
	result := &chatResult{Content: message.Content}
	for _, call := range message.ToolCalls {
		arguments := json.RawMessage(call.Function.Arguments)
		if len(arguments) == 0 {
			arguments = json.RawMessage(`{}`)
		}
		result.ToolCalls = append(result.ToolCalls, toolCall{ID: call.ID, Name: call.Function.Name, Arguments: arguments})
	}
	return result, nil
}

type openAIToolSpec struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

func openAITools(tools []tool.Tool) []openAIToolSpec {
	out := make([]openAIToolSpec, 0, len(tools))
	for _, item := range tools {
		spec := openAIToolSpec{Type: "function"}
		spec.Function.Name = item.Name()
		spec.Function.Description = item.Description()
		spec.Function.Parameters = item.InputSchema()
		out = append(out, spec)
	}
	return out
}

type anthropicProvider struct {
	model *aimgr.RuntimeModel
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

func (p *anthropicProvider) Chat(ctx context.Context, system string, messages []chatMessage, tools []tool.Tool) (*chatResult, error) {
	converted := make([]anthropicMessage, 0, len(messages))
	for index := 0; index < len(messages); index++ {
		item := messages[index]
		switch item.Role {
		case "assistant":
			blocks := make([]anthropicContent, 0, len(item.ToolCalls)+1)
			if item.Content != "" {
				blocks = append(blocks, anthropicContent{Type: "text", Text: item.Content})
			}
			for _, call := range item.ToolCalls {
				blocks = append(blocks, anthropicContent{Type: "tool_use", ID: call.ID, Name: call.Name, Input: call.Arguments})
			}
			converted = append(converted, anthropicMessage{Role: "assistant", Content: blocks})
		case "tool":
			blocks := make([]anthropicContent, 0, 4)
			for ; index < len(messages) && messages[index].Role == "tool"; index++ {
				toolMessage := messages[index]
				blocks = append(blocks, anthropicContent{
					Type: "tool_result", ToolUseID: toolMessage.ToolCallID, Content: toolMessage.Content,
				})
			}
			index--
			converted = append(converted, anthropicMessage{Role: "user", Content: blocks})
		default:
			converted = append(converted, anthropicMessage{Role: "user", Content: []anthropicContent{{Type: "text", Text: item.Content}}})
		}
	}
	request := map[string]any{
		"model":      p.model.Model,
		"max_tokens": 2048,
		"messages":   converted,
	}
	if strings.TrimSpace(system) != "" {
		request["system"] = system
	}
	if len(tools) > 0 {
		specs := make([]anthropicTool, 0, len(tools))
		for _, item := range tools {
			specs = append(specs, anthropicTool{Name: item.Name(), Description: item.Description(), InputSchema: item.InputSchema()})
		}
		request["tools"] = specs
	}
	var response struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := postJSON(ctx, p.model.EndpointURL, map[string]string{
		"x-api-key":         p.model.APIKey,
		"anthropic-version": "2023-06-01",
	}, request, &response); err != nil {
		return nil, err
	}
	result := &chatResult{}
	for _, block := range response.Content {
		switch block.Type {
		case "text":
			result.Content += block.Text
		case "tool_use":
			input := block.Input
			if len(input) == 0 {
				input = json.RawMessage(`{}`)
			}
			result.ToolCalls = append(result.ToolCalls, toolCall{ID: block.ID, Name: block.Name, Arguments: input})
		}
	}
	return result, nil
}

func postJSON(ctx context.Context, endpoint string, headers map[string]string, payload, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode model request: %w", err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create model request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("request model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("model returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(result); err != nil {
		return fmt.Errorf("decode model response: %w", err)
	}
	return nil
}
