package tools

import (
	"context"
	"encoding/json"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/collectionmgr/v1"
)

const pageSchema = `{
  "type":"object",
  "properties":{
    "page":{"type":"integer","minimum":1,"default":1},
    "pageSize":{"type":"integer","minimum":1,"maximum":100,"default":20},
    "keyword":{"type":"string"}
  }
}`

type pageInput struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Keyword  string `json:"keyword"`
}

func (p pageInput) query() apis.PageQuery {
	return apis.PageQuery{Page: p.Page, PageSize: p.PageSize}
}

func gatewayTools(deps Deps) []tool.Tool {
	out := make([]tool.Tool, 0)
	if deps.Gateway != nil {
		out = append(out,
			&tool.Function{
				ToolName:        "gateway.get_meta",
				ToolDescription: "Get gateway identity and metadata.",
				Schema:          json.RawMessage(`{"type":"object","properties":{}}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, _ json.RawMessage) (any, error) {
					return deps.Gateway.GetGatewayMetaVo()
				},
			},
			&tool.Function{
				ToolName:        "gateway.get_system_info",
				ToolDescription: "Get gateway host and runtime information.",
				Schema:          json.RawMessage(`{"type":"object","properties":{}}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, _ json.RawMessage) (any, error) {
					return deps.Gateway.GetSystemInfoMeta(ctx), nil
				},
			},
		)
	}
	if deps.Drivers != nil {
		out = append(out,
			&tool.Function{
				ToolName:        "drivers.list",
				ToolDescription: "List built-in and custom drivers grouped by driver type.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"driverType":{"type":"string"}}}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						DriverType string `json:"driverType"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Drivers.ListGrouped(ctx, input.DriverType)
				},
			},
			&tool.Function{
				ToolName:        "drivers.get",
				ToolDescription: "Get one driver by ID.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						ID string `json:"id"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Drivers.Get(ctx, input.ID)
				},
			},
			&tool.Function{
				ToolName:        "drivers.get_schema",
				ToolDescription: "Get the protocol schema for one driver model.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"driverModel":{"type":"string"}},"required":["driverModel"]}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						DriverModel string `json:"driverModel"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Drivers.GetSchemaByModel(ctx, input.DriverModel)
				},
			},
		)
	}
	if deps.Collections != nil {
		out = append(out,
			&tool.Function{
				ToolName:        "collections.list",
				ToolDescription: "List collection instances with status and last error. variableCount includes collected and calculated variables; collectedVariableCount excludes calculated variables.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"page":{"type":"integer","minimum":1},"pageSize":{"type":"integer","minimum":1,"maximum":100},"keyword":{"type":"string"},"driverType":{"type":"string"},"driverModel":{"type":"string"}}}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					var input struct {
						Page        int    `json:"page"`
						PageSize    int    `json:"pageSize"`
						Keyword     string `json:"keyword"`
						DriverType  string `json:"driverType"`
						DriverModel string `json:"driverModel"`
					}
					if err := json.Unmarshal(raw, &input); err != nil {
						return nil, err
					}
					return deps.Collections.List(ctx, v1.CollectionQuery{
						PageQuery:   apis.PageQuery{Page: input.Page, PageSize: input.PageSize},
						Keyword:     input.Keyword,
						DriverType:  input.DriverType,
						DriverModel: input.DriverModel,
					})
				},
			},
			&tool.Function{
				ToolName:        "collections.get",
				ToolDescription: "Get one collection detail by ID.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					input, err := tool.DecodeInput[struct {
						ID string `json:"id"`
					}](raw)
					if err != nil {
						return nil, err
					}
					return deps.Collections.GetDetail(ctx, input.ID)
				},
			},
			&tool.Function{
				ToolName:        "collections.get_variables",
				ToolDescription: "List variables and current values for one collection. isCalculated=true means a calculated variable that is not collected from the device.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"collectionId":{"type":"string"},"page":{"type":"integer","minimum":1},"pageSize":{"type":"integer","minimum":1,"maximum":100},"keyword":{"type":"string"}},"required":["collectionId"]}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					var input struct {
						CollectionID string `json:"collectionId"`
						Page         int    `json:"page"`
						PageSize     int    `json:"pageSize"`
						Keyword      string `json:"keyword"`
					}
					if err := json.Unmarshal(raw, &input); err != nil {
						return nil, err
					}
					return deps.Collections.ListVariables(ctx, input.CollectionID, v1.VariableQuery{
						PageQuery: apis.PageQuery{Page: input.Page, PageSize: input.PageSize},
						Keyword:   input.Keyword,
					})
				},
			},
		)
	}
	if deps.Push != nil {
		out = append(out,
			&tool.Function{
				ToolName:        "push.connectors.list",
				ToolDescription: "List push connectors. groupCount is only the number of groups; use push.groups.list with connectorId to read each group dataTopic and commandTopic.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"page":{"type":"integer","minimum":1},"pageSize":{"type":"integer","minimum":1,"maximum":100},"keyword":{"type":"string"},"type":{"type":"string"}}}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					var input struct {
						Page     int    `json:"page"`
						PageSize int    `json:"pageSize"`
						Keyword  string `json:"keyword"`
						Type     string `json:"type"`
					}
					if err := json.Unmarshal(raw, &input); err != nil {
						return nil, err
					}
					return deps.Push.ListConnectors(ctx, apis.PageQuery{Page: input.Page, PageSize: input.PageSize}, input.Keyword, input.Type)
				},
			},
			&tool.Function{
				ToolName:        "push.groups.list",
				ToolDescription: "List push groups with name, dataTopic, commandTopic, connectorId, status and push interval.",
				Schema:          json.RawMessage(`{"type":"object","properties":{"page":{"type":"integer","minimum":1},"pageSize":{"type":"integer","minimum":1,"maximum":100},"keyword":{"type":"string"},"connectorId":{"type":"string"}}}`),
				RiskLevel:       tool.RiskRead,
				Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
					var input struct {
						Page        int    `json:"page"`
						PageSize    int    `json:"pageSize"`
						Keyword     string `json:"keyword"`
						ConnectorID string `json:"connectorId"`
					}
					if err := json.Unmarshal(raw, &input); err != nil {
						return nil, err
					}
					return deps.Push.ListGroups(ctx, apis.PageQuery{Page: input.Page, PageSize: input.PageSize}, input.Keyword, input.ConnectorID)
				},
			},
		)
	}
	if deps.Users != nil {
		out = append(out, &tool.Function{
			ToolName:        "users.list",
			ToolDescription: "List gateway users.",
			Schema:          json.RawMessage(pageSchema),
			RiskLevel:       tool.RiskRead,
			Handler: func(ctx context.Context, _ tool.Principal, raw json.RawMessage) (any, error) {
				input, err := tool.DecodeInput[pageInput](raw)
				if err != nil {
					return nil, err
				}
				return deps.Users.List(ctx, input.query(), input.Keyword)
			},
		})
	}
	if deps.License != nil {
		out = append(out, &tool.Function{
			ToolName:        "license.get_status",
			ToolDescription: "Get current license status and limits.",
			Schema:          json.RawMessage(`{"type":"object","properties":{}}`),
			RiskLevel:       tool.RiskRead,
			Handler: func(ctx context.Context, _ tool.Principal, _ json.RawMessage) (any, error) {
				return deps.License.GetStatus(), nil
			},
		})
	}
	return out
}
