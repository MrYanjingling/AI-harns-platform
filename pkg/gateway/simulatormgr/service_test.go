package simulatormgr

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/simonvetter/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	return port
}

func openService(t *testing.T, svc *Service) int {
	t.Helper()
	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		port := freePort(t)
		if _, err := svc.Open(context.Background(), port); err == nil {
			return port
		} else {
			lastErr = err
		}
	}
	t.Fatalf("failed to open simulator: %v", lastErr)
	return 0
}

func TestOpenCloseRetainsPointsAndRegisters(t *testing.T) {
	svc := NewService()
	openService(t, svc)

	_, err := svc.AddPoint(1, "", "", DataTypeInt16, ModeIncrement)
	require.NoError(t, err)
	svc.simulateOnce()
	state := svc.State()
	require.Len(t, state.Points, 1)
	assert.Equal(t, uint16(1), state.Points[0].Value)

	closed, err := svc.Close(context.Background())
	require.NoError(t, err)
	assert.False(t, closed.Running)
	require.Len(t, closed.Points, 1)

	openService(t, svc)
	reopened := svc.State()
	assert.True(t, reopened.Running)
	require.Len(t, reopened.Points, 1)
	assert.Equal(t, uint16(1), reopened.Points[0].Value)
	_, _ = svc.Close(context.Background())
}

func TestPointModesAndDefaultRegister(t *testing.T) {
	svc := NewService()
	_, err := svc.AddPoint(1, "", "", DataTypeInt16, ModeRandom)
	require.NoError(t, err)
	_, err = svc.AddPoint(1, "", "", DataTypeInt16, ModeIncrement)
	require.Error(t, err)
	_, err = svc.AddPoint(0, "", "", DataTypeInt16, ModeIncrement)
	require.Error(t, err)
	_, err = svc.AddPoint(2, "", "", DataTypeInt16, Mode("bad"))
	require.Error(t, err)

	svc.simulateOnce()
	state := svc.State()
	require.Len(t, state.Points, 1)

	require.NoError(t, svc.DeletePoint(1))
	require.Error(t, svc.DeletePoint(1))
	state = svc.State()
	assert.Empty(t, state.Points)
}

func TestServerReadWriteThroughInputAndHoldingRegisters(t *testing.T) {
	svc := NewService()
	port := openService(t, svc)
	t.Cleanup(func() { _, _ = svc.Close(context.Background()) })

	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     fmt.Sprintf("tcp://127.0.0.1:%d", port),
		Timeout: time.Second,
	})
	require.NoError(t, err)
	require.NoError(t, client.Open())
	t.Cleanup(func() { _ = client.Close() })
	client.SetUnitId(1)

	registers, err := client.ReadRegisters(5, 1, modbus.INPUT_REGISTER)
	require.NoError(t, err)
	assert.Equal(t, []uint16{0}, registers)

	require.NoError(t, client.WriteRegister(5, 123))
	registers, err = client.ReadRegisters(5, 1, modbus.INPUT_REGISTER)
	require.NoError(t, err)
	assert.Equal(t, []uint16{123}, registers)

	client.SetUnitId(2)
	_, err = client.ReadRegisters(5, 1, modbus.INPUT_REGISTER)
	require.Error(t, err)
}

func TestSimulationModes(t *testing.T) {
	svc := NewService()
	_, err := svc.AddPoint(1, "", "", DataTypeInt16, ModeIncrement)
	require.NoError(t, err)
	_, err = svc.AddPoint(2, "", "", DataTypeInt16, ModeDecrement)
	require.NoError(t, err)
	svc.simulateOnce()
	state := svc.State()
	require.Len(t, state.Points, 2)
	assert.Equal(t, uint16(1), state.Points[0].Value)
	assert.Equal(t, uint16(65535), state.Points[1].Value)
}

func TestPointDataTypesAddressRangesAndValues(t *testing.T) {
	svc := NewService()
	_, err := svc.AddPoint(1, "temperature", "", DataTypeFloat, ModeRandom)
	require.NoError(t, err)
	_, err = svc.AddPoint(2, "overlap", "", DataTypeInt16, ModeRandom)
	require.Error(t, err)
	_, err = svc.AddPoint(3, "counter", "", DataTypeInt32, ModeIncrement)
	require.NoError(t, err)
	_, err = svc.AddPoint(5, "float-counter", "", DataTypeFloat, ModeIncrement)
	require.NoError(t, err)
	_, err = svc.AddPoint(7, "flag", "", DataTypeInt16, ModeRandom)
	require.NoError(t, err)

	svc.simulateOnce()
	state := svc.State()
	require.Len(t, state.Points, 4)
	assert.Equal(t, "temperature", state.Points[0].Name)
	assert.Equal(t, RegisterTypeHolding, state.Points[0].RegisterType)
	assert.Equal(t, DataTypeFloat, state.Points[0].DataType)
	assert.Equal(t, "counter", state.Points[1].Name)
	assert.Equal(t, int32(1), state.Points[1].Value)
	assert.Equal(t, "float-counter", state.Points[2].Name)
	assert.InDelta(t, float32(0.1), state.Points[2].Value, 0.000001)
	assert.Equal(t, "flag", state.Points[3].Name)

	require.NoError(t, svc.DeletePoint(3))
	svc.mu.RLock()
	_, first := svc.registers[3]
	_, second := svc.registers[4]
	svc.mu.RUnlock()
	assert.False(t, first)
	assert.False(t, second)
}
