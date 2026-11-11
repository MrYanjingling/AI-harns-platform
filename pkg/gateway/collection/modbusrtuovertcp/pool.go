package modbusrtuovertcp

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"harns-platform/pkg/gateway/collection/connpool"
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

func exchange(ctx context.Context, pool *connpool.Pool, request, response []byte) error {
	client, err := pool.Get(ctx)
	if err != nil {
		return err
	}
	netConn, ok := client.(*netClient)
	if !ok {
		pool.Put(client, true)
		return fmt.Errorf("invalid modbusRtuOverTcp connection type")
	}
	err = exchangeConn(netConn.Conn, request, response)
	pool.Put(client, err != nil)
	return err
}

func probeTCP(ctx context.Context, address string, request, response []byte) error {
	if address == "" {
		return fmt.Errorf("probe address is empty")
	}
	dialer := net.Dialer{Timeout: tcpConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	return exchangeConn(conn, request, response)
}

func exchangeConn(conn net.Conn, request, response []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(tcpConnectTimeout)); err != nil {
		return err
	}
	if _, err := conn.Write(request); err != nil {
		return err
	}
	if err := conn.SetReadDeadline(time.Now().Add(tcpConnectTimeout)); err != nil {
		return err
	}
	if _, err := io.ReadFull(conn, response); err != nil {
		return err
	}
	return nil
}
