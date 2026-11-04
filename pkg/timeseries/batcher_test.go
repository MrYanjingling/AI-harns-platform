package timeseries

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWriter struct {
	mu      sync.Mutex
	batches [][]*Point
}

func (f *fakeWriter) WritePoints(ctx context.Context, points []*Point) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := make([]*Point, len(points))
	copy(copied, points)
	f.batches = append(f.batches, copied)
	return nil
}

func (f *fakeWriter) batchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.batches)
}

func (f *fakeWriter) totalPoints() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := 0
	for _, batch := range f.batches {
		total += len(batch)
	}
	return total
}

func newTestPoint(value int64) *Point {
	return &Point{
		Measurement: "test",
		Tags:        map[string]string{"device_id": "d1"},
		Fields:      map[string]any{"v": value},
		Time:        time.Now(),
	}
}

func TestBatcherFlushOnSize(t *testing.T) {
	writer := &fakeWriter{}
	b := newBatcher(writer, 2, time.Hour, time.Second, 10)
	b.start()
	t.Cleanup(b.close)

	require.NoError(t, b.enqueue(context.Background(), newTestPoint(1), newTestPoint(2)))
	assert.Eventually(t, func() bool { return writer.batchCount() == 1 }, time.Second, 10*time.Millisecond)
	assert.Equal(t, 2, writer.totalPoints())
}

func TestBatcherFlushOnInterval(t *testing.T) {
	writer := &fakeWriter{}
	b := newBatcher(writer, 100, 50*time.Millisecond, time.Second, 10)
	b.start()
	t.Cleanup(b.close)

	require.NoError(t, b.enqueue(context.Background(), newTestPoint(1)))
	assert.Eventually(t, func() bool { return writer.batchCount() >= 1 }, time.Second, 10*time.Millisecond)
}

func TestBatcherCloseDrains(t *testing.T) {
	writer := &fakeWriter{}
	b := newBatcher(writer, 100, time.Hour, time.Second, 10)
	b.start()

	require.NoError(t, b.enqueue(context.Background(), newTestPoint(1)))
	b.close()
	assert.Equal(t, 1, writer.totalPoints())

	err := b.enqueue(context.Background(), newTestPoint(2))
	assert.ErrorIs(t, err, ErrBatcherClosed)
}

func TestBatcherEnqueueContextCanceled(t *testing.T) {
	writer := &fakeWriter{}
	b := newBatcher(writer, 1, time.Hour, time.Second, 1)
	// Do not start the worker: the channel fills and the next enqueue blocks.
	require.NoError(t, b.enqueue(context.Background(), newTestPoint(1)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := b.enqueue(ctx, newTestPoint(2))
	assert.ErrorIs(t, err, context.Canceled)
}

func TestBatcherMultipleBatches(t *testing.T) {
	writer := &fakeWriter{}
	b := newBatcher(writer, 2, time.Hour, time.Second, 10)
	b.start()
	t.Cleanup(b.close)

	require.NoError(t, b.enqueue(context.Background(), newTestPoint(1), newTestPoint(2), newTestPoint(3), newTestPoint(4)))
	assert.Eventually(t, func() bool { return writer.totalPoints() == 4 }, time.Second, 10*time.Millisecond)
	assert.Equal(t, 2, writer.batchCount())
}
