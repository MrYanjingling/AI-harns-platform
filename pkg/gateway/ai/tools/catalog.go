package tools

import "harns-platform/pkg/gateway/ai/tool"

// NewRegistry builds the assistant tool registry.
func NewRegistry(deps Deps) (*tool.Registry, error) {
	items := append(gatewayTools(deps), writeTools(deps)...)
	items = append(items, adminTools(deps)...)
	return tool.NewRegistry(items...)
}
