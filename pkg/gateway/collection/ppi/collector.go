package ppi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/collection/meterlink"
	"harns-platform/pkg/gateway/collection/mpi"
	"harns-platform/pkg/gateway/validation"
)

type tagRef struct {
	variable *collection.Variable
	address  mpi.Address
}

type collectorRuntime struct {
	mu      sync.Mutex
	tags    []*tagRef
	pool    *connpool.Pool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	last    time.Time
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("PPI collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if c.Location == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: location is required", collection.ErrInvalidConfig)
	}
	if c.Link == 1 && (c.Port < 1 || c.Port > 65535) {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: TCP port is invalid", collection.ErrInvalidConfig)
	}
	if c.Station < 1 || c.Station > 255 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: station is invalid", collection.ErrInvalidConfig)
	}
	tags := make([]*tagRef, 0)
	for _, variable := range c.GetCollectionMeta().GetVariables() {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := mpi.ParseAddress(variable.Address, variable.DataType)
		if err != nil {
			c.SetCollectStatus(collection.CollectStatusError)
			return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
		}
		tags = append(tags, &tagRef{variable: variable, address: address})
	}
	runtime := &collectorRuntime{tags: tags}
	c.runtime = runtime
	if len(tags) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		item, err := openClient(ctx, c.Link, c.Location, c.Port, byte(c.Station), c.timeout(), meterlink.SerialMode(c.BaudRate, c.DataBits, c.Parity, c.StopBits))
		if err != nil {
			return nil, collection.ConnectionLost(err)
		}
		return item, nil
	})
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.pool = pool
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) Start(ctx context.Context) error {
	runtime := c.runtime
	if runtime == nil {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	runtime.mu.Lock()
	if runtime.started {
		runtime.mu.Unlock()
		return nil
	}
	if runtime.pool == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("PPI runtime is not connected")
	}
	runtime.started = true
	runtime.last = time.Time{}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.wg.Add(1)
	runtime.mu.Unlock()
	go c.loop(runCtx)
	return nil
}

func (c *Collection) Stop(ctx context.Context) error {
	runtime := c.runtime
	if runtime == nil {
		c.SetCollectStatus(collection.CollectStatusStopped)
		return nil
	}
	runtime.mu.Lock()
	cancel := runtime.cancel
	runtime.cancel = nil
	runtime.started = false
	pool := runtime.pool
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if pool != nil {
		_ = pool.Close()
	}
	runtime.mu.Lock()
	if runtime.pool == pool {
		runtime.pool = nil
	}
	runtime.mu.Unlock()
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) loop(ctx context.Context) {
	runtime := c.runtime
	if runtime == nil {
		return
	}
	defer runtime.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
			if cycle <= 0 {
				cycle = time.Second
			}
			if !runtime.last.IsZero() && now.Sub(runtime.last) < cycle {
				continue
			}
			runtime.last = now
			if err := c.poll(ctx, runtime); err != nil {
				if collection.IsConnectionLost(err) {
					c.GetCollectionMeta().ReportError(err)
					collection.MarkConnectionLost(c, err)
					return
				}
				c.GetCollectionMeta().ReportError(err)
				c.SetCollectStatus(collection.CollectStatusCollectingError)
				continue
			}
			c.SetCollectStatus(collection.CollectStatusCollecting)
			c.GetCollectionMeta().ReportError(nil)
		}
	}
}

func (c *Collection) poll(ctx context.Context, runtime *collectorRuntime) error {
	for index, tag := range runtime.tags {
		if index > 0 && c.IntervalMs > 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(c.IntervalMs) * time.Millisecond):
			}
		}
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.ConnectionLost(err)
		}
		conn, ok := client.(*ppiClient)
		if !ok {
			runtime.pool.Put(client, true)
			return collection.ConnectionLost(fmt.Errorf("invalid PPI client type"))
		}
		data, err := conn.Read(tag.address)
		if err != nil {
			runtime.pool.Put(client, true)
			return collection.ConnectionLost(err)
		}
		value, err := mpi.DecodeValue(data, tag.address, tag.variable)
		runtime.pool.Put(client, false)
		if err != nil {
			return err
		}
		c.GetCollectionMeta().SetVariableValue(tag.variable.Name, collection.ApplyTransform(value, tag.variable))
	}
	return nil
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
}

func (c *Collection) timeout() time.Duration {
	if c.TimeoutMs <= 0 {
		return time.Duration(DefaultTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.TimeoutMs) * time.Millisecond
}

var _ collection.Collector = (*Collection)(nil)
