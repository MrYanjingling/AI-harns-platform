package ethernetip1400

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
)

const frameRetryCount = 3

type collectorRuntime struct {
	mu      sync.Mutex
	frames  []*pcccFrame
	calcs   []*collection.Calculation
	calcErr error
	pool    *connpool.Pool

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("ethernet/ip 1400 collector is nil")
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
	variables := c.GetCollectionMeta().GetVariables()
	frames, err := buildReadFrames(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calculations, calcErr := collection.CompileCalculations(variables)
	runtime := &collectorRuntime{
		frames:  frames,
		calcs:   calculations,
		calcErr: calcErr,
	}
	c.runtime = runtime
	if len(frames) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		return newCIPClient(ctx, c.endpoint(), cipTimeout)
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
	if len(runtime.frames) == 0 {
		runtime.mu.Unlock()
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	if runtime.pool == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("ethernet/ip 1400 runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
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
	cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
	if cycle <= 0 {
		cycle = time.Second
	}
	for {
		started := time.Now()
		if !c.poll(ctx) {
			return
		}
		delay := cycle - time.Since(started)
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (c *Collection) poll(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return false
	}
	failed := false
	succeeded := 0
	connectionFailures := 0
	protocolFailure := false
	var cycleErr error
	for _, frame := range runtime.frames {
		if ctx.Err() != nil {
			return false
		}
		values, err := c.readFrame(ctx, frame)
		for _, item := range values {
			if item.Variable != nil {
				c.GetCollectionMeta().SetVariableValue(item.Variable.Name, item.Value)
			}
		}
		if err == nil || len(values) > 0 {
			succeeded++
		}
		if err != nil {
			failed = true
			if cycleErr == nil {
				cycleErr = err
			}
			if collection.IsConnectionLost(err) {
				connectionFailures++
				c.GetCollectionMeta().ReportError(cycleErr)
				return collection.MarkConnectionLost(c, cycleErr)
			}
			protocolFailure = true
		}
	}
	if ctx.Err() != nil {
		return false
	}
	if failed {
		c.GetCollectionMeta().ReportError(cycleErr)
		return collection.ApplyCycleFailure(c, cycleErr, succeeded, connectionFailures, protocolFailure)
	}
	c.SetCollectStatus(collection.CollectStatusCollecting)
	if runtime.calcErr != nil {
		c.GetCollectionMeta().ReportError(runtime.calcErr)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	if err := collection.Calculate(c.GetCollectionMeta(), runtime.calcs); err != nil {
		c.GetCollectionMeta().ReportError(err)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.GetCollectionMeta().ReportError(nil)
	return true
}

func (c *Collection) readFrame(ctx context.Context, frame *pcccFrame) ([]decodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("ethernet/ip 1400 runtime is not connected"))
	}
	var lastErr error
	for attempt := 0; attempt < frameRetryCount; attempt++ {
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return nil, collection.ConnectionLost(err)
		}
		session, ok := client.(*cipClient)
		if !ok {
			runtime.pool.Put(client, true)
			return nil, fmt.Errorf("invalid ethernet/ip 1400 client type")
		}
		response, wireRequest, wireResponse, err := session.Exchange(ctx, frame.Request)
		traceID := c.traceRequest("read", wireRequest)
		c.traceResponse(traceID, "read", wireResponse, err)
		runtime.pool.Put(client, err != nil)
		if err != nil {
			lastErr = collection.ConnectionLost(err)
			continue
		}
		values, err := decodeFrame(frame, response)
		if err != nil {
			return values, err
		}
		return values, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ethernet/ip 1400 frame failed")
	}
	return nil, lastErr
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		address, err := parseAddress(action.Variable.Address, action.Variable.DataType)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		if isTimerCounter(address) && address.SubElement == subElementACC {
			if err := c.writeTimerCounterAction(ctx, runtime, action, address); err != nil {
				return err
			}
			continue
		}
		request, err := writeRequest(address, action.Value, action.Variable.DataType)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		session, ok := client.(*cipClient)
		if !ok {
			runtime.pool.Put(client, true)
			return collection.DeviceActionError(action, fmt.Errorf("invalid ethernet/ip 1400 client type"))
		}
		response, wireRequest, wireResponse, exchangeErr := session.Exchange(ctx, request)
		traceID := c.traceRequest("write", wireRequest)
		c.traceResponse(traceID, "write", wireResponse, exchangeErr)
		runtime.pool.Put(client, exchangeErr != nil)
		if exchangeErr != nil {
			return collection.DeviceActionError(action, exchangeErr)
		}
		if err := parseWriteResponse(response); err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func (c *Collection) writeTimerCounterAction(ctx context.Context, runtime *collectorRuntime, action collection.ActionValue, address pcccAddress) error {
	readRequest, err := readRequest(address, action.Variable.DataType)
	if err != nil {
		return actionNotWritable(action)
	}
	client, err := runtime.pool.Get(ctx)
	if err != nil {
		return collection.DeviceActionError(action, err)
	}
	session, ok := client.(*cipClient)
	if !ok {
		runtime.pool.Put(client, true)
		return collection.DeviceActionError(action, fmt.Errorf("invalid ethernet/ip 1400 client type"))
	}
	readResponse, wireRequest, wireResponse, readErr := session.Exchange(ctx, readRequest)
	traceID := c.traceRequest("read", wireRequest)
	c.traceResponse(traceID, "read", wireResponse, readErr)
	if readErr != nil {
		runtime.pool.Put(client, true)
		return collection.DeviceActionError(action, readErr)
	}
	current, err := parsePCCCResponse(readResponse)
	if err != nil {
		runtime.pool.Put(client, false)
		return collection.DeviceActionError(action, err)
	}
	request, err := timerCounterWriteRequest(address, current, action.Value, action.Variable.DataType)
	if err != nil {
		runtime.pool.Put(client, false)
		return actionNotWritable(action)
	}
	writeResponse, wireRequest, wireResponse, writeErr := session.Exchange(ctx, request)
	traceID = c.traceRequest("write", wireRequest)
	c.traceResponse(traceID, "write", wireResponse, writeErr)
	runtime.pool.Put(client, writeErr != nil)
	if writeErr != nil {
		return collection.DeviceActionError(action, writeErr)
	}
	if err := parseWriteResponse(writeResponse); err != nil {
		return collection.DeviceActionError(action, err)
	}
	c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	return nil
}

func actionNotWritable(action collection.ActionValue) error {
	return validation.New([]validation.FieldIssue{{
		Field: fmt.Sprintf("actions[%d].name", action.Index),
		MsgID: "validate.action_variable_not_writable",
		Args:  []interface{}{action.Variable.Name},
	}})
}

func (c *Collection) endpoint() string {
	return fmt.Sprintf("%s:%d", c.Location, c.Port)
}

func (c *Collection) SetFrameTracer(tracer collection.FrameTracer) {
	c.frameTracer = tracer
}

func (c *Collection) traceRequest(operation string, data []byte) uint64 {
	if c.frameTracer == nil || len(data) == 0 {
		return 0
	}
	traceID := c.frameTracer.NextTraceID()
	c.frameTracer.TraceRequest(traceID, operation, data)
	return traceID
}

func (c *Collection) traceResponse(traceID uint64, operation string, data []byte, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, 0, err)
}

var _ collection.Collector = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
