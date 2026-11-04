package timeseries

import (
	"context"
	"errors"
	"sync"
	"time"

	"harns-platform/pkg/logging"
)

// ErrBatcherClosed is returned when enqueueing after Close.
var ErrBatcherClosed = errors.New("timeseries: batcher closed")

type pointWriter interface {
	WritePoints(ctx context.Context, points []*Point) error
}

type batcher struct {
	ch           chan *Point
	writer       pointWriter
	size         int
	interval     time.Duration
	writeTimeout time.Duration

	mu     sync.RWMutex
	closed bool
	wg     sync.WaitGroup
}

func newBatcher(writer pointWriter, size int, interval, writeTimeout time.Duration, queueSize int) *batcher {
	return &batcher{
		ch:           make(chan *Point, queueSize),
		writer:       writer,
		size:         size,
		interval:     interval,
		writeTimeout: writeTimeout,
	}
}

func (b *batcher) start() {
	b.wg.Add(1)
	go b.run()
}

func (b *batcher) run() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	batch := make([]*Point, 0, b.size)
	for {
		select {
		case point, ok := <-b.ch:
			if !ok {
				b.flush(batch)
				return
			}
			batch = append(batch, point)
			if len(batch) >= b.size {
				b.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				b.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

func (b *batcher) enqueue(ctx context.Context, points ...*Point) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return ErrBatcherClosed
	}
	for _, point := range points {
		if point == nil {
			continue
		}
		select {
		case b.ch <- point:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (b *batcher) close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	close(b.ch)
	b.mu.Unlock()

	b.wg.Wait()
}

func (b *batcher) flush(points []*Point) {
	if len(points) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), b.writeTimeout)
	defer cancel()
	if err := b.writer.WritePoints(ctx, points); err != nil {
		logging.ErrorS(err, "Failed to write timeseries batch", "points", len(points))
	}
}
