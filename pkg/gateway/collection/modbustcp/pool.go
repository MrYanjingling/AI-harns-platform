package modbustcp

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/utils/binutil"
)

const (
	tcpConnectTimeout   = 3 * time.Second
	tcpMBAPHeaderLength = 7
	tcpMaxLengthField   = 254
)

type netClient struct {
	net.Conn
}

func (c *netClient) Available() bool { return c != nil && c.Conn != nil }

func (c *netClient) Close() error {
	if c == nil || c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func newNetPool(ctx context.Context, address string, size int, timeout time.Duration) (*connpool.Pool, error) {
	if address == "" {
		return nil, fmt.Errorf("modbusTcp address is empty")
	}
	if timeout <= 0 {
		timeout = tcpConnectTimeout
	}
	pool, err := connpool.NewPool(ctx, size, func(ctx context.Context) (connpool.Client, error) {
		dialer := net.Dialer{Timeout: timeout}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		return &netClient{Conn: conn}, nil
	})
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func exchange(ctx context.Context, pool *connpool.Pool, request []byte, timeout time.Duration) ([]byte, error) {
	client, err := pool.Get(ctx)
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	netConn, ok := client.(*netClient)
	if !ok {
		pool.Put(client, true)
		return nil, fmt.Errorf("invalid modbusTcp connection type")
	}
	response, err := exchangeConn(netConn.Conn, request, timeout)
	pool.Put(client, err != nil)
	if err != nil {
		if collection.IsConnectionLost(err) {
			return nil, collection.ConnectionLost(err)
		}
		return nil, err
	}
	return response, nil
}

func probeTCP(ctx context.Context, address string, request []byte, timeout time.Duration) ([]byte, error) {
	if address == "" {
		return nil, fmt.Errorf("probe address is empty")
	}
	if timeout <= 0 {
		timeout = tcpConnectTimeout
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return exchangeConn(conn, request, timeout)
}

func exchangeConn(conn net.Conn, request []byte, timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		timeout = tcpConnectTimeout
	}
	deadline := time.Now().Add(timeout)
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return nil, err
	}
	if _, err := conn.Write(request); err != nil {
		return nil, err
	}
	if err := conn.SetReadDeadline(deadline); err != nil {
		return nil, err
	}
	return readTCPResponse(conn)
}

func readTCPResponse(conn net.Conn) ([]byte, error) {
	header := make([]byte, tcpMBAPHeaderLength)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := int(binutil.ParseUint16BigEndian(header[4:]))
	if length < 2 || length > tcpMaxLengthField {
		return nil, fmt.Errorf("modbusTcp frame length %d is invalid", length)
	}
	body := make([]byte, length-1)
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, err
	}
	return append(header, body...), nil
}
