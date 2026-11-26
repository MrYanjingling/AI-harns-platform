package assistant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/aimgr"
	"harns-platform/pkg/logging"
)

const (
	maxToolRounds      = 6
	maxMessages        = 20
	maxToolResultLen   = 32 << 10
	maxAttachments     = 5
	maxAttachmentBytes = 256 << 10
	maxAttachmentTotal = 512 << 10
	pendingTTL         = 10 * time.Minute
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

// ToolApproval is one user confirmation for a pending write or action tool.
type ToolApproval struct {
	ID       string `json:"id"`
	Approved bool   `json:"approved"`
}

// ChatRequest is the assistant chat request.
type ChatRequest struct {
	Messages    []Message      `json:"messages"`
	Attachments []Attachment   `json:"attachments,omitempty"`
	DocumentIDs []string       `json:"documentIds,omitempty"`
	PendingID   string         `json:"pendingId,omitempty"`
	Approvals   []ToolApproval `json:"approvals,omitempty"`
}

// PendingToolCall is a write/action tool waiting for user confirmation.
type PendingToolCall struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Risk        tool.Risk       `json:"risk"`
	Arguments   json.RawMessage `json:"arguments"`
}

// ChatResponse is the assistant chat response.
type ChatResponse struct {
	Reply         string            `json:"reply"`
	Tools         []string          `json:"tools,omitempty"`
	Clarification *Clarification    `json:"clarification,omitempty"`
	PendingID     string            `json:"pendingId,omitempty"`
	Pending       []PendingToolCall `json:"pending,omitempty"`
}

type pendingTurn struct {
	ownerID       string
	messages      []chatMessage
	pending       []toolCall
	system        string
	providerTools []tool.Tool
	toolNames     map[string]string
	usedTools     []string
	nextRound     int
	created       time.Time
}

// Service runs the bounded assistant tool loop.
type Service struct {
	config    *aimgr.Service
	tools     *tool.Registry
	documents *DocumentStore

	pendingMu sync.Mutex
	pending   map[string]*pendingTurn
}

// NewService creates an assistant service.
func NewService(config *aimgr.Service, tools *tool.Registry, documents ...*DocumentStore) *Service {
	service := &Service{config: config, tools: tools, pending: make(map[string]*pendingTurn)}
	if len(documents) > 0 {
		service.documents = documents[0]
	}
	return service
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
	provider := newProvider(model)

	var (
		messages      []chatMessage
		system        string
		providerTools []tool.Tool
		toolNames     map[string]string
		usedTools     []string
		startRound    int
	)
	if pendingID := strings.TrimSpace(req.PendingID); pendingID != "" {
		state, err := s.takePending(pendingID, principal.UserID)
		if err != nil {
			return nil, err
		}
		messages = state.messages
		system = state.system
		providerTools = state.providerTools
		toolNames = state.toolNames
		usedTools = state.usedTools
		startRound = state.nextRound
		approved := make(map[string]bool, len(req.Approvals))
		for _, item := range req.Approvals {
			approved[item.ID] = item.Approved
		}
		for _, call := range state.pending {
			content := `{"error":"user denied"}`
			if approved[call.ID] {
				usedTools = append(usedTools, call.Name)
				result, callErr := s.callTool(ctx, principal, call)
				if callErr != nil {
					content = fmt.Sprintf(`{"error":%q}`, callErr.Error())
				} else {
					content = result
				}
			}
			messages = append(messages, chatMessage{Role: "tool", Content: trimToolResult(call.Name, content), ToolCallID: call.ID})
		}
	} else {
		messages = toChatMessages(req.Messages)
		if len(messages) == 0 {
			return nil, response.ErrAIProviderInvalid("messages are required")
		}
		if err := s.appendDocuments(ctx, principal, req.DocumentIDs, messages); err != nil {
			return nil, err
		}
		if err := appendAttachments(messages, req.Attachments); err != nil {
			return nil, err
		}
		available := append([]tool.Tool{}, s.tools.List()...)
		available = append(available, askUserTool{})
		providerTools, toolNames, err = providerToolAliases(available)
		if err != nil {
			logging.ErrorS(err, "AI assistant tool registration is invalid")
			return nil, response.ErrAIBusy()
		}
		toolNamesForPrompt := make([]string, 0, len(providerTools))
		for _, item := range providerTools {
			toolNamesForPrompt = append(toolNamesForPrompt, item.Name())
		}
		system, err = renderSystemPrompt(lang, systemPromptData{Tools: toolNamesForPrompt})
		if err != nil {
			logging.ErrorS(err, "AI assistant prompt rendering failed")
			return nil, response.ErrAIBusy()
		}
		usedTools = make([]string, 0)
	}

	callSignatureCounts := make(map[string]int)
	for round := startRound; round < maxToolRounds; round++ {
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
		if call := findClarificationCall(result.ToolCalls, toolNames); call != nil {
			clarification, err := parseClarification(call.Arguments)
			if err != nil {
				logging.ErrorS(err, "AI assistant clarification is invalid")
				return nil, response.ErrAIProviderInvalid(err.Error())
			}
			return &ChatResponse{Reply: clarification.Question, Clarification: clarification, Tools: usedTools}, nil
		}
		signature := toolCallSignature(result.ToolCalls)
		if signature != "" {
			callSignatureCounts[signature]++
			if callSignatureCounts[signature] > 2 {
				logging.Warnf("AI assistant repeated tool calls: %v", toolCallNames(result.ToolCalls, toolNames))
				return s.finalAnswer(ctx, provider, system, messages, usedTools)
			}
		}
		messages = append(messages, chatMessage{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})
		risky := make([]toolCall, 0)
		for _, call := range result.ToolCalls {
			toolName := toolNames[call.Name]
			if toolName == "" {
				toolName = call.Name
			}
			item, ok := s.tools.Get(toolName)
			if ok && (item.Risk() == tool.RiskWrite || item.Risk() == tool.RiskAction) {
				risky = append(risky, toolCall{
					ID:          call.ID,
					Name:        toolName,
					Arguments:   call.Arguments,
					Risk:        item.Risk(),
					Description: item.Description(),
				})
				continue
			}
			usedTools = append(usedTools, toolName)
			content, callErr := s.callTool(ctx, principal, toolCall{ID: call.ID, Name: toolName, Arguments: call.Arguments})
			if callErr != nil {
				content = fmt.Sprintf(`{"error":%q}`, callErr.Error())
			}
			messages = append(messages, chatMessage{Role: "tool", Content: trimToolResult(call.Name, content), ToolCallID: call.ID})
		}
		if len(risky) > 0 {
			pendingID, err := s.savePending(&pendingTurn{
				ownerID:       principal.UserID,
				messages:      messages,
				pending:       risky,
				system:        system,
				providerTools: providerTools,
				toolNames:     toolNames,
				usedTools:     usedTools,
				nextRound:     round + 1,
				created:       time.Now(),
			})
			if err != nil {
				logging.ErrorS(err, "AI assistant pending tool state failed")
				return nil, response.ErrAIBusy()
			}
			return &ChatResponse{PendingID: pendingID, Pending: pendingCalls(risky)}, nil
		}
	}
	logging.Warnf("AI assistant tool loop exceeded: %v", usedTools)
	return s.finalAnswer(ctx, provider, system, messages, usedTools)
}

func pendingCalls(calls []toolCall) []PendingToolCall {
	out := make([]PendingToolCall, 0, len(calls))
	for _, call := range calls {
		risk := call.Risk
		if risk != tool.RiskWrite && risk != tool.RiskAction {
			risk = tool.RiskAction
		}
		out = append(out, PendingToolCall{
			ID:          call.ID,
			Name:        call.Name,
			Description: call.Description,
			Risk:        risk,
			Arguments:   call.Arguments,
		})
	}
	return out
}

func (s *Service) savePending(state *pendingTurn) (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	id := hex.EncodeToString(raw)
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pending == nil {
		s.pending = make(map[string]*pendingTurn)
	}
	if len(s.pending) >= 128 {
		for key, item := range s.pending {
			if time.Since(item.created) > pendingTTL {
				delete(s.pending, key)
			}
		}
		if len(s.pending) >= 128 {
			return "", fmt.Errorf("too many pending assistant confirmations")
		}
	}
	s.pending[id] = state
	return id, nil
}

func (s *Service) takePending(id, ownerID string) (*pendingTurn, error) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	state, ok := s.pending[id]
	if !ok {
		return nil, response.ErrAIProviderInvalid("pending confirmation is missing or expired")
	}
	delete(s.pending, id)
	if state.ownerID != ownerID || time.Since(state.created) > pendingTTL {
		return nil, response.ErrAIProviderInvalid("pending confirmation is missing or expired")
	}
	return state, nil
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

func trimToolResult(name string, value string) string {
	if len(value) <= maxToolResultLen {
		return value
	}
	logging.Warnf("AI tool result truncated: tool=%s bytes=%d", name, len(value))
	encoded, err := json.Marshal(map[string]any{
		"truncated":     true,
		"tool":          name,
		"originalBytes": len(value),
		"error":         "tool result is too large; narrow the request or use pagination",
	})
	if err != nil {
		return `{"truncated":true,"error":"tool result is too large"}`
	}
	return string(encoded)
}

func toolCallSignature(calls []toolCall) string {
	if len(calls) == 0 {
		return ""
	}
	parts := make([]string, 0, len(calls))
	for _, call := range calls {
		parts = append(parts, call.Name+"\x00"+string(call.Arguments))
	}
	return strings.Join(parts, "\x01")
}

func toolCallNames(calls []toolCall, names map[string]string) []string {
	out := make([]string, 0, len(calls))
	for _, call := range calls {
		name := names[call.Name]
		if name == "" {
			name = call.Name
		}
		out = append(out, name)
	}
	return out
}

func (s *Service) finalAnswer(ctx context.Context, provider provider, system string, messages []chatMessage, usedTools []string) (*ChatResponse, error) {
	instruction := "\n\n工具调用次数已达上限。不要继续调用工具，基于已有工具结果回答；如果信息不足，请明确说明缺少什么，并只问一个必要的问题。"
	result, err := provider.Chat(ctx, system+instruction, messages, nil)
	if err != nil {
		logging.ErrorS(err, "AI assistant final answer failed")
		return nil, response.ErrAIBusy()
	}
	reply := strings.TrimSpace(result.Content)
	if reply == "" {
		logging.ErrorS(fmt.Errorf("model returned an empty final response"), "AI assistant final answer failed")
		return nil, response.ErrAIBusy()
	}
	return &ChatResponse{Reply: reply, Tools: usedTools}, nil
}
