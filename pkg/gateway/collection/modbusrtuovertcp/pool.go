package modbusrtuovertcp

import (
	"context"
	"fmt"
	"net"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/collection/modbus"
)

const tcpConnectTimeout = time.Second

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

func newNetPool(ctx context.Context, address string, size int) (*connpool.Pool, error) {
	if address == "" {
		return nil, fmt.Errorf("modbusRtuOverTcp address is empty")
	}
	pool, err := connpool.NewPool(ctx, size, func(ctx context.Context) (connpool.Client, error) {
		dialer := net.Dialer{Timeout: tcpConnectTimeout}
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

func exchange(ctx context.Context, pool *connpool.Pool, request []byte, write bool) ([]byte, error) {
	var response []byte
	err := pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
		netConn, ok := client.(*netClient)
		if !ok {
			_ = client.Close()
			return false, fmt.Errorf("invalid modbusRtuOverTcp connection type")
		}
		resp, exchangeErr := exchangeConn(netConn.Conn, request, write)
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

func probeTCP(ctx context.Context, address string, request []byte) ([]byte, error) {
	if address == "" {
		return nil, fmt.Errorf("probe address is empty")
	}
	dialer := net.Dialer{Timeout: tcpConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return exchangeConn(conn, request, false)
}

func exchangeConn(conn net.Conn, request []byte, write bool) ([]byte, error) {
	if err := conn.SetWriteDeadline(time.Now().Add(tcpConnectTimeout)); err != nil {
		return nil, err
	}
	if _, err := conn.Write(request); err != nil {
		return nil, err
	}
	if err := conn.SetReadDeadline(time.Now().Add(tcpConnectTimeout)); err != nil {
		return nil, err
	}
	if write {
		return modbus.ReadRTUWriteResponse(conn)
	}
	return modbus.ReadRTUReadResponse(conn)
}
