package fins

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const tcpMagic = "FINS"

type transport interface {
	Exchange(ctx context.Context, request []byte) ([]byte, error)
	Close() error
}

type tcpTransport struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
	srcNode byte
	dstNode byte
	sid     byte
}

func newTCPTransport(ctx context.Context, address string, timeout time.Duration) (*tcpTransport, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	client := &tcpTransport{conn: conn, timeout: timeout}
	clientNode, serverNode, err := client.handshake()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	client.srcNode = clientNode
	client.dstNode = serverNode
	return client, nil
}

func (t *tcpTransport) handshake() (byte, byte, error) {
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return 0, 0, err
	}
	request := wrapTCP(0, []byte{0, 0, 0, 0})
	if _, err := t.conn.Write(request); err != nil {
		return 0, 0, err
	}
	command, errorCode, payload, err := readTCPPacket(t.conn)
	if err != nil {
		return 0, 0, err
	}
	if command != 1 {
		return 0, 0, fmt.Errorf("fins/tcp handshake command is %d", command)
	}
	if errorCode != 0 {
		return 0, 0, fmt.Errorf("fins/tcp handshake error %d", errorCode)
	}
	if len(payload) < 8 {
		return 0, 0, fmt.Errorf("fins/tcp handshake response is too short")
	}
	return payload[3], payload[7], nil
}

func (t *tcpTransport) Exchange(ctx context.Context, request []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil, fmt.Errorf("fins/tcp connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.sid++
	frame := append([]byte(nil), request...)
	if len(frame) < 10 {
		return nil, fmt.Errorf("fins request is too short")
	}
	frame[4] = t.dstNode
	frame[7] = t.srcNode
	frame[9] = t.sid
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return nil, err
	}
	if _, err := t.conn.Write(wrapTCP(2, frame)); err != nil {
		return nil, err
	}
	command, errorCode, payload, err := readTCPPacket(t.conn)
	if err != nil {
		return nil, err
	}
	if command != 2 {
		return nil, fmt.Errorf("fins/tcp data command is %d", command)
	}
	if errorCode != 0 {
		return nil, fmt.Errorf("fins/tcp data error %d", errorCode)
	}
	return payload, nil
}

func (t *tcpTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	conn := t.conn
	t.conn = nil
	return conn.Close()
}

func wrapTCP(command uint32, payload []byte) []byte {
	frame := make([]byte, 16+len(payload))
	copy(frame[:4], tcpMagic)
	length := uint32(8 + len(payload))
	frame[4] = byte(length >> 24)
	frame[5] = byte(length >> 16)
	frame[6] = byte(length >> 8)
	frame[7] = byte(length)
	frame[8] = byte(command >> 24)
	frame[9] = byte(command >> 16)
	frame[10] = byte(command >> 8)
	frame[11] = byte(command)
	copy(frame[16:], payload)
	return frame
}

func readTCPPacket(conn net.Conn) (uint32, uint32, []byte, error) {
	header := make([]byte, 16)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, 0, nil, err
	}
	if string(header[:4]) != tcpMagic {
		return 0, 0, nil, fmt.Errorf("fins/tcp magic mismatch")
	}
	length := int(uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7]))
	if length < 8 || length > 65535 {
		return 0, 0, nil, fmt.Errorf("invalid fins/tcp length %d", length)
	}
	command := uint32(header[8])<<24 | uint32(header[9])<<16 | uint32(header[10])<<8 | uint32(header[11])
	errorCode := uint32(header[12])<<24 | uint32(header[13])<<16 | uint32(header[14])<<8 | uint32(header[15])
	if command == 3 {
		return 0, 0, nil, fmt.Errorf("fins/tcp frame error %d", errorCode)
	}
	payload := make([]byte, length-8)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, 0, nil, err
	}
	return command, errorCode, payload, nil
}

type udpTransport struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
	dstNode byte
	sid     byte
}

func newUDPTransport(ctx context.Context, host string, port int, timeout time.Duration) (*udpTransport, error) {
	dialer := net.Dialer{Timeout: timeout}
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := dialer.DialContext(ctx, "udp", address)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	var node byte
	if ip != nil {
		if v4 := ip.To4(); v4 != nil {
			node = v4[3]
		}
	}
	return &udpTransport{conn: conn, timeout: timeout, dstNode: node}, nil
}

func (t *udpTransport) Exchange(ctx context.Context, request []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil, fmt.Errorf("fins/udp connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.sid++
	frame := append([]byte(nil), request...)
	if len(frame) < 10 {
		return nil, fmt.Errorf("fins request is too short")
	}
	frame[4] = t.dstNode
	frame[7] = 0
	frame[9] = t.sid
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return nil, err
	}
	if _, err := t.conn.Write(frame); err != nil {
		return nil, err
	}
	buffer := make([]byte, 65535)
	length, err := t.conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:length], nil
}

func (t *udpTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	conn := t.conn
	t.conn = nil
	return conn.Close()
}
