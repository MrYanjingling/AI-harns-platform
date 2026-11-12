package s7

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"harns-platform/pkg/utils/binutil"
)

const s7Timeout = time.Second

type s7Client struct {
	conn    net.Conn
	timeout time.Duration
	pdu     uint16
	mu      sync.Mutex
	closed  atomic.Bool
}

func newS7Client(ctx context.Context, address string, cotp []byte, pduSize int) (*s7Client, error) {
	dialer := net.Dialer{Timeout: s7Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	client := &s7Client{conn: conn, timeout: s7Timeout}
	_ = conn.SetDeadline(time.Now().Add(s7Timeout))
	if _, err := conn.Write(cotp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	cotpResponse := make([]byte, 22)
	if _, err := io.ReadFull(conn, cotpResponse); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if cotpResponse[5] != 208 {
		_ = conn.Close()
		return nil, fmt.Errorf("s7 COTP handshake failed")
	}
	if _, err := conn.Write(newS7COMMSetupMessage(pduSize)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	setupResponse := make([]byte, 27)
	if _, err := io.ReadFull(conn, setupResponse); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if setupResponse[8] != 3 {
		_ = conn.Close()
		return nil, fmt.Errorf("s7 setup failed")
	}
	client.pdu = binutil.ParseUint16(setupResponse[25:])
	if client.pdu == 0 {
		_ = conn.Close()
		return nil, fmt.Errorf("s7 PDU length is zero")
	}
	return client, nil
}

func (c *s7Client) Available() bool {
	return c != nil && c.conn != nil && !c.closed.Load()
}

func (c *s7Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	c.closed.Store(true)
	return c.conn.Close()
}

func (c *s7Client) ask(request []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(request); err != nil {
		return nil, err
	}
	header := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, err
	}
	total := int(binutil.ParseUint16(header[2:]))
	if total < 4 || total > 65535 {
		return nil, fmt.Errorf("invalid s7 TPKT length %d", total)
	}
	response := make([]byte, total)
	copy(response, header)
	if _, err := io.ReadFull(c.conn, response[4:]); err != nil {
		return nil, err
	}
	return response, nil
}

func newCOTPConnectMessage(rack, slot, connectionType uint8) []byte {
	rackSlotByte := (rack*2)<<4 + slot
	return []byte{
		0x03, 0x00, 0x00, 0x16,
		0x11, 0xe0, 0x00, 0x00, 0x00, 0x01, 0x00,
		0xc0, 0x01, 0x0a,
		0xc1, 0x02, 0x01, 0x00,
		0xc2, 0x02, connectionType, rackSlotByte,
	}
}

func newS7200COTPConnectMessage(localTSAP, remoteTSAP uint16) []byte {
	return []byte{
		0x03, 0x00, 0x00, 0x16,
		0x11, 0xe0, 0x00, 0x00, 0x00, 0x01, 0x00,
		0xc0, 0x01, 0x0a,
		0xc1, 0x02, byte(localTSAP >> 8), byte(localTSAP),
		0xc2, 0x02, byte(remoteTSAP >> 8), byte(remoteTSAP),
	}
}

func newS7COMMSetupMessage(pduSize int) []byte {
	if pduSize == 0 {
		pduSize = DefaultPDUSize
	}
	message := []byte{
		0x03, 0x00, 0x00, 0x19,
		0x02, 0xf0, 0x80,
		0x32, 0x01, 0x00, 0x00, 0x04, 0x00, 0x00, 0x08, 0x00, 0x00,
		0xf0, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0xe0,
	}
	binutil.WriteUint16BigEndian(message[23:], uint16(pduSize))
	return message
}
