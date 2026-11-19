package modbustcp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbus"
)

func TestDuplicateAddressesAllDecoded(t *testing.T) {
	variables := make([]*collection.Variable, 0, 100)
	for index := 0; index < 100; index++ {
		variables = append(variables, &collection.Variable{
			Name:       fmt.Sprintf("variable_%d", index),
			DataType:   "int16",
			Address:    "1!40001",
			AccessMode: collection.AccessModeRead,
		})
	}
	frames, err := modbus.BuildFrames(variables, 1)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	require.Len(t, frames[0].Variables, 100)

	values, err := modbus.DecodeFrame(frames[0], []byte{0x00, 0x0A}, "ABCD")
	require.NoError(t, err)
	require.Len(t, values, 100)

	meta := &collection.CollectionMeta{Variables: variables}
	meta.IndexCollection()
	for _, value := range values {
		meta.SetVariableValue(value.Variable.Name, value.Value)
	}
	for _, variable := range variables {
		value, ok := meta.GetVariableValue(variable.Name)
		require.True(t, ok)
		assert.Equal(t, int16(10), value)
	}
}

func TestCollectorCollectsDuplicateAddresses(t *testing.T) {
	server := newFakeModbusServer(t)
	server.SetRegister(0, 10)
	variables := make([]*collection.Variable, 0, 100)
	for index := 0; index < 100; index++ {
		variables = append(variables, &collection.Variable{
			Name:       fmt.Sprintf("variable_%d", index),
			DataType:   "int16",
			Address:    "1!40001",
			AccessMode: collection.AccessModeRead,
		})
	}
	created, err := NewCollection(&CreateReq{
		Name:             "duplicate-address",
		CollectorCycleMs: 20,
		ProtocolConfig: ProtocolConfig{
			Location:        server.Addr(),
			MemoryLayout:    "ABCD",
			PositionAddress: 1,
			Concurrency:     1,
		},
		Variables: variables,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = created.Stop(context.Background()) })
	require.NoError(t, created.Connect(context.Background()))
	require.NoError(t, created.Start(context.Background()))

	require.Eventually(t, func() bool {
		for _, variable := range variables {
			value, ok := created.GetVariableValue(variable.Name)
			if !ok || value != int16(10) {
				return false
			}
		}
		return true
	}, 2*time.Second, 20*time.Millisecond)
}
