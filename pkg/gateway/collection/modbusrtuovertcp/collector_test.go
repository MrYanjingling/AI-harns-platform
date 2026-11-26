package modbusrtuovertcp

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
	"harns-platform/pkg/utils/crcutil"
)

type fakeRTUServer struct {
	listener  net.Listener
	mu        sync.Mutex
	values    map[uint16]uint16
	exception bool
}

func newFakeRTUServer(t *testing.T) *fakeRTUServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &fakeRTUServer{listener: listener, values: make(map[uint16]uint16)}
	go server.accept()
	t.Cleanup(func() { _ = listener.Close() })
	return server
}

func (s *fakeRTUServer) Addr() string { return s.listener.Addr().String() }

func (s *fakeRTUServer) SetException() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exception = true
}

func (s *fakeRTUServer) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeRTUServer) handle(conn net.Conn) {
	defer conn.Close()
	request := make([]byte, 8)
	for {
		if _, err := io.ReadFull(conn, request); err != nil {
			return
		}
		function := request[1]
		start := binutil.ParseUint16BigEndian(request[2:])
		quantity := binutil.ParseUint16BigEndian(request[4:])
		s.mu.Lock()
		exception := s.exception
		s.mu.Unlock()
		if exception {
			response := []byte{request[0], function | 0x80, 0x02}
			crc := make([]byte, 2)
			binutil.WriteUint16BigEndian(crc, crcutil.CheckCrc16sum(response))
			if _, err := conn.Write(append(response, crc...)); err != nil {
				return
			}
			continue
		}
		data := make([]byte, int(quantity)*2)
		s.mu.Lock()
		for i := uint16(0); i < quantity; i++ {
			binutil.WriteUint16BigEndian(data[int(i)*2:], s.values[start+i])
		}
		s.mu.Unlock()
		response := []byte{request[0], function, byte(len(data))}
		response = append(response, data...)
		crc := make([]byte, 2)
		binutil.WriteUint16BigEndian(crc, crcutil.CheckCrc16sum(response))
		response = append(response, crc...)
		if _, err := conn.Write(response); err != nil {
			return
		}
	}
}

func TestCollectorReadsRTUOverTCP(t *testing.T) {
	server := newFakeRTUServer(t)
	server.values[0] = 0x3F80
	server.values[1] = 0x0000

	created, err := NewCollection(&CreateReq{
		Name:             "rtu-over-tcp",
		CollectorCycleMs: 20,
		ProtocolConfig: ProtocolConfig{
			Location:        server.Addr(),
			MemoryLayout:    "ABCD",
			PositionAddress: 1,
			Concurrency:     1,
		},
		Variables: []*collection.Variable{{
			Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead,
		}},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = created.Stop(context.Background()) })
	require.NoError(t, created.Connect(context.Background()))
	require.NoError(t, created.Start(context.Background()))

	require.Eventually(t, func() bool {
		value, ok := created.GetVariableValue("temperature")
		return ok && value == float32(1)
	}, 2*time.Second, 10*time.Millisecond)
}

func TestExceptionResponseKeepsRTUOverTCPConnected(t *testing.T) {
	server := newFakeRTUServer(t)
	server.SetException()

	created, err := NewCollection(&CreateReq{
		Name:             "rtu-over-tcp-exception",
		CollectorCycleMs: 20,
		ProtocolConfig: ProtocolConfig{
			Location:        server.Addr(),
			MemoryLayout:    "ABCD",
			PositionAddress: 1,
			Concurrency:     1,
		},
		Variables: []*collection.Variable{{
			Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead,
		}},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = created.Stop(context.Background()) })

	errors := make(chan error, 1)
	created.SetErrorReporter(func(err error) {
		if err == nil {
			return
		}
		select {
		case errors <- err:
		default:
		}
	})
	require.NoError(t, created.Connect(context.Background()))
	require.NoError(t, created.Start(context.Background()))

	select {
	case err := <-errors:
		require.Error(t, err)
		require.Contains(t, err.Error(), "modbus exception")
	case <-time.After(2 * time.Second):
		t.Fatal("expected modbus exception")
	}
	require.Eventually(t, func() bool {
		return created.GetCollectStatus() == collection.CollectStatusCollectingError
	}, 2*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, collection.CollectStatusCollectingError, created.GetCollectStatus())
}
