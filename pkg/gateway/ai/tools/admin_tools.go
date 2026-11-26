package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"harns-platform/pkg/gateway/ai/tool"
	userv1 "harns-platform/pkg/gateway/usermgr/v1"
)

func adminTools(deps Deps) []tool.Tool {
	out := make([]tool.Tool, 0)
	if deps.Users != nil {
		out = append(out,
			bodyTool("users.create", "Create a gateway user. Admin only.", tool.RiskWrite, func(ctx context.Context, p tool.Principal, body json.RawMessage) (any, error) {
				if err := requireAdmin(p); err != nil {
					return nil, err
				}
				var req userv1.CreateUserReq
				if err := json.Unmarshal(body, &req); err != nil {
					return nil, err
				}
				return deps.Users.Create(ctx, &req)
			}),
			bodyWithIDSchema("users.update", "Update a gateway user. Admin only.", tool.RiskWrite, func(ctx context.Context, p tool.Principal, id string, body json.RawMessage) (any, error) {
				if err := requireAdmin(p); err != nil {
					return nil, err
				}
				var req userv1.UpdateUserReq
				if err := json.Unmarshal(body, &req); err != nil {
					return nil, err
				}
				return deps.Users.Update(ctx, id, &req)
			}),
			idTool("users.delete", "Delete a gateway user. Admin only.", tool.RiskAction, func(ctx context.Context, p tool.Principal, id string) (any, error) {
				if err := requireAdmin(p); err != nil {
					return nil, err
				}
				return nil, deps.Users.Delete(ctx, id)
			}),
			idTool("users.reset_password", "Reset a gateway user's password. Admin only.", tool.RiskAction, func(ctx context.Context, p tool.Principal, id string) (any, error) {
				if err := requireAdmin(p); err != nil {
					return nil, err
				}
				return deps.Users.ResetPassword(ctx, id)
			}),
		)
	}
	if deps.License != nil {
		out = append(out, &tool.Function{
			ToolName:        "license.import",
			ToolDescription: "Import a signed gateway license. Admin only.",
			Schema:          json.RawMessage(`{"type":"object","properties":{"license":{"type":"string"}},"required":["license"]}`),
			RiskLevel:       tool.RiskAction,
			Handler: func(ctx context.Context, p tool.Principal, raw json.RawMessage) (any, error) {
				if err := requireAdmin(p); err != nil {
					return nil, err
				}
				input, err := tool.DecodeInput[struct {
					License string `json:"license"`
				}](raw)
				if err != nil {
					return nil, err
				}
				return deps.License.Import(ctx, input.License)
			},
		})
	}
	return out
}

func requireAdmin(p tool.Principal) error {
	if !p.IsAdmin {
		return fmt.Errorf("admin permission is required")
	}
	return nil
}
