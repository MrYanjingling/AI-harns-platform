package modbustcp

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbus"
	"harns-platform/pkg/utils/binutil"
)

type closingModbusServer struct {
	listener net.Listener
	mu       sync.Mutex
	conns    []net.Conn
}

func newClosingModbusServer(t *testing.T) *closingModbusServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &closingModbusServer{listener: listener}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			server.mu.Lock()
			server.conns = append(server.conns, conn)
			server.mu.Unlock()
			_ = conn.Close()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		server.mu.Lock()
		defer server.mu.Unlock()
		for _, conn := range server.conns {
			_ = conn.Close()
		}
	})
	return server
}

func (s *closingModbusServer) Addr() string { return s.listener.Addr().String() }

type fakeModbusServer struct {
	listener net.Listener

	mu        sync.Mutex
	registers map[uint16]uint16
	conns     []net.Conn
}

func newFakeModbusServer(t *testing.T) *fakeModbusServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &fakeModbusServer{
		listener:  listener,
		registers: make(map[uint16]uint16),
	}
	go server.accept()
	t.Cleanup(server.Close)
	return server
}

func (s *fakeModbusServer) Addr() string { return s.listener.Addr().String() }

func (s *fakeModbusServer) SetRegister(address uint16, value uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registers[address] = value
}

func (s *fakeModbusServer) Close() {
	_ = s.listener.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, conn := range s.conns {
		_ = conn.Close()
	}
	s.conns = nil
}

func (s *fakeModbusServer) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()
		go s.handle(conn)
	}
}

func (s *fakeModbusServer) handle(conn net.Conn) {
	defer conn.Close()
	request := make([]byte, 12)
	for {
		if _, err := io.ReadFull(conn, request); err != nil {
			return
		}
		response := s.response(request)
		if _, err := conn.Write(response); err != nil {
			return
		}
	}
}

func (s *fakeModbusServer) response(request []byte) []byte {
	slave := request[6]
	function := request[7]
	start := binutil.ParseUint16BigEndian(request[8:])
	quantity := binutil.ParseUint16BigEndian(request[10:])

	data := make([]byte, 0)
	switch function {
	case modbus.FunctionReadCoils, modbus.FunctionReadDiscreteInputs:
		data = make([]byte, (quantity+7)/8)
	case modbus.FunctionReadHoldingRegisters, modbus.FunctionReadInputRegisters:
		data = make([]byte, int(quantity)*2)
		s.mu.Lock()
		for i := uint16(0); i < quantity; i++ {
			value := s.registers[start+i]
			binutil.WriteUint16BigEndian(data[int(i)*2:], value)
		}
		s.mu.Unlock()
	}

	response := make([]byte, 9+len(data))
	copy(response[0:2], request[0:2])
	binutil.WriteUint16BigEndian(response[4:], uint16(3+len(data)))
	response[6] = slave
	response[7] = function
	response[8] = byte(len(data))
	copy(response[9:], data)
	return response
}

func TestCollectorCollectsAndCalculates(t *testing.T) {
	server := newFakeModbusServer(t)
	server.SetRegister(0, 0x3F80)
	server.SetRegister(1, 0x0000)

	created, err := NewCollection(&CreateReq{
		Name:             "modbus-tcp-test",
		CollectorCycleMs: 20,
		ProtocolConfig: ProtocolConfig{
			Location:        server.Addr(),
			MemoryLayout:    "ABCD",
			PositionAddress: 1,
			TCPCount:        1,
		},
		Variables: []*collection.Variable{
			{Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead},
			{Name: "double", DataType: "float64", IsCalculated: true, Expression: "temperature * 2", AccessMode: collection.AccessModeRead},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = created.Stop(context.Background()) })
	require.NoError(t, created.Connect(context.Background()))
	require.NoError(t, created.Start(context.Background()))

	require.Eventually(t, func() bool {
		value, ok := created.GetVariableValue("double")
		return ok && value == float64(2)
	}, 2*time.Second, 10*time.Millisecond)
	value, ok := created.GetVariableValue("temperature")
	require.True(t, ok)
	assert.Equal(t, float32(1), value)
}

func TestTCPPoolFallsBackToBackup(t *testing.T) {
	backup := newFakeModbusServer(t)
	pool := newTCPPool("127.0.0.1:1", backup.Addr(), 1)
	require.NoError(t, pool.connect(context.Background()))
	defer pool.close()
	assert.True(t, pool.isBackup())
	assert.Equal(t, backup.Addr(), pool.activeAddress())
}

func TestTCPPoolSize(t *testing.T) {
	server := newFakeModbusServer(t)
	pool := newTCPPool(server.Addr(), "", 3)
	require.NoError(t, pool.connect(context.Background()))
	defer pool.close()
	assert.Len(t, pool.idle, 3)
}

func TestCollectorSwitchesToBackupAfterFailures(t *testing.T) {
	broken := newClosingModbusServer(t)
	backup := newFakeModbusServer(t)
	backup.SetRegister(0, 0x3F80)

	created, err := NewCollection(&CreateReq{
		Name:             "modbus-tcp-failover",
		CollectorCycleMs: 10,
		ProtocolConfig: ProtocolConfig{
			Location:        broken.Addr(),
			BackupLocation:  backup.Addr(),
			MemoryLayout:    "ABCD",
			PositionAddress: 1,
			TCPCount:        1,
		},
		Variables: []*collection.Variable{
			{Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = created.Stop(context.Background()) })
	require.NoError(t, created.Connect(context.Background()))
	require.NoError(t, created.Start(context.Background()))

	require.Eventually(t, func() bool {
		value, ok := created.GetVariableValue("temperature")
		return ok && value == float32(1)
	}, 3*time.Second, 10*time.Millisecond)
	assert.True(t, created.activeBackup)
}

func TestCollectorFailbackAfterProbeSuccess(t *testing.T) {
	primary := newFakeModbusServer(t)
	backup := newFakeModbusServer(t)
	primary.SetRegister(0, 0x3F80)

	created, err := NewCollection(&CreateReq{
		Name:             "modbus-tcp-failback",
		CollectorCycleMs: 20,
		ProtocolConfig: ProtocolConfig{
			Location:        primary.Addr(),
			BackupLocation:  backup.Addr(),
			MemoryLayout:    "ABCD",
			PositionAddress: 1,
			TCPCount:        1,
		},
		Variables: []*collection.Variable{
			{Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = created.Stop(context.Background()) })
	require.NoError(t, created.Connect(context.Background()))

	oldHold, oldProbe := backupHoldDuration, failbackProbeInterval
	backupHoldDuration = 0
	failbackProbeInterval = 0
	t.Cleanup(func() {
		backupHoldDuration = oldHold
		failbackProbeInterval = oldProbe
	})

	require.NoError(t, created.runtime.pool.switchTo(context.Background(), backup.Addr()))
	created.activeBackup = true
	created.runtime.mu.Lock()
	created.runtime.backupSince = time.Now()
	created.runtime.probeSuccess = 0
	created.runtime.lastProbe = time.Time{}
	created.runtime.mu.Unlock()

	for i := 0; i < failbackSuccessCount; i++ {
		created.maybeFailback(context.Background())
	}
	assert.False(t, created.runtime.pool.isBackup())
	assert.Equal(t, primary.Addr(), created.runtime.pool.activeAddress())
}
