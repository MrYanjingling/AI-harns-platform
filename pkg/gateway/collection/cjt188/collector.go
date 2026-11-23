package cjt188

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/collection/meterlink"
	"harns-platform/pkg/gateway/validation"
)

type frameRef struct {
	variable *collection.Variable
	address  meterAddress
}

type collectorRuntime struct {
	mu      sync.Mutex
	frames  []*frameRef
	pool    *connpool.Pool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	last    time.Time
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("CJ/T 188 collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Location) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: location is required", collection.ErrInvalidConfig)
	}
	if c.Link == 1 && (c.Port < 1 || c.Port > 65535) {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: TCP port is invalid", collection.ErrInvalidConfig)
	}
	if c.IntervalMs < 0 {
		c.IntervalMs = DefaultIntervalMs
	}
	frames := make([]*frameRef, 0)
	for _, variable := range c.GetCollectionMeta().GetVariables() {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := parseAddress(variable.Address)
		if err != nil {
			c.SetCollectStatus(collection.CollectStatusError)
			return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
		}
		frames = append(frames, &frameRef{variable: variable, address: address})
	}
	runtime := &collectorRuntime{frames: frames}
	c.runtime = runtime
	if len(frames) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := meterlink.NewPool(ctx, c.Link, c.Location, c.Port, c.timeout(), meterlink.SerialMode(c.BaudRate, c.DataBits, c.Parity, c.StopBits))
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
		return fmt.Errorf("CJ/T 188 runtime is not connected")
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
	for index, frame := range runtime.frames {
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
		item, ok := client.(*meterlink.Client)
		if !ok {
			runtime.pool.Put(client, true)
			return fmt.Errorf("invalid CJ/T 188 client type")
		}
		withSER := c.Version == Model2018
		request := buildReadFrame(frame.address, 0x01, withSER)
		traceID := c.traceRequest("read", request)
		started := time.Now()
		err = meterlink.WriteFrame(item, request, c.timeout())
		if err == nil {
			var response []byte
			response, err = meterlink.ReadFrameWithLength(item, c.timeout(), 512, 10)
			c.traceResponse(traceID, "read", response, time.Since(started), err)
			if err == nil {
				var payload []byte
				payload, err = parseReadFrame(response, frame.address, 0x81, withSER)
				if err == nil {
					var value any
					value, err = decodeDI(frame.address, payload, frame.variable.DataType)
					if err == nil {
						c.GetCollectionMeta().SetVariableValue(frame.variable.Name, collection.ApplyTransform(value, frame.variable))
					}
				}
			}
		} else {
			c.traceResponse(traceID, "read", nil, time.Since(started), err)
		}
		if err != nil {
			runtime.pool.Put(client, true)
			if collection.IsConnectionLost(err) || isTransportError(err) {
				return collection.ConnectionLost(err)
			}
			return err
		}
		runtime.pool.Put(client, false)
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

func (c *Collection) SetFrameTracer(tracer collection.FrameTracer) { c.frameTracer = tracer }

func (c *Collection) traceRequest(operation string, data []byte) uint64 {
	if c.frameTracer == nil {
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
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "timeout") || strings.Contains(message, "eof") || strings.Contains(message, "closed") || strings.Contains(message, "refused")
}
