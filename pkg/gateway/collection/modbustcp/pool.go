package modbustcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const tcpConnectTimeout = time.Second

var errPoolClosed = errors.New("modbusTcp connection pool closed")

type tcpPool struct {
	mu sync.Mutex

	primary string
	backup  string
	active  string
	size    int
	timeout time.Duration

	idle   chan net.Conn
	conns  map[net.Conn]struct{}
	done   chan struct{}
	closed bool
}

func newTCPPool(primary, backup string, size int) *tcpPool {
	if size < 1 {
		size = 1
	}
	return &tcpPool{
		primary: primary,
		backup:  backup,
		active:  primary,
		size:    size,
		timeout: tcpConnectTimeout,
		idle:    make(chan net.Conn, size),
		conns:   make(map[net.Conn]struct{}, size),
		done:    make(chan struct{}),
	}
}

func (p *tcpPool) connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errPoolClosed
	}
	p.closeConnsLocked()
	if p.primary == "" {
		return fmt.Errorf("modbusTcp primary address is empty")
	}
	target := p.active
	if target == "" {
		target = p.primary
	}
	if err := p.dialLocked(ctx, target); err == nil {
		p.active = target
		return nil
	} else if target != p.primary || p.backup == "" || p.backup == p.primary {
		return err
	}
	p.active = p.backup
	if err := p.dialLocked(ctx, p.backup); err != nil {
		return err
	}
	return nil
}

func (p *tcpPool) preferBackup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.backup != "" {
		p.active = p.backup
	}
}

func (p *tcpPool) acquire(ctx context.Context) (net.Conn, error) {
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, errPoolClosed
		}
		if len(p.conns) < p.size {
			conn, err := p.dialOneLocked(p.active)
			if err != nil {
				p.mu.Unlock()
				return nil, err
			}
			p.mu.Unlock()
			return conn, nil
		}
		p.mu.Unlock()

		select {
		case conn := <-p.idle:
			if conn == nil {
				return nil, errPoolClosed
			}
			return conn, nil
		case <-p.done:
			return nil, errPoolClosed
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (p *tcpPool) release(conn net.Conn, failed bool) {
	if conn == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || failed {
		_ = conn.Close()
		delete(p.conns, conn)
		return
	}
	if _, ok := p.conns[conn]; !ok {
		_ = conn.Close()
		return
	}
	select {
	case p.idle <- conn:
	default:
		_ = conn.Close()
		delete(p.conns, conn)
	}
}

func (p *tcpPool) isBackup() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.backup != "" && p.active == p.backup
}

func (p *tcpPool) activeAddress() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active
}

func (p *tcpPool) switchTo(ctx context.Context, address string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errPoolClosed
	}
	previous := p.active
	p.closeConnsLocked()
	p.active = address
	if err := p.dialLocked(ctx, address); err == nil {
		return nil
	} else {
		p.active = previous
		if restoreErr := p.dialLocked(ctx, previous); restoreErr == nil {
			return err
		}
		return err
	}
}

func (p *tcpPool) probe(ctx context.Context, address string, request, response []byte) error {
	if address == "" {
		return fmt.Errorf("probe address is empty")
	}
	dialer := net.Dialer{Timeout: p.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	return exchangeConn(conn, p.timeout, request, response)
}

func (p *tcpPool) exchange(ctx context.Context, request, response []byte) error {
	conn, err := p.acquire(ctx)
	if err != nil {
		return err
	}
	err = exchangeConn(conn, p.timeout, request, response)
	p.release(conn, err != nil)
	return err
}

func (p *tcpPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.done)
	p.closeConnsLocked()
}

func (p *tcpPool) dialLocked(ctx context.Context, address string) error {
	conns := make([]net.Conn, 0, p.size)
	for i := 0; i < p.size; i++ {
		conn, err := p.dialOneLocked(address)
		if err != nil {
			for _, item := range conns {
				_ = item.Close()
				delete(p.conns, item)
			}
			return err
		}
		conns = append(conns, conn)
	}
	for _, conn := range conns {
		p.idle <- conn
	}
	return nil
}

func (p *tcpPool) dialOneLocked(address string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: p.timeout}
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(p.timeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	p.conns[conn] = struct{}{}
	return conn, nil
}

func (p *tcpPool) closeConnsLocked() {
	for {
		select {
		case conn := <-p.idle:
			if conn != nil {
				_ = conn.Close()
			}
		default:
			goto drainDone
		}
	}
drainDone:
	for conn := range p.conns {
		_ = conn.Close()
		delete(p.conns, conn)
	}
}

func exchangeConn(conn net.Conn, timeout time.Duration, request, response []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	if _, err := conn.Write(request); err != nil {
		return err
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	if _, err := io.ReadFull(conn, response); err != nil {
		return err
	}
	return nil
}
