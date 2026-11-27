package ppi

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/collection/meterlink"
	"harns-platform/pkg/gateway/collection/mpi"
	"harns-platform/pkg/gateway/validation"
)

const (
	maxReadBatchBytes = 200
	maxReadGapBytes   = 8
)

type tagRef struct {
	variable *collection.Variable
	address  mpi.Address
}

type readBatch struct {
	address mpi.Address
	tags    []*tagRef
	merged  bool
}

type collectorRuntime struct {
	mu      sync.Mutex
	tags    []*tagRef
	batches []*readBatch
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
	runtime := &collectorRuntime{tags: tags, batches: buildReadBatches(tags)}
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

func isByteMergeableArea(area byte) bool {
	switch area {
	case 4, 5, 128, 129, 130, 131, 132, 133:
		return true
	default:
		return false
	}
}

func buildReadBatches(tags []*tagRef) []*readBatch {
	if len(tags) == 0 {
		return nil
	}
	mergeable := make([]*tagRef, 0, len(tags))
	batches := make([]*readBatch, 0)
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		if !isByteMergeableArea(tag.address.Area) || tag.address.Length == 0 || tag.address.Length > maxReadBatchBytes {
			batches = append(batches, &readBatch{
				address: tag.address,
				tags:    []*tagRef{tag},
				merged:  false,
			})
			continue
		}
		mergeable = append(mergeable, tag)
	}
	sort.SliceStable(mergeable, func(i, j int) bool {
		a, b := mergeable[i].address, mergeable[j].address
		if a.Area != b.Area {
			return a.Area < b.Area
		}
		if a.DB != b.DB {
			return a.DB < b.DB
		}
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		return a.Bit < b.Bit
	})
	var current *readBatch
	var batchEnd uint32
	for _, tag := range mergeable {
		tagStart := uint32(tag.address.Start)
		tagEnd := tagStart + uint32(tag.address.Length)
		if current != nil && current.address.Area == tag.address.Area && current.address.DB == tag.address.DB {
			newEnd := batchEnd
			if tagEnd > newEnd {
				newEnd = tagEnd
			}
			if tagStart <= batchEnd+maxReadGapBytes && newEnd-uint32(current.address.Start) <= maxReadBatchBytes {
				current.tags = append(current.tags, tag)
				batchEnd = newEnd
				current.address.Length = uint16(batchEnd - uint32(current.address.Start))
				continue
			}
		}
		current = &readBatch{
			address: mpi.Address{
				Area:   tag.address.Area,
				DB:     tag.address.DB,
				Start:  tag.address.Start,
				Length: tag.address.Length,
				Type:   2,
			},
			tags:   []*tagRef{tag},
			merged: true,
		}
		batchEnd = tagEnd
		batches = append(batches, current)
	}
	return batches
}

func (c *Collection) poll(ctx context.Context, runtime *collectorRuntime) error {
	for index, batch := range runtime.batches {
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
		data, err := conn.Read(batch.address)
		if err != nil {
			runtime.pool.Put(client, true)
			return collection.ConnectionLost(err)
		}
		runtime.pool.Put(client, false)
		for _, tag := range batch.tags {
			payload := data
			if batch.merged {
				offset := int(tag.address.Start - batch.address.Start)
				end := offset + int(tag.address.Length)
				if offset < 0 || end > len(data) {
					return fmt.Errorf("PPI batch response too short for variable %s", tag.variable.Name)
				}
				payload = data[offset:end]
			}
			value, err := mpi.DecodeValue(payload, tag.address, tag.variable)
			if err != nil {
				return err
			}
			c.GetCollectionMeta().SetVariableValue(tag.variable.Name, collection.ApplyTransform(value, tag.variable))
		}
	}
	return nil
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil || c.GetCollectStatus() != collection.CollectStatusCollecting {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		if action.Variable == nil {
			continue
		}
		address, err := mpi.ParseAddress(action.Variable.Address, action.Variable.DataType)
		if err != nil {
			return err
		}
		payload, err := encodeWritePayload(address, action.Variable, action.Value)
		if err != nil {
			return err
		}
		err = runtime.pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
			conn, ok := client.(*ppiClient)
			if !ok {
				return true, fmt.Errorf("invalid PPI client type")
			}
			writeErr := conn.Write(address, payload)
			return writeErr != nil, writeErr
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Collection) timeout() time.Duration {
	if c.TimeoutMs <= 0 {
		return time.Duration(DefaultTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.TimeoutMs) * time.Millisecond
}

var _ collection.Collector = (*Collection)(nil)
