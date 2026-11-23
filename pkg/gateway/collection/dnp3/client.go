package dnp3

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

const dnp3Timeout = 3 * time.Second

type dnp3Client struct {
	conn net.Conn

	mu             sync.Mutex
	masterID       uint16
	slaveID        uint16
	transportSeq   byte
	applicationSeq byte
	timeout        time.Duration
	closed         bool
}

func newDNP3Client(ctx context.Context, endpoint string, masterID, slaveID int, timeout time.Duration) (*dnp3Client, error) {
	if timeout <= 0 {
		timeout = dnp3Timeout
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	client := &dnp3Client{
		conn:     conn,
		masterID: uint16(masterID),
		slaveID:  uint16(slaveID),
		timeout:  timeout,
	}
	if err := client.resetLink(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *dnp3Client) Available() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.closed
}

func (c *dnp3Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *dnp3Client) resetLink() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	frame, err := buildLinkFrame(c.slaveID, c.masterID, 0xC0, nil)
	if err != nil {
		return err
	}
	if err := c.writeFrame(frame); err != nil {
		return err
	}
	link, _, err := readLinkFrame(c.conn, c.timeout)
	if err != nil {
		return err
	}
	switch linkFunction(link.Control) {
	case linkFnResetLinkAck, linkFnResetLink, linkFnLinkStatus:
		return nil
	default:
		return fmt.Errorf("unexpected DNP3 link reset function 0x%02X", linkFunction(link.Control))
	}
}

func (c *dnp3Client) Exchange(ctx context.Context, app []byte) ([]byte, []byte, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil || c.closed {
		return nil, nil, nil, fmt.Errorf("DNP3 connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	transport := wrapTransport(c.transportSeq, app)
	c.transportSeq = (c.transportSeq + 1) & 0x3F
	c.applicationSeq = (c.applicationSeq + 1) & 0x0F
	request, err := buildLinkFrame(c.slaveID, c.masterID, 0xC4, transport)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := c.writeFrame(request); err != nil {
		return nil, request, nil, err
	}
	for attempt := 0; attempt < 4; attempt++ {
		link, wire, err := readLinkFrame(c.conn, c.timeout)
		if err != nil {
			return nil, request, wire, err
		}
		switch linkFunction(link.Control) {
		case linkFnConfirmedData, linkFnUnconfirmedData:
			_, response, err := unwrapTransport(link.Payload)
			if err != nil {
				return nil, request, wire, err
			}
			return response, request, wire, nil
		case linkFnResetLink, linkFnResetLinkAck, linkFnLinkStatus:
			continue
		default:
			continue
		}
	}
	return nil, request, nil, fmt.Errorf("DNP3 response timeout")
}

func (c *dnp3Client) writeFrame(frame []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.conn.Write(frame)
	return err
}
