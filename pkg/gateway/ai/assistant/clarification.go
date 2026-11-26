package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/ai/tool"
)

const askUserToolName = "assistant_ask_user"

var askUserToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "question": {
      "type": "string",
      "description": "The single question to ask the user."
    },
    "options": {
      "type": "array",
      "description": "Two to six concise answer options.",
      "items": { "type": "string" },
      "minItems": 2,
      "maxItems": 6
    },
    "allowCustom": {
      "type": "boolean",
      "description": "Whether the user may type a custom answer instead of selecting an option.",
      "default": true
    }
  },
  "required": ["question", "options"]
}`)

// Clarification is one question presented to the user with selectable options.
type Clarification struct {
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	AllowCustom bool     `json:"allowCustom"`
}

type askUserInput struct {
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	AllowCustom *bool    `json:"allowCustom"`
}

type askUserTool struct{}

func (askUserTool) Name() string { return askUserToolName }
func (askUserTool) Description() string {
	return "Ask the user exactly one clarification question with 2 to 6 options. Ask only one question per turn."
}
func (askUserTool) InputSchema() json.RawMessage { return askUserToolSchema }
func (askUserTool) Risk() tool.Risk              { return tool.RiskRead }

func (askUserTool) Execute(_ context.Context, _ tool.Principal, _ json.RawMessage) (any, error) {
	return nil, fmt.Errorf("ask_user is handled by the assistant service")
}

func parseClarification(input json.RawMessage) (*Clarification, error) {
	var value askUserInput
	if err := json.Unmarshal(input, &value); err != nil {
		return nil, fmt.Errorf("decode clarification: %w", err)
	}
	question := strings.TrimSpace(value.Question)
	if question == "" {
		return nil, fmt.Errorf("clarification question is required")
	}
	options := make([]string, 0, len(value.Options))
	seen := make(map[string]bool, len(value.Options))
	for _, item := range value.Options {
		option := strings.TrimSpace(item)
		if option == "" || seen[option] {
			continue
		}
		seen[option] = true
		options = append(options, option)
	}
	if len(options) < 2 || len(options) > 6 {
		return nil, fmt.Errorf("clarification requires 2 to 6 options")
	}
	allowCustom := true
	if value.AllowCustom != nil {
		allowCustom = *value.AllowCustom
	}
	return &Clarification{Question: question, Options: options, AllowCustom: allowCustom}, nil
}

func findClarificationCall(calls []toolCall, names map[string]string) *toolCall {
	for index := range calls {
		name := names[calls[index].Name]
		if name == "" {
			name = calls[index].Name
		}
		if name == askUserToolName {
			return &calls[index]
		}
	}
	return nil
}
