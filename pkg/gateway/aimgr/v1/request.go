package v1

// UpdateSettingsReq updates AI service switches.
type UpdateSettingsReq struct {
	AssistantEnabled bool `json:"assistantEnabled"`
	MCPEnabled       bool `json:"mcpEnabled"`
}

// UpdateModelReq updates the single AI model configuration.
type UpdateModelReq struct {
	Name        string `json:"name"`
	Remark      string `json:"remark"`
	Model       string `json:"model"`
	APIKey      string `json:"apiKey"`
	EndpointURL string `json:"endpointUrl"`
}
