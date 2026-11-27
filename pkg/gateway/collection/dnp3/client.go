package dnp3

import (
	"context"
	"errors"
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
	wait := c.timeout
	if wait > 1500*time.Millisecond {
		wait = 1500 * time.Millisecond
	}
	deadline := time.Now().Add(wait)
	for attempt := 0; attempt < 4; attempt++ {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			// Per IEEE 1815 over TCP, RESET_LINK_STATES is optional/obsolete;
			// some outstations ignore it and accept Application Layer polls directly.
			return nil
		}
		link, _, err := readLinkFrame(c.conn, remaining)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return nil
			}
			return err
		}
		switch linkFunction(link.Control) {
		case linkFnResetLinkAck, linkFnResetLink, linkFnLinkStatus:
			return nil
		case linkFnConfirmedData, linkFnUnconfirmedData:
			// Like Neuron dnp3_stack_recv, ignore non-ACK frames (e.g. unsolicited
			// FUNC=4 frames pushed on connect) while waiting for link ACK, but send
			// Application Layer CONFIRM if CON=1 so the outstation exits ConfirmWait.
			if _, app, unwrapErr := unwrapTransport(link.Payload); unwrapErr == nil {
				c.maybeSendAppConfirm(app)
			}
			continue
		default:
			continue
		}
	}
	return nil
}

func (c *dnp3Client) maybeSendAppConfirm(app []byte) {
	if len(app) < 2 || app[0]&0x20 == 0 {
		return
	}
	// Preserve UNS (bit 4) and SEQ (bits 0..3); set FIR=1, FIN=1, CON=0; FUNC=0x00 (CONFIRM).
	confirmCtrl := byte(0xC0) | (app[0] & 0x1F)
	confirmApp := []byte{confirmCtrl, 0x00}
	transport := wrapTransport(c.transportSeq, confirmApp)
	c.transportSeq = (c.transportSeq + 1) & 0x3F
	frame, err := buildLinkFrame(c.slaveID, c.masterID, 0xC4, transport)
	if err != nil {
		return
	}
	_ = c.writeFrame(frame)
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
	var assembled []byte
	var lastWire []byte
	for attempt := 0; attempt < 64; attempt++ {
		link, wire, err := readLinkFrame(c.conn, c.timeout)
		if err != nil {
			return nil, request, wire, err
		}
		lastWire = wire
		switch linkFunction(link.Control) {
		case linkFnConfirmedData, linkFnUnconfirmedData:
			if len(link.Payload) < 1 {
				return nil, request, wire, fmt.Errorf("DNP3 transport payload is empty")
			}
			th := link.Payload[0]
			fir := th&0x40 != 0
			fin := th&0x80 != 0
			seg := link.Payload[1:]
			if fir {
				assembled = append(assembled[:0], seg...)
			} else {
				assembled = append(assembled, seg...)
			}
			if !fin {
				continue
			}
			c.maybeSendAppConfirm(assembled)
			if len(assembled) >= 2 && assembled[1] == appFunctionUnsolicited {
				assembled = assembled[:0]
				continue
			}
			return assembled, request, lastWire, nil
		case linkFnResetLink, linkFnResetLinkAck, linkFnLinkStatus:
			continue
		default:
			continue
		}
	}
	return nil, request, lastWire, fmt.Errorf("DNP3 response timeout")
}

func (c *dnp3Client) writeFrame(frame []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.conn.Write(frame)
	return err
}
