package mc

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
)

type mcClient struct {
	conn    net.Conn
	family  Family
	ascii   bool
	udp     bool
	timeout time.Duration
}

func (c *mcClient) Available() bool { return c != nil && c.conn != nil }

func (c *mcClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *mcClient) Exchange(ctx context.Context, request []byte) ([]byte, error) {
	if c == nil || c.conn == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("MC connection is closed"))
	}
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if _, err := c.conn.Write(request); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if c.udp {
		buffer := make([]byte, 4096)
		n, err := c.conn.Read(buffer)
		if err != nil {
			return nil, collection.ConnectionLost(err)
		}
		return append([]byte(nil), buffer[:n]...), nil
	}
	if c.ascii {
		header := make([]byte, 18)
		if _, err := io.ReadFull(c.conn, header); err != nil {
			return nil, collection.ConnectionLost(err)
		}
		length, err := strconv.ParseUint(string(header[14:18]), 16, 16)
		if err != nil || int(length) > 65535-18 {
			return nil, fmt.Errorf("MC ASCII response length %q is invalid", string(header[14:18]))
		}
		response := make([]byte, 18+int(length))
		copy(response, header)
		if length > 0 {
			if _, err := io.ReadFull(c.conn, response[18:]); err != nil {
				return nil, collection.ConnectionLost(err)
			}
		}
		return response, nil
	}
	if c.family == Family1E {
		header := make([]byte, 2)
		if _, err := io.ReadFull(c.conn, header); err != nil {
			return nil, collection.ConnectionLost(err)
		}
		if header[1] != 0x00 {
			if header[1] == 0x5B {
				extra := make([]byte, 1)
				if _, err := io.ReadFull(c.conn, extra); err != nil {
					return nil, collection.ConnectionLost(err)
				}
				return append(header, extra...), nil
			}
			return header, nil
		}
		dataLen := a1eExpectedDataLength(request)
		response := make([]byte, 2+dataLen)
		copy(response, header)
		if dataLen > 0 {
			if _, err := io.ReadFull(c.conn, response[2:]); err != nil {
				return nil, collection.ConnectionLost(err)
			}
		}
		return response, nil
	}
	headerSize := responseHeaderSize(c.family)
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	lengthOffset := responseLengthOffset(c.family)
	length := int(binary.LittleEndian.Uint16(header[lengthOffset : lengthOffset+2]))
	if length < 0 || length > 65535-headerSize {
		return nil, fmt.Errorf("MC response length %d is invalid", length)
	}
	response := make([]byte, headerSize+length)
	copy(response, header)
	if length > 0 {
		if _, err := io.ReadFull(c.conn, response[headerSize:]); err != nil {
			return nil, collection.ConnectionLost(err)
		}
	}
	return response, nil
}

func newPool(ctx context.Context, config ProtocolConfig, info Info) (*connpool.Pool, error) {
	address := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	timeout := time.Duration(config.TimeoutMs) * time.Millisecond
	return connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		if config.TransportMode == 1 {
			remote, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				return nil, err
			}
			local := &net.UDPAddr{Port: config.SrcPort}
			conn, err := net.DialUDP("udp", local, remote)
			if err != nil {
				return nil, err
			}
			return &mcClient{conn: conn, family: info.Family, ascii: info.ASCII, udp: true, timeout: timeout}, nil
		}
		dialer := net.Dialer{Timeout: timeout}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		return &mcClient{conn: conn, family: info.Family, ascii: info.ASCII, timeout: timeout}, nil
	})
}
