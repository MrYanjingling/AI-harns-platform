package v1

import "time"

// SettingsVo is the AI settings response.
type SettingsVo struct {
	AssistantEnabled bool      `json:"assistantEnabled"`
	MCPEnabled       bool      `json:"mcpEnabled"`
	Name             string    `json:"name"`
	Remark           string    `json:"remark"`
	Model            string    `json:"model"`
	EndpointURL      string    `json:"endpointUrl"`
	APIKey           string    `json:"apiKey"`
	HasAPIKey        bool      `json:"hasApiKey"`
	MCPToken         string    `json:"mcpToken"`
	UpdatedTime      time.Time `json:"updatedTime"`
}
