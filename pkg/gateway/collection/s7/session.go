package s7

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"harns-platform/pkg/utils/binutil"
)

const s7Timeout = time.Second

type s7Session struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
}

func newS7Session(ctx context.Context, address string, rack, slot uint8) (*s7Session, uint16, error) {
	dialer := net.Dialer{Timeout: s7Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, 0, err
	}
	session := &s7Session{conn: conn, timeout: s7Timeout}
	_ = conn.SetDeadline(time.Now().Add(s7Timeout))
	if _, err := conn.Write(newCOTPConnectMessage(rack, slot)); err != nil {
		_ = conn.Close()
		return nil, 0, err
	}
	cotpResponse := make([]byte, 22)
	if _, err := io.ReadFull(conn, cotpResponse); err != nil {
		_ = conn.Close()
		return nil, 0, err
	}
	if cotpResponse[5] != 208 {
		_ = conn.Close()
		return nil, 0, fmt.Errorf("s7 COTP handshake failed")
	}
	if _, err := conn.Write(newS7COMMSetupMessage()); err != nil {
		_ = conn.Close()
		return nil, 0, err
	}
	setupResponse := make([]byte, 27)
	if _, err := io.ReadFull(conn, setupResponse); err != nil {
		_ = conn.Close()
		return nil, 0, err
	}
	if setupResponse[8] != 3 {
		_ = conn.Close()
		return nil, 0, fmt.Errorf("s7 setup failed")
	}
	pdu := binutil.ParseUint16(setupResponse[25:])
	if pdu == 0 {
		_ = conn.Close()
		return nil, 0, fmt.Errorf("s7 PDU length is zero")
	}
	return session, pdu, nil
}

func (s *s7Session) ask(request []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.conn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
		return nil, err
	}
	if _, err := s.conn.Write(request); err != nil {
		return nil, err
	}
	header := make([]byte, 4)
	if _, err := io.ReadFull(s.conn, header); err != nil {
		return nil, err
	}
	total := int(binutil.ParseUint16(header[2:]))
	if total < 4 || total > 65535 {
		return nil, fmt.Errorf("invalid s7 TPKT length %d", total)
	}
	response := make([]byte, total)
	copy(response, header)
	if _, err := io.ReadFull(s.conn, response[4:]); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *s7Session) close() {
	if s != nil && s.conn != nil {
		_ = s.conn.Close()
	}
}

type s7Pool struct {
	address string
	rack    uint8
	slot    uint8
	size    int
	pdu     uint16
	idle    chan *s7Session
}

func newS7Pool(address string, rack, slot uint8, size int) *s7Pool {
	if size < 1 {
		size = 1
	}
	return &s7Pool{
		address: address,
		rack:    rack,
		slot:    slot,
		size:    size,
		idle:    make(chan *s7Session, size),
	}
}

func (p *s7Pool) connect(ctx context.Context) error {
	sessions := make([]*s7Session, 0, p.size)
	for i := 0; i < p.size; i++ {
		session, pdu, err := newS7Session(ctx, p.address, p.rack, p.slot)
		if err != nil {
			for _, item := range sessions {
				item.close()
			}
			return err
		}
		if p.pdu == 0 {
			p.pdu = pdu
		}
		sessions = append(sessions, session)
	}
	for _, session := range sessions {
		p.idle <- session
	}
	return nil
}

func (p *s7Pool) acquire(ctx context.Context) (*s7Session, error) {
	select {
	case session := <-p.idle:
		if session == nil {
			return nil, fmt.Errorf("s7 session pool closed")
		}
		return session, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *s7Pool) release(session *s7Session, failed bool) {
	if session == nil {
		return
	}
	if failed {
		session.close()
		replacement, _, err := newS7Session(context.Background(), p.address, p.rack, p.slot)
		if err == nil {
			p.idle <- replacement
		}
		return
	}
	p.idle <- session
}

func (p *s7Pool) close() {
	for {
		select {
		case session := <-p.idle:
			session.close()
		default:
			return
		}
	}
}

func newCOTPConnectMessage(rack, slot uint8) []byte {
	rackSlotByte := (rack*2)<<4 + slot
	return []byte{
		0x03, 0x00, 0x00, 0x16,
		0x11, 0xe0, 0x00, 0x00, 0x00, 0x01, 0x00,
		0xc0, 0x01, 0x0a,
		0xc1, 0x02, 0x01, 0x00,
		0xc2, 0x02, 0x01, rackSlotByte,
	}
}

func newS7COMMSetupMessage() []byte {
	return []byte{
		0x03, 0x00, 0x00, 0x19,
		0x02, 0xf0, 0x80,
		0x32, 0x01, 0x00, 0x00, 0x04, 0x00, 0x00, 0x08, 0x00, 0x00,
		0xf0, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0xe0,
	}
}
