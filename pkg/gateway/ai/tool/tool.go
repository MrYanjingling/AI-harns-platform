// Package tool defines the gateway AI tool registry.
package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// Risk describes how dangerous a tool is for the assistant UI and MCP clients.
type Risk string

const (
	RiskRead   Risk = "read"
	RiskWrite  Risk = "write"
	RiskAction Risk = "action"
)

// Principal is the authenticated actor executing a tool.
type Principal struct {
	UserID   string
	Username string
	IsAdmin  bool
	Source   string
}

// Tool is one callable gateway capability.
type Tool interface {
	Name() string
	Description() string
	InputSchema() json.RawMessage
	Risk() Risk
	Execute(ctx context.Context, p Principal, input json.RawMessage) (any, error)
}

// Function adapts a function into a Tool.
type Function struct {
	ToolName        string
	ToolDescription string
	Schema          json.RawMessage
	RiskLevel       Risk
	Handler         func(ctx context.Context, p Principal, input json.RawMessage) (any, error)
}

func (f *Function) Name() string                 { return f.ToolName }
func (f *Function) Description() string          { return f.ToolDescription }
func (f *Function) InputSchema() json.RawMessage { return f.Schema }
func (f *Function) Risk() Risk                   { return f.RiskLevel }

func (f *Function) Execute(ctx context.Context, p Principal, input json.RawMessage) (any, error) {
	if f == nil || f.Handler == nil {
		return nil, fmt.Errorf("tool %q has no handler", f.ToolName)
	}
	return f.Handler(ctx, p, input)
}

// Registry stores tools by name.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an immutable registry from the given tools.
func NewRegistry(tools ...Tool) (*Registry, error) {
	r := &Registry{tools: make(map[string]Tool, len(tools))}
	for _, item := range tools {
		if item == nil {
			return nil, fmt.Errorf("tool is nil")
		}
		name := item.Name()
		if name == "" {
			return nil, fmt.Errorf("tool name is empty")
		}
		if _, exists := r.tools[name]; exists {
			return nil, fmt.Errorf("duplicate tool %q", name)
		}
		schema := bytes.TrimSpace(item.InputSchema())
		if len(schema) > 0 {
			if !json.Valid(schema) || schema[0] != '{' {
				return nil, fmt.Errorf("tool %q input schema must be a JSON object", name)
			}
		}
		r.tools[name] = item
	}
	return r, nil
}

// List returns tools in registration map order.
func (r *Registry) List() []Tool {
	if r == nil {
		return nil
	}
	out := make([]Tool, 0, len(r.tools))
	for _, item := range r.tools {
		out = append(out, item)
	}
	return out
}

// Get returns one tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	item, ok := r.tools[name]
	return item, ok
}

// Call executes one tool.
func (r *Registry) Call(ctx context.Context, p Principal, name string, input json.RawMessage) (any, error) {
	item, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	if len(input) == 0 {
		input = json.RawMessage(`{}`)
	}
	if !json.Valid(input) {
		return nil, fmt.Errorf("tool %q input is not valid JSON", name)
	}
	return item.Execute(ctx, p, input)
}

// DecodeInput decodes and validates a JSON tool input.
func DecodeInput[T any](input json.RawMessage) (*T, error) {
	var value T
	if len(input) == 0 {
		input = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(input, &value); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	return &value, nil
}
