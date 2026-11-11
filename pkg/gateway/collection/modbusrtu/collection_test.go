package modbusrtu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

func TestNewCollectionDefaults(t *testing.T) {
	created, err := NewCollection(&CreateReq{
		Name:             "rtu",
		CollectorCycleMs: 1000,
		ProtocolConfig: ProtocolConfig{
			Location:     "/dev/ttyS0",
			MemoryLayout: "ABCD",
		},
		Variables: []*collection.Variable{{
			Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead,
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, 9600, created.BaudRate)
	assert.Equal(t, 8, created.DataBits)
	assert.Equal(t, "noParity", created.Parity)
	assert.Equal(t, "1", created.StopBits)
}
