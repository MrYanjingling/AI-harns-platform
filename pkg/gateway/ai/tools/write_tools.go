package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/collection"
	collectionv1 "harns-platform/pkg/gateway/collectionmgr/v1"
	driverv1 "harns-platform/pkg/gateway/drivermgr/v1"
	pushv1 "harns-platform/pkg/gateway/pushmgr/v1"
)

const bodySchema = `{
  "type":"object",
  "properties":{"body":{"type":"object"}},
  "required":["body"]
}`

func writeTools(deps Deps) []tool.Tool {
	out := make([]tool.Tool, 0)
	if deps.Collections != nil {
		out = append(out,
			bodyTool("collections.create", "Create a collection from a full collection JSON body. Use collections.get and drivers.get_schema first.", tool.RiskWrite, func(ctx context.Context, _ tool.Principal, body json.RawMessage) (any, error) {
				return deps.Collections.Create(ctx, body)
			}),
			bodyWithIDSchema("collections.update", "Update collection name, cycle, and protocol config. Use collections.get first.", tool.RiskWrite, func(ctx context.Context, _ tool.Principal, id string, body json.RawMessage) (any, error) {
				return deps.Collections.Update(ctx, id, body)
			}),
			idTool("collections.delete", "Delete one collection and its variables.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
				return deps.Collections.Delete(ctx, id)
			}),
			idTool("collections.start", "Start one collection.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
				return deps.Collections.Start(ctx, id)
			}),
			idTool("collections.stop", "Stop one collection.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
				return deps.Collections.Stop(ctx, id)
			}),
			idTool("collections.restart", "Restart one collection.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
				if err := deps.Collections.Restart(ctx, id); err != nil {
					return nil, err
				}
				return map[string]any{"restarted": true}, nil
			}),
			&tool.Function{
				ToolName:        "collections.set_enable",
				ToolDescription: "Enable or disable auto-start for one collection.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"enable":{"type":"boolean"}},"required":["id","enable"]}`),
				RiskLevel:       tool.RiskAction,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						ID     string `json:"id"`
						Enable bool   `json:"enable"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Collections.SetEnable(ctx, input.ID, input.Enable)
				},
			},
			&tool.Function{
				ToolName:        "collections.add_variables",
				ToolDescription: "Add variables to one collection. Requires collectionId and variables in the same shape as the collection API.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"collectionId":{"type":"string"},"variables":{"type":"array","items":{"type":"object"}}},"required":["collectionId","variables"]}`),
				RiskLevel:       tool.RiskWrite,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						CollectionID string                 `json:"collectionId"`
						Variables    []*collection.Variable `json:"variables"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Collections.AddVariables(ctx, input.CollectionID, input.Variables)
				},
			},
			&tool.Function{
				ToolName:        "collections.update_variables",
				ToolDescription: "Update variables in one collection. Names are immutable.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"collectionId":{"type":"string"},"variables":{"type":"array","items":{"type":"object"}}},"required":["collectionId","variables"]}`),
				RiskLevel:       tool.RiskWrite,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						CollectionID string                 `json:"collectionId"`
						Variables    []*collection.Variable `json:"variables"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Collections.UpdateVariables(ctx, input.CollectionID, input.Variables)
				},
			},
			&tool.Function{
				ToolName:        "collections.delete_variables",
				ToolDescription: "Delete variables from one collection by name.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"collectionId":{"type":"string"},"names":{"type":"array","items":{"type":"string"}}},"required":["collectionId","names"]}`),
				RiskLevel:       tool.RiskAction,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						CollectionID string   `json:"collectionId"`
						Names        []string `json:"names"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Collections.DeleteVariables(ctx, input.CollectionID, input.Names)
				},
			},
			&tool.Function{
				ToolName:        "collections.action",
				ToolDescription: "Write values to one collection. Requires collectionId and actions [{name,value}].",
				Schema:          json.RawMessage(`{"type":"object","properties":{"collectionId":{"type":"string"},"actions":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"value":{"type":"string"}},"required":["name","value"]}}},"required":["collectionId","actions"]}`),
				RiskLevel:       tool.RiskAction,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						CollectionID string                    `json:"collectionId"`
						Actions      []collectionv1.ActionItem `json:"actions"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Collections.DeliverAction(ctx, input.CollectionID, collectionv1.ActionReq{Actions: input.Actions})
				},
			},
		)
	}
	if deps.Drivers != nil {
		out = append(out,
			bodyTool("drivers.create", "Create a custom driver. Body uses the driver create API shape.", tool.RiskWrite, func(ctx context.Context, _ tool.Principal, body json.RawMessage) (any, error) {
				var req driverv1.CreateDriverReq
				if err := json.Unmarshal(body, &req); err != nil {
					return nil, err
				}
				return deps.Drivers.Create(ctx, &req)
			}),
			bodyWithIDSchema("drivers.update", "Update a custom driver. Body uses the driver update API shape.", tool.RiskWrite, func(ctx context.Context, _ tool.Principal, id string, body json.RawMessage) (any, error) {
				var req driverv1.UpdateDriverReq
				if err := json.Unmarshal(body, &req); err != nil {
					return nil, err
				}
				return deps.Drivers.Update(ctx, id, &req)
			}),
			idTool("drivers.delete", "Delete a custom driver.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
				return nil, deps.Drivers.Delete(ctx, id)
			}),
		)
	}
	if deps.Push != nil {
		out = append(out, pushWriteTools(deps)...)
	}
	return out
}

func bodyTool(name, description string, risk tool.Risk, handler func(context.Context, tool.Principal, json.RawMessage) (any, error)) tool.Tool {
	return &tool.Function{
		ToolName: name, ToolDescription: description, Schema: json.RawMessage(bodySchema), RiskLevel: risk,
		Handler: func(ctx context.Context, p tool.Principal, raw json.RawMessage) (any, error) {
			input, err := tool.DecodeInput[struct {
				Body json.RawMessage `json:"body"`
			}](raw)
			if err != nil {
				return nil, err
			}
			body := bytes.TrimSpace(input.Body)
			if len(body) == 0 || body[0] != '{' {
				return nil, fmt.Errorf("body must be a JSON object")
			}
			return handler(ctx, p, body)
		},
	}
}

func bodyWithIDSchema(name, description string, risk tool.Risk, handler func(context.Context, tool.Principal, string, json.RawMessage) (any, error)) tool.Tool {
	return &tool.Function{
		ToolName: name, ToolDescription: description,
		Schema:    json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"body":{"type":"object"}},"required":["id","body"]}`),
		RiskLevel: risk,
		Handler: func(ctx context.Context, p tool.Principal, raw json.RawMessage) (any, error) {
			input, err := tool.DecodeInput[struct {
				ID   string          `json:"id"`
				Body json.RawMessage `json:"body"`
			}](raw)
			if err != nil {
				return nil, err
			}
			body := bytes.TrimSpace(input.Body)
			if input.ID == "" || len(body) == 0 || body[0] != '{' {
				return nil, fmt.Errorf("id and body are required")
			}
			return handler(ctx, p, input.ID, body)
		},
	}
}

func idTool(name, description string, risk tool.Risk, handler func(context.Context, tool.Principal, string) (any, error)) tool.Tool {
	return &tool.Function{
		ToolName: name, ToolDescription: description,
		Schema:    json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		RiskLevel: risk,
		Handler: func(ctx context.Context, p tool.Principal, raw json.RawMessage) (any, error) {
			input, err := tool.DecodeInput[struct {
				ID string `json:"id"`
			}](raw)
			if err != nil {
				return nil, err
			}
			if input.ID == "" {
				return nil, fmt.Errorf("id is required")
			}
			return handler(ctx, p, input.ID)
		},
	}
}

func pushWriteTools(deps Deps) []tool.Tool {
	return []tool.Tool{
		bodyTool("push.connectors.create", "Create a push connector. Body uses the connector create API shape.", tool.RiskWrite, func(ctx context.Context, _ tool.Principal, body json.RawMessage) (any, error) {
			var req pushv1.CreateConnectorReq
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, err
			}
			return deps.Push.CreateConnector(ctx, &req)
		}),
		bodyWithIDSchema("push.connectors.update", "Update a push connector.", tool.RiskWrite, func(ctx context.Context, _ tool.Principal, id string, body json.RawMessage) (any, error) {
			var req pushv1.UpdateConnectorReq
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, err
			}
			return deps.Push.UpdateConnector(ctx, id, &req)
		}),
		&tool.Function{
			ToolName: "push.connectors.delete", ToolDescription: "Delete a push connector. Set cascade to also remove its groups.",
			Schema:    json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"cascade":{"type":"boolean"}},"required":["id"]}`),
			RiskLevel: tool.RiskAction,
			Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
				input, err := tool.DecodeInput[struct {
					ID      string `json:"id"`
					Cascade bool   `json:"cascade"`
				}](raw)
				if err != nil {
					return nil, err
				}
				return nil, deps.Push.DeleteConnector(ctx, input.ID, input.Cascade)
			},
		},
		idTool("push.connectors.start", "Start a push connector.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
			return deps.Push.StartConnector(ctx, id)
		}),
		idTool("push.connectors.stop", "Stop a push connector.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
			return deps.Push.StopConnector(ctx, id)
		}),
		idTool("push.connectors.restart", "Restart a push connector.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
			return deps.Push.RestartConnector(ctx, id)
		}),
		bodyTool("push.groups.create", "Create a push group. Body uses the push group create API shape.", tool.RiskWrite, func(ctx context.Context, _ tool.Principal, body json.RawMessage) (any, error) {
			var req pushv1.CreatePushGroupReq
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, err
			}
			return deps.Push.CreateGroup(ctx, &req)
		}),
		bodyWithIDSchema("push.groups.update", "Update a push group.", tool.RiskWrite, func(ctx context.Context, _ tool.Principal, id string, body json.RawMessage) (any, error) {
			var req pushv1.UpdatePushGroupReq
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, err
			}
			return deps.Push.UpdateGroup(ctx, id, &req)
		}),
		idTool("push.groups.delete", "Delete a push group.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
			return nil, deps.Push.DeleteGroup(ctx, id)
		}),
		idTool("push.groups.start", "Start a push group.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
			return deps.Push.StartGroup(ctx, id)
		}),
		idTool("push.groups.stop", "Stop a push group.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) {
			return deps.Push.StopGroup(ctx, id)
		}),
		idTool("push.groups.restart", "Restart a push group.", tool.RiskAction, func(ctx context.Context, _ tool.Principal, id string) (any, error) { return deps.Push.RestartGroup(id) }),
	}
}
