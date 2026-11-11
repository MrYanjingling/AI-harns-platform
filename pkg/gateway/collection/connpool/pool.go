// Package connpool provides a protocol-neutral connection pool.
package connpool

import (
	"context"
	"errors"
	"sync"
)

// ErrClosed reports that a pool has been closed.
var ErrClosed = errors.New("connection pool is closed")

// Client is one reusable protocol connection.
type Client interface {
	Available() bool
	Close() error
}

// Factory creates one already-connected client.
type Factory func(ctx context.Context) (Client, error)

// Pool manages clients without knowing protocol-specific operations.
type Pool struct {
	mu sync.Mutex

	newClient Factory
	max       int
	total     int
	idle      []Client
	waiters   map[uint64]chan Client
	nextID    uint64
	closed    bool
	done      chan struct{}
}

// NewPool creates a pool and opens up to max clients immediately.
func NewPool(ctx context.Context, max int, factory Factory) (*Pool, error) {
	if max < 1 {
		max = 1
	}
	pool := &Pool{
		newClient: factory,
		max:       max,
		waiters:   make(map[uint64]chan Client),
		done:      make(chan struct{}),
	}
	for i := 0; i < max; i++ {
		client, err := factory(ctx)
		if err != nil {
			_ = pool.Close()
			return nil, err
		}
		pool.idle = append(pool.idle, client)
		pool.total++
	}
	return pool, nil
}

// Get acquires one available client.
func (p *Pool) Get(ctx context.Context) (Client, error) {
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, ErrClosed
		}
		for len(p.idle) > 0 {
			client := p.idle[0]
			p.idle = p.idle[1:]
			if client.Available() {
				p.mu.Unlock()
				return client, nil
			}
			_ = client.Close()
			p.total--
		}
		if p.total < p.max {
			p.total++
			p.mu.Unlock()

			client, err := p.newClient(ctx)
			if err != nil {
				p.mu.Lock()
				p.total--
				p.mu.Unlock()
				return nil, err
			}
			return client, nil
		}
		waiterID := p.nextID
		p.nextID++
		waiter := make(chan Client, 1)
		p.waiters[waiterID] = waiter
		p.mu.Unlock()

		select {
		case client, ok := <-waiter:
			if !ok {
				return nil, ErrClosed
			}
			return client, nil
		case <-ctx.Done():
			p.mu.Lock()
			delete(p.waiters, waiterID)
			p.mu.Unlock()
			select {
			case client, ok := <-waiter:
				if ok {
					p.Put(client, false)
				}
			default:
			}
			return nil, ctx.Err()
		case <-p.done:
			return nil, ErrClosed
		}
	}
}

// Put returns one client to the pool.
func (p *Pool) Put(client Client, failed bool) {
	if client == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || failed || !client.Available() {
		_ = client.Close()
		p.total--
		return
	}
	for id, waiter := range p.waiters {
		delete(p.waiters, id)
		waiter <- client
		return
	}
	p.idle = append(p.idle, client)
}

// Close closes all idle clients and wakes all waiters.
func (p *Pool) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	close(p.done)
	for id, waiter := range p.waiters {
		delete(p.waiters, id)
		close(waiter)
	}
	for _, client := range p.idle {
		_ = client.Close()
	}
	p.idle = nil
	p.total = 0
	return nil
}
