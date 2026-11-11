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
	listener net.Listener
	mu       sync.Mutex
	values   map[uint16]uint16
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
			TCPCount:        1,
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
