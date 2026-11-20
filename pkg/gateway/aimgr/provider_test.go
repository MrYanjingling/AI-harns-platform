package aimgr

import "testing"

func TestResolveEndpointURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		protocol string
		want     string
	}{
		{"openai full", "https://api.openai.com/v1/chat/completions", ProtocolOpenAICompatible, "https://api.openai.com/v1/chat/completions"},
		{"deepseek root", "https://api.deepseek.com", ProtocolOpenAICompatible, "https://api.deepseek.com/v1/chat/completions"},
		{"openai v1", "https://api.openai.com/v1", ProtocolOpenAICompatible, "https://api.openai.com/v1/chat/completions"},
		{"anthropic root", "https://api.anthropic.com", ProtocolAnthropicMessages, "https://api.anthropic.com/v1/messages"},
		{"anthropic v1", "https://api.anthropic.com/v1", ProtocolAnthropicMessages, "https://api.anthropic.com/v1/messages"},
		{"anthropic full", "https://api.anthropic.com/v1/messages", ProtocolAnthropicMessages, "https://api.anthropic.com/v1/messages"},
		{"openai trailing slash", "https://api.openai.com/v1/chat/completions/", ProtocolOpenAICompatible, "https://api.openai.com/v1/chat/completions"},
		{"anthropic trailing slash", "https://api.anthropic.com/v1/messages/", ProtocolAnthropicMessages, "https://api.anthropic.com/v1/messages"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolveEndpointURL(test.endpoint, test.protocol); got != test.want {
				t.Fatalf("resolveEndpointURL() = %q, want %q", got, test.want)
			}
		})
	}
}
