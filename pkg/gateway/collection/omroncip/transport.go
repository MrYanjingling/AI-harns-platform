package omroncip

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const cipTimeout = 2 * time.Second

type cipClient struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
	session uint32
	context uint64
	closed  bool
}

func newCIPClient(ctx context.Context, address string, timeout time.Duration) (*cipClient, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	client := &cipClient{conn: conn, timeout: timeout}
	if err := client.registerSession(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *cipClient) registerSession(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, _, session, err := c.request(ctx, 0x0065, 0, []byte{1, 0, 0, 0})
	if err != nil {
		return err
	}
	if session == 0 {
		return fmt.Errorf("omron cip register session returned zero handle")
	}
	c.session = session
	return nil
}

func (c *cipClient) Exchange(ctx context.Context, cipRequest []byte) ([]byte, []byte, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	payload := make([]byte, 0, 16+len(cipRequest))
	payload = append(payload, make([]byte, 4)...)
	payload = append(payload, 10, 0)
	payload = append(payload, 2, 0)
	payload = append(payload, 0, 0, 0, 0)
	payload = append(payload, 0xB2, 0x00, byte(len(cipRequest)), byte(len(cipRequest)>>8))
	payload = append(payload, cipRequest...)
	wire, response, _, err := c.request(ctx, 0x006F, c.session, payload)
	if err != nil {
		return nil, wire, nil, err
	}
	data, err := parseRRData(response)
	if err != nil {
		return nil, wire, response, err
	}
	return data, wire, response, nil
}

func (c *cipClient) request(ctx context.Context, command uint16, session uint32, payload []byte) ([]byte, []byte, uint32, error) {
	if c.conn == nil || c.closed {
		return nil, nil, 0, fmt.Errorf("omron cip connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, 0, err
	}
	c.context++
	wire := make([]byte, 24+len(payload))
	binary.LittleEndian.PutUint16(wire[0:], command)
	binary.LittleEndian.PutUint16(wire[2:], uint16(len(payload)))
	binary.LittleEndian.PutUint32(wire[4:], session)
	binary.LittleEndian.PutUint64(wire[12:], c.context)
	copy(wire[24:], payload)
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, nil, 0, err
	}
	if _, err := c.conn.Write(wire); err != nil {
		return nil, nil, 0, err
	}
	header := make([]byte, 24)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return wire, nil, 0, err
	}
	if binary.LittleEndian.Uint16(header[:2]) != command {
		return wire, nil, 0, fmt.Errorf("omron cip encapsulation command mismatch")
	}
	length := int(binary.LittleEndian.Uint16(header[2:]))
	if length > 65535 {
		return wire, nil, 0, fmt.Errorf("invalid omron cip length %d", length)
	}
	response := make([]byte, length)
	if _, err := io.ReadFull(c.conn, response); err != nil {
		return wire, nil, 0, err
	}
	status := binary.LittleEndian.Uint32(header[8:])
	if status != 0 {
		return wire, response, 0, fmt.Errorf("omron cip encapsulation status %d (%s) for %s", status, encapsulationStatusText(status), encapsulationCommandName(command))
	}
	return wire, response, binary.LittleEndian.Uint32(header[4:]), nil
}

func parseRRData(payload []byte) ([]byte, error) {
	if len(payload) < 8 {
		return nil, fmt.Errorf("omron cip send rr data response is too short")
	}
	count := int(binary.LittleEndian.Uint16(payload[6:]))
	offset := 8
	for i := 0; i < count; i++ {
		if offset+4 > len(payload) {
			return nil, fmt.Errorf("omron cip send rr data item is too short")
		}
		itemType := binary.LittleEndian.Uint16(payload[offset:])
		itemLength := int(binary.LittleEndian.Uint16(payload[offset+2:]))
		offset += 4
		if offset+itemLength > len(payload) {
			return nil, fmt.Errorf("omron cip send rr data item length is invalid")
		}
		if itemType == 0x00B2 {
			return payload[offset : offset+itemLength], nil
		}
		offset += itemLength
	}
	return nil, fmt.Errorf("omron cip send rr data response has no unconnected data")
}

func (c *cipClient) Available() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.closed
}

func (c *cipClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	c.closed = true
	conn := c.conn
	c.conn = nil
	return conn.Close()
}

func encapsulationStatusText(status uint32) string {
	switch status {
	case 0:
		return "success"
	case 1:
		return "invalid or unsupported encapsulation command"
	case 2:
		return "insufficient memory"
	case 3:
		return "incorrect data in encapsulation packet"
	case 4:
		return "invalid session handle"
	case 5:
		return "invalid length"
	case 6:
		return "unsupported protocol revision"
	default:
		return "unknown encapsulation status"
	}
}

func encapsulationCommandName(command uint16) string {
	switch command {
	case 0x0063:
		return "ListServices (0x0063)"
	case 0x0064:
		return "ListIdentity (0x0064)"
	case 0x0065:
		return "RegisterSession (0x0065)"
	case 0x0066:
		return "UnregisterSession (0x0066)"
	case 0x006F:
		return "SendRRData (0x006F)"
	case 0x0070:
		return "SendUnitData (0x0070)"
	default:
		return fmt.Sprintf("command 0x%04X", command)
	}
}
