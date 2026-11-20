package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTool(name string, risk Risk, fn func(context.Context, Principal, json.RawMessage) (any, error)) *Function {
	return &Function{
		ToolName:        name,
		ToolDescription: name,
		Schema:          json.RawMessage(`{"type":"object"}`),
		RiskLevel:       risk,
		Handler:         fn,
	}
}

func TestRegistryRejectsDuplicatesAndInvalidSchema(t *testing.T) {
	_, err := NewRegistry(
		testTool("collections.list", RiskRead, func(context.Context, Principal, json.RawMessage) (any, error) { return nil, nil }),
		testTool("collections.list", RiskRead, func(context.Context, Principal, json.RawMessage) (any, error) { return nil, nil }),
	)
	require.Error(t, err)

	bad := testTool("bad", RiskRead, func(context.Context, Principal, json.RawMessage) (any, error) { return nil, nil })
	bad.Schema = json.RawMessage(`[]`)
	_, err = NewRegistry(bad)
	require.Error(t, err)
}

func TestRegistryCall(t *testing.T) {
	registry, err := NewRegistry(testTool("echo", RiskRead, func(_ context.Context, _ Principal, input json.RawMessage) (any, error) {
		var value struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(input, &value); err != nil {
			return nil, err
		}
		return value.Text, nil
	}))
	require.NoError(t, err)

	value, err := registry.Call(context.Background(), Principal{UserID: "u1"}, "echo", json.RawMessage(`{"text":"hi"}`))
	require.NoError(t, err)
	assert.Equal(t, "hi", value)

	_, err = registry.Call(context.Background(), Principal{}, "missing", nil)
	require.Error(t, err)

	_, err = NewRegistry(testTool("boom", RiskRead, func(context.Context, Principal, json.RawMessage) (any, error) {
		return nil, errors.New("boom")
	}))
	require.NoError(t, err)
}
