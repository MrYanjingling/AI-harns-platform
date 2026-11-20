package aimgr

import "time"

const (
	ProtocolOpenAICompatible  = "openai-compatible"
	ProtocolAnthropicMessages = "anthropic-messages"
)

// Settings holds the single AI model configuration, MCP token and service switches.
type Settings struct {
	AssistantEnabled bool
	MCPEnabled       bool
	Name             string
	Remark           string
	Model            string
	EndpointURL      string
	APIKey           string
	MCPToken         string
	UpdatedTime      time.Time
}
