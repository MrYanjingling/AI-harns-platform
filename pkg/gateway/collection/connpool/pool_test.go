package connpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	available bool
	closed    atomic.Bool
}

func (c *testClient) Available() bool { return c.available && !c.closed.Load() }
func (c *testClient) Close() error {
	c.closed.Store(true)
	return nil
}

func TestPoolCreatesOnConstruction(t *testing.T) {
	var created atomic.Int32
	pool, err := NewPool(context.Background(), 2, func(context.Context) (Client, error) {
		created.Add(1)
		return &testClient{available: true}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), created.Load())

	first, err := pool.Get(context.Background())
	require.NoError(t, err)
	second, err := pool.Get(context.Background())
	require.NoError(t, err)
	pool.Put(first, false)
	pool.Put(second, false)
	require.NoError(t, pool.Close())
	assert.True(t, first.(*testClient).closed.Load())
}

func TestPoolFailedPutCreatesOnNextGet(t *testing.T) {
	var created atomic.Int32
	pool, err := NewPool(context.Background(), 1, func(context.Context) (Client, error) {
		created.Add(1)
		return &testClient{available: true}, nil
	})
	require.NoError(t, err)
	client, err := pool.Get(context.Background())
	require.NoError(t, err)
	pool.Put(client, true)
	assert.Equal(t, int32(1), created.Load())

	replacement, err := pool.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(2), created.Load())
	pool.Put(replacement, false)
	_ = pool.Close()
}

func TestPoolWaiterReceivesReleasedClient(t *testing.T) {
	pool, err := NewPool(context.Background(), 1, func(context.Context) (Client, error) {
		return &testClient{available: true}, nil
	})
	require.NoError(t, err)
	client, err := pool.Get(context.Background())
	require.NoError(t, err)

	result := make(chan Client, 1)
	go func() {
		next, _ := pool.Get(context.Background())
		result <- next
	}()
	time.Sleep(10 * time.Millisecond)
	pool.Put(client, false)
	select {
	case next := <-result:
		assert.Same(t, client, next)
	case <-time.After(time.Second):
		t.Fatal("waiter did not receive released client")
	}
	_ = pool.Close()
}

func TestPoolCloseWakesWaiter(t *testing.T) {
	pool, err := NewPool(context.Background(), 1, func(context.Context) (Client, error) {
		return &testClient{available: true}, nil
	})
	require.NoError(t, err)
	_, err = pool.Get(context.Background())
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	errCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		_, err := pool.Get(context.Background())
		errCh <- err
	}()
	time.Sleep(10 * time.Millisecond)
	_ = pool.Close()
	wg.Wait()
	assert.ErrorIs(t, <-errCh, ErrClosed)
}

func TestPoolWaiterRevivedAfterFailedPut(t *testing.T) {
	var created atomic.Int32
	pool, err := NewPool(context.Background(), 1, func(context.Context) (Client, error) {
		created.Add(1)
		return &testClient{available: true}, nil
	})
	require.NoError(t, err)
	client, err := pool.Get(context.Background())
	require.NoError(t, err)

	result := make(chan error, 1)
	go func() {
		next, err := pool.Get(context.Background())
		if err == nil {
			pool.Put(next, false)
		}
		result <- err
	}()
	time.Sleep(10 * time.Millisecond)
	pool.Put(client, true)

	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("waiter did not retry after failed put")
	}
	assert.Equal(t, int32(2), created.Load())
	_ = pool.Close()
}

func TestPoolWaiterRevivedWhenDialFails(t *testing.T) {
	var created atomic.Int32
	pool, err := NewPool(context.Background(), 1, func(context.Context) (Client, error) {
		if created.Add(1) == 1 {
			return &testClient{available: true}, nil
		}
		return nil, errors.New("dial failed")
	})
	require.NoError(t, err)
	client, err := pool.Get(context.Background())
	require.NoError(t, err)

	result := make(chan error, 1)
	go func() {
		_, err := pool.Get(context.Background())
		result <- err
	}()
	time.Sleep(10 * time.Millisecond)
	pool.Put(client, true)

	select {
	case err := <-result:
		assert.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("waiter did not retry after failed dial")
	}
	_ = pool.Close()
}

func TestPoolDoWithRetryReconnectsOnce(t *testing.T) {
	var created atomic.Int32
	pool, err := NewPool(context.Background(), 1, func(context.Context) (Client, error) {
		created.Add(1)
		return &testClient{available: true}, nil
	})
	require.NoError(t, err)
	defer pool.Close()

	var calls int
	err = pool.DoWithRetry(context.Background(), DefaultWriteAttempts, func(client Client) (bool, error) {
		calls++
		if calls == 1 {
			return true, errors.New("EOF")
		}
		return false, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.Equal(t, int32(2), created.Load())
}

func TestPoolDoWithRetrySkipsNonRetryableError(t *testing.T) {
	var created atomic.Int32
	pool, err := NewPool(context.Background(), 1, func(context.Context) (Client, error) {
		created.Add(1)
		return &testClient{available: true}, nil
	})
	require.NoError(t, err)
	defer pool.Close()

	var calls int
	protoErr := errors.New("protocol error")
	err = pool.DoWithRetry(context.Background(), DefaultWriteAttempts, func(client Client) (bool, error) {
		calls++
		return false, protoErr
	})
	assert.ErrorIs(t, err, protoErr)
	assert.Equal(t, 1, calls)
	assert.Equal(t, int32(1), created.Load())
}
