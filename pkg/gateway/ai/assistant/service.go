package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/aimgr"
	"harns-platform/pkg/logging"
)

const (
	maxToolRounds      = 4
	maxMessages        = 20
	maxToolResultLen   = 32 << 10
	maxAttachments     = 5
	maxAttachmentBytes = 256 << 10
	maxAttachmentTotal = 512 << 10
)

// Message is one public chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Attachment is one text attachment supplied with the current turn.
type Attachment struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ChatRequest is the assistant chat request.
type ChatRequest struct {
	Messages    []Message    `json:"messages"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// ChatResponse is the assistant chat response.
type ChatResponse struct {
	Reply string   `json:"reply"`
	Tools []string `json:"tools,omitempty"`
}

// Service runs the bounded assistant tool loop.
type Service struct {
	config *aimgr.Service
	tools  *tool.Registry
}

// NewService creates an assistant service.
func NewService(config *aimgr.Service, tools *tool.Registry) *Service {
	return &Service{config: config, tools: tools}
}

// Chat runs one assistant conversation turn.
func (s *Service) Chat(ctx context.Context, principal tool.Principal, lang string, req *ChatRequest) (*ChatResponse, error) {
	defer logging.Trace("assistant.Chat", "userId", principal.UserID)()

	if s == nil || s.config == nil {
		logging.ErrorS(fmt.Errorf("assistant is not configured"), "AI assistant unavailable")
		return nil, response.ErrAIBusy()
	}
	if s.tools == nil {
		logging.ErrorS(fmt.Errorf("tool registry is not configured"), "AI assistant unavailable")
		return nil, response.ErrAIBusy()
	}
	model, err := s.config.RuntimeModel(ctx)
	if err != nil {
		logging.ErrorS(err, "AI assistant model configuration is invalid")
		return nil, response.ErrAIBusy()
	}
	messages := toChatMessages(req.Messages)
	if len(messages) == 0 {
		return nil, response.ErrAIProviderInvalid("messages are required")
	}
	if err := appendAttachments(messages, req.Attachments); err != nil {
		return nil, err
	}
	available := s.tools.List()
	providerTools, toolNames, err := providerToolAliases(available)
	if err != nil {
		logging.ErrorS(err, "AI assistant tool registration is invalid")
		return nil, response.ErrAIBusy()
	}
	provider := newProvider(model)
	toolNamesForPrompt := make([]string, 0, len(providerTools))
	for _, item := range providerTools {
		toolNamesForPrompt = append(toolNamesForPrompt, item.Name())
	}
	system, err := renderSystemPrompt(lang, systemPromptData{
		Tools: toolNamesForPrompt,
	})
	if err != nil {
		logging.ErrorS(err, "AI assistant prompt rendering failed")
		return nil, response.ErrAIBusy()
	}
	usedTools := make([]string, 0)

	for round := 0; round < maxToolRounds; round++ {
		result, err := provider.Chat(ctx, system, messages, providerTools)
		if err != nil {
			logging.ErrorS(err, "AI assistant model request failed")
			return nil, response.ErrAIBusy()
		}
		if len(result.ToolCalls) == 0 {
			reply := strings.TrimSpace(result.Content)
			if reply == "" {
				logging.ErrorS(fmt.Errorf("model returned an empty response"), "AI assistant model request failed")
				return nil, response.ErrAIBusy()
			}
			return &ChatResponse{Reply: reply, Tools: usedTools}, nil
		}
		messages = append(messages, chatMessage{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})
		for _, call := range result.ToolCalls {
			toolName := toolNames[call.Name]
			if toolName == "" {
				toolName = call.Name
			}
			usedTools = append(usedTools, toolName)
			content, callErr := s.callTool(ctx, principal, toolCall{
				ID:        call.ID,
				Name:      toolName,
				Arguments: call.Arguments,
			})
			if callErr != nil {
				content = fmt.Sprintf(`{"error":%q}`, callErr.Error())
			}
			messages = append(messages, chatMessage{
				Role:       "tool",
				Content:    trimToolResult(content),
				ToolCallID: call.ID,
			})
		}
	}
	logging.ErrorS(fmt.Errorf("too many tool calls"), "AI assistant tool loop exceeded")
	return nil, response.ErrAIBusy()
}

type aliasedTool struct {
	tool.Tool
	name string
}

func (t aliasedTool) Name() string { return t.name }

func providerToolAliases(items []tool.Tool) ([]tool.Tool, map[string]string, error) {
	aliases := make([]tool.Tool, 0, len(items))
	names := make(map[string]string, len(items))
	for _, item := range items {
		name := sanitizeToolName(item.Name())
		if existing, ok := names[name]; ok {
			return nil, nil, fmt.Errorf("tool name alias conflict between %q and %q", existing, item.Name())
		}
		names[name] = item.Name()
		aliases = append(aliases, aliasedTool{Tool: item, name: name})
	}
	return aliases, names, nil
}

func sanitizeToolName(value string) string {
	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z',
			char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '_', char == '-':
			builder.WriteRune(char)
		default:
			builder.WriteByte('_')
		}
	}
	name := builder.String()
	if name == "" {
		return "tool"
	}
	return name
}

func (s *Service) callTool(ctx context.Context, principal tool.Principal, call toolCall) (string, error) {
	value, err := s.tools.Call(ctx, principal, call.Name, call.Arguments)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func appendAttachments(messages []chatMessage, attachments []Attachment) error {
	if len(attachments) == 0 {
		return nil
	}
	if len(attachments) > maxAttachments {
		return response.ErrAIProviderInvalid(fmt.Sprintf("too many attachments, max %d", maxAttachments))
	}
	total := 0
	var builder strings.Builder
	last := len(messages) - 1
	builder.WriteString(messages[last].Content)
	for _, item := range attachments {
		name := strings.TrimSpace(item.Name)
		content := item.Content
		if name == "" {
			return response.ErrAIProviderInvalid("attachment name is required")
		}
		if len(content) > maxAttachmentBytes {
			return response.ErrAIProviderInvalid(fmt.Sprintf("attachment %s is too large", name))
		}
		total += len(content)
		if total > maxAttachmentTotal {
			return response.ErrAIProviderInvalid("attachments are too large")
		}
		builder.WriteString("\n\n--- 附件: ")
		builder.WriteString(name)
		builder.WriteString(" ---\n")
		builder.WriteString(content)
	}
	messages[last].Content = builder.String()
	return nil
}

func toChatMessages(items []Message) []chatMessage {
	if len(items) > maxMessages {
		items = items[len(items)-maxMessages:]
	}
	out := make([]chatMessage, 0, len(items))
	for _, item := range items {
		role := strings.TrimSpace(item.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		out = append(out, chatMessage{Role: role, Content: content})
	}
	return out
}

func trimToolResult(value string) string {
	if len(value) <= maxToolResultLen {
		return value
	}
	return value[:maxToolResultLen] + `..."`
}
