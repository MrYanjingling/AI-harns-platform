package assistant

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/ai/tool"
)

func testTool(name string) *tool.Function {
	return &tool.Function{
		ToolName:        name,
		ToolDescription: name,
		Schema:          json.RawMessage(`{"type":"object"}`),
		RiskLevel:       tool.RiskRead,
		Handler: func(context.Context, tool.Principal, json.RawMessage) (any, error) {
			return nil, nil
		},
	}
}

func TestProviderToolAliases(t *testing.T) {
	aliases, names, err := providerToolAliases([]tool.Tool{
		testTool("collections.list"),
		testTool("collections.get"),
	})
	require.NoError(t, err)
	require.Len(t, aliases, 2)
	assert.Equal(t, "collections_list", aliases[0].Name())
	assert.Equal(t, "collections.list", names["collections_list"])
	assert.Equal(t, "collections_get", aliases[1].Name())
	assert.Equal(t, "collections.get", names["collections_get"])
}

func TestProviderToolAliasConflict(t *testing.T) {
	_, _, err := providerToolAliases([]tool.Tool{
		testTool("collections.list"),
		testTool("collections_list"),
	})
	require.Error(t, err)
}
