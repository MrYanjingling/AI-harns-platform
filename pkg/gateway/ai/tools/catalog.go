package tools

import "harns-platform/pkg/gateway/ai/tool"

// NewRegistry builds the read-only assistant tool registry.
func NewRegistry(deps Deps) (*tool.Registry, error) {
	items := gatewayTools(deps)
	return tool.NewRegistry(items...)
}
