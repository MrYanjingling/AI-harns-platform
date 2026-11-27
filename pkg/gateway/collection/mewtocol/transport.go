package mewtocol

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
)

const maxResponseSize = 4096

type tcpTransport struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
}

func newTCPTransport(ctx context.Context, address string, timeout time.Duration) (*tcpTransport, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return &tcpTransport{conn: conn, timeout: timeout}, nil
}

func (t *tcpTransport) Available() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}

func (t *tcpTransport) Close() error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	conn := t.conn
	t.conn = nil
	return conn.Close()
}

func (t *tcpTransport) Exchange(ctx context.Context, request []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("mewtocol connection is closed"))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if _, err := t.conn.Write(request); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	response := make([]byte, 0, 128)
	buffer := make([]byte, 1)
	for len(response) < maxResponseSize {
		if _, err := io.ReadFull(t.conn, buffer); err != nil {
			return nil, collection.ConnectionLost(err)
		}
		response = append(response, buffer[0])
		if buffer[0] == '\r' {
			return response, nil
		}
	}
	return nil, fmt.Errorf("mewtocol response exceeds %d bytes", maxResponseSize)
}

func newMewtocolPool(ctx context.Context, address string, timeout time.Duration) (*connpool.Pool, error) {
	return connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		return newTCPTransport(ctx, address, timeout)
	})
}

func exchange(ctx context.Context, pool *connpool.Pool, request []byte) ([]byte, error) {
	var response []byte
	err := pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
		transport, ok := client.(*tcpTransport)
		if !ok {
			_ = client.Close()
			return false, fmt.Errorf("invalid mewtocol transport type")
		}
		resp, exchangeErr := transport.Exchange(ctx, request)
		if exchangeErr != nil {
			return true, exchangeErr
		}
		response = resp
		return false, nil
	})
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	return response, nil
}
