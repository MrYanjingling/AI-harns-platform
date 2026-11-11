package opcua

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestBuildBatchesExcludesCalculatedAndWriteOnly(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", DataType: "number", Address: "ns=2;i=1001", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "number", Address: "ns=2;s=Tag", AccessMode: collection.AccessModeRead},
		{Name: "w", DataType: "number", Address: "ns=2;i=1002", AccessMode: collection.AccessModeWrite},
		{Name: "c", DataType: "number", IsCalculated: true, Expression: "a + b", AccessMode: collection.AccessModeRead},
	}
	batches, err := buildBatches(variables)
	require.NoError(t, err)
	require.Len(t, batches, 1)
	require.Len(t, batches[0].variables, 2)
	assert.Equal(t, "ns=2;i=1001", batches[0].request.NodesToRead[0].NodeID.String())
}

func TestRedactConfig(t *testing.T) {
	handler := &CreateHandler{}
	raw := json.RawMessage(`{"location":"opc.tcp://host","port":4840,"username":"u","password":"secret","tcpCount":1}`)
	redacted := handler.RedactConfig(raw)
	assert.Contains(t, string(redacted), `"password":"***"`)
	assert.NotContains(t, string(redacted), "secret")
}

func TestEndpoint(t *testing.T) {
	item := &OpcUaCollection{Location: "opc.tcp://host", Port: 4840}
	assert.Equal(t, "opc.tcp://host:4840", item.endpoint())
}
