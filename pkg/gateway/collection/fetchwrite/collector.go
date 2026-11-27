package fetchwrite

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
)

const (
	maxReadBatchBytes = 1024
	maxReadGapUnits   = 8
)

type tagRef struct {
	variable *collection.Variable
	address  Address
}

type readBatch struct {
	org       byte
	db        byte
	start     uint16
	unitCount uint16
	tags      []*tagRef
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
		return fmt.Errorf("fetch/write collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Location) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: location is required", collection.ErrInvalidConfig)
	}
	if c.Port < 1 || c.Port > 65535 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: port is invalid", collection.ErrInvalidConfig)
	}
	tags := make([]*tagRef, 0)
	for _, variable := range c.GetCollectionMeta().GetVariables() {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := ParseAddress(variable.Address, variable.DataType)
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
	concurrency := c.Concurrency
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	if concurrency > MaxConcurrency {
		concurrency = MaxConcurrency
	}
	endpoint := c.endpoint()
	pool, err := connpool.NewPool(ctx, concurrency, func(ctx context.Context) (connpool.Client, error) {
		item, err := dialClient(ctx, endpoint, c.timeout())
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
		return fmt.Errorf("fetch/write runtime is not connected")
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

func buildReadBatches(tags []*tagRef) []*readBatch {
	if len(tags) == 0 {
		return nil
	}
	sorted := make([]*tagRef, 0, len(tags))
	for _, tag := range tags {
		if tag != nil {
			sorted = append(sorted, tag)
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i].address, sorted[j].address
		if a.Org != b.Org {
			return a.Org < b.Org
		}
		if a.DB != b.DB {
			return a.DB < b.DB
		}
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		if a.ByteOffset != b.ByteOffset {
			return a.ByteOffset < b.ByteOffset
		}
		return a.Bit < b.Bit
	})

	batches := make([]*readBatch, 0)
	var current *readBatch
	var batchEnd uint32
	for _, tag := range sorted {
		ub := uint32(unitBytes(tag.address.Org))
		maxUnits := uint32(maxReadBatchBytes) / ub
		if maxUnits == 0 {
			maxUnits = 1
		}
		tagStart := uint32(tag.address.Start)
		tagEnd := tagStart + uint32(tag.address.UnitCount)
		if current != nil && current.org == tag.address.Org && current.db == tag.address.DB {
			newEnd := batchEnd
			if tagEnd > newEnd {
				newEnd = tagEnd
			}
			if tagStart <= batchEnd+maxReadGapUnits && newEnd-uint32(current.start) <= maxUnits {
				current.tags = append(current.tags, tag)
				batchEnd = newEnd
				current.unitCount = uint16(batchEnd - uint32(current.start))
				continue
			}
		}
		current = &readBatch{
			org:       tag.address.Org,
			db:        tag.address.DB,
			start:     tag.address.Start,
			unitCount: tag.address.UnitCount,
			tags:      []*tagRef{tag},
		}
		batchEnd = tagEnd
		batches = append(batches, current)
	}
	return batches
}

func (c *Collection) poll(ctx context.Context, runtime *collectorRuntime) error {
	for _, batch := range runtime.batches {
		if ctx.Err() != nil {
			return nil
		}
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.ConnectionLost(err)
		}
		conn, ok := client.(*fwClient)
		if !ok {
			runtime.pool.Put(client, true)
			return collection.ConnectionLost(fmt.Errorf("invalid fetch/write client type"))
		}
		started := time.Now()
		data, req, rawResp, err := conn.Fetch(batch.org, batch.db, batch.start, batch.unitCount)
		traceID := c.traceRequest("fetch", req)
		c.traceResponse(traceID, "fetch", rawResp, time.Since(started), err)
		if err != nil {
			runtime.pool.Put(client, true)
			if isTransportError(err) {
				return collection.ConnectionLost(err)
			}
			return err
		}
		runtime.pool.Put(client, false)

		ub := unitBytes(batch.org)
		for _, tag := range batch.tags {
			unitOffset := int(tag.address.Start - batch.start)
			startByte := unitOffset * ub
			endByte := startByte + int(tag.address.UnitCount)*ub
			if startByte < 0 || endByte > len(data) {
				return fmt.Errorf("fetch/write batch response too short for variable %s", tag.variable.Name)
			}
			value, err := decodeValue(data[startByte:endByte], tag.address, tag.variable)
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
		address, err := ParseAddress(action.Variable.Address, action.Variable.DataType)
		if err != nil {
			return err
		}
		err = runtime.pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
			conn, ok := client.(*fwClient)
			if !ok {
				return true, fmt.Errorf("invalid fetch/write client type")
			}
			writeErr := c.writeAction(conn, address, action.Variable, action.Value)
			return writeErr != nil, writeErr
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Collection) writeAction(conn *fwClient, addr Address, variable *collection.Variable, raw any) error {
	ub := unitBytes(addr.Org)
	unitBytesTotal := int(addr.UnitCount) * ub
	needsRMW := addr.IsBit || addr.ByteOffset != 0 || int(addr.ByteLength) != unitBytesTotal

	var payload []byte
	if needsRMW {
		existing, req, resp, err := conn.Fetch(addr.Org, addr.DB, addr.Start, addr.UnitCount)
		traceID := c.traceRequest("fetch-rmw", req)
		c.traceResponse(traceID, "fetch-rmw", resp, 0, err)
		if err != nil {
			return err
		}
		payload = make([]byte, unitBytesTotal)
		copy(payload, existing)
		if addr.IsBit {
			bitVal, err := toBool(raw)
			if err != nil {
				return err
			}
			if err := applyBit(payload, addr, bitVal); err != nil {
				return err
			}
		} else {
			encoded, err := encodeValue(addr, variable, raw)
			if err != nil {
				return err
			}
			copy(payload[addr.ByteOffset:int(addr.ByteOffset)+len(encoded)], encoded)
		}
	} else {
		var err error
		payload, err = encodeValue(addr, variable, raw)
		if err != nil {
			return err
		}
	}
	started := time.Now()
	req, resp, err := conn.Write(addr.Org, addr.DB, addr.Start, addr.UnitCount, payload)
	traceID := c.traceRequest("write", req)
	c.traceResponse(traceID, "write", resp, time.Since(started), err)
	return err
}

func (c *Collection) timeout() time.Duration {
	if c.TimeoutMs <= 0 {
		return time.Duration(DefaultTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.TimeoutMs) * time.Millisecond
}

func (c *Collection) SetFrameTracer(tracer collection.FrameTracer) { c.frameTracer = tracer }

func (c *Collection) traceRequest(operation string, data []byte) uint64 {
	if c.frameTracer == nil || len(data) == 0 {
		return 0
	}
	id := c.frameTracer.NextTraceID()
	c.frameTracer.TraceRequest(id, operation, data)
	return id
}

func (c *Collection) traceResponse(id uint64, operation string, data []byte, duration time.Duration, err error) {
	if c.frameTracer == nil || id == 0 {
		return
	}
	c.frameTracer.TraceResponse(id, operation, data, duration, err)
}

func isTransportError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "eof") ||
		strings.Contains(msg, "closed") || strings.Contains(msg, "refused") ||
		strings.Contains(msg, "reset") || strings.Contains(msg, "broken pipe")
}
