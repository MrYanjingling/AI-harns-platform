package kuka

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"harns-platform/pkg/gateway/collection"
)

const collectRequest = "<collect><cmd>1</cmd></collect>"

type kukaClient struct {
	conn     net.Conn
	listener net.Listener
	timeout  time.Duration
	closed   bool
}

func openKUKAClient(host string, port int, mode int, timeout time.Duration) (*kukaClient, error) {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	if mode == 1 {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return nil, err
		}
		if tcpListener, ok := listener.(*net.TCPListener); ok {
			_ = tcpListener.SetDeadline(time.Now().Add(timeout))
		}
		conn, err := listener.Accept()
		if err != nil {
			_ = listener.Close()
			return nil, err
		}
		if tcpListener, ok := listener.(*net.TCPListener); ok {
			_ = tcpListener.SetDeadline(time.Time{})
		}
		return &kukaClient{conn: conn, listener: listener, timeout: timeout}, nil
	}
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, err
	}
	return &kukaClient{conn: conn, timeout: timeout}, nil
}

func (c *kukaClient) Available() bool { return c != nil && c.conn != nil && !c.closed }
func (c *kukaClient) Close() error {
	if c == nil {
		return nil
	}
	c.closed = true
	if c.conn != nil {
		_ = c.conn.Close()
	}
	if c.listener != nil {
		return c.listener.Close()
	}
	return nil
}

func (c *kukaClient) Read(ctx context.Context, address string, dataType string) (any, error) {
	_ = ctx
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("KUKA connection is not established")
	}
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.Write([]byte(collectRequest)); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	data := make([]byte, 0, 2048)
	chunk := make([]byte, 2048)
	for len(data) < 1<<20 {
		n, err := c.conn.Read(chunk)
		if n > 0 {
			data = append(data, chunk[:n]...)
			root, parseErr := parseXML(data)
			if parseErr == nil {
				text, err := findXPath(root, address)
				if err != nil {
					return nil, err
				}
				return convertValue(text, dataType)
			}
		}
		if err != nil {
			if err == io.EOF && len(data) > 0 {
				break
			}
			return nil, collection.ConnectionLost(err)
		}
	}
	return nil, fmt.Errorf("invalid KUKA XML response")
}
