package opcua

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestBuildWriteRequest(t *testing.T) {
	request, err := buildWriteRequest([]collection.ActionValue{
		{Variable: &collection.Variable{Name: "enabled", DataType: "bool", Address: "ns=2;s=Enabled"}, Value: true},
		{Variable: &collection.Variable{Name: "pressure", DataType: "number", Address: "ns=2;s=Pressure"}, Value: float64(12.5)},
	})
	require.NoError(t, err)
	require.Len(t, request.NodesToWrite, 2)
	assert.Equal(t, ua.AttributeIDValue, request.NodesToWrite[0].AttributeID)
	assert.Equal(t, "ns=2;s=Enabled", request.NodesToWrite[0].NodeID.String())
	assert.Equal(t, true, request.NodesToWrite[0].Value.Value.Value())
	assert.Equal(t, float64(12.5), request.NodesToWrite[1].Value.Value.Value())
}

func TestBuildWriteRequestRejectsInvalidNodeID(t *testing.T) {
	_, err := buildWriteRequest([]collection.ActionValue{{
		Variable: &collection.Variable{Name: "bad", DataType: "number", Address: ""},
		Value:    float64(1),
	}})
	require.Error(t, err)
}
