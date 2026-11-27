package ethernetip

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

type cipFrame struct {
	Variable *collection.Variable
	Address  parsedAddress
}

type collectorRuntime struct {
	mu      sync.Mutex
	frames  []*cipFrame
	calcs   []*collection.Calculation
	calcErr error
	pool    *connpool.Pool

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("ethernet/ip collector is nil")
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
	if c.Slot < 0 || c.Slot > 255 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: slot must be between 0 and 255", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	frames, err := buildReadFrames(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calculations, calcErr := collection.CompileCalculations(variables)
	runtime := &collectorRuntime{frames: frames, calcs: calculations, calcErr: calcErr}
	c.runtime = runtime
	if len(frames) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	// Neuron keeps one class-3 connection per driver and pipelines requests on
	// it. A pool of one preserves that model while reconnecting after a lost
	// session.
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
		return fmt.Errorf("ethernet/ip runtime is not connected")
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
		value, err := c.readFrame(ctx, frame)
		if err != nil {
			failed = true
			if cycleErr == nil {
				cycleErr = err
			}
			if collection.IsConnectionLost(err) {
				connectionFailures++
			} else {
				protocolFailure = true
			}
			continue
		}
		succeeded++
		c.GetCollectionMeta().SetVariableValue(frame.Variable.Name, value)
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

func (c *Collection) readFrame(ctx context.Context, frame *cipFrame) (any, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("ethernet/ip runtime is not connected"))
	}
	client, err := runtime.pool.Get(ctx)
	if err != nil {
		return nil, err
	}
	session, ok := client.(*cipClient)
	if !ok {
		runtime.pool.Put(client, true)
		return nil, fmt.Errorf("invalid ethernet/ip client type")
	}
	data, wireRequest, wireResponse, err := session.ReadTag(ctx, frame.Address)
	traceID := c.traceRequest("read", wireRequest)
	c.traceResponse(traceID, "read", wireResponse, err)
	runtime.pool.Put(client, err != nil && collection.IsConnectionLost(err))
	if err != nil {
		if collection.IsConnectionLost(err) {
			return nil, collection.ConnectionLost(err)
		}
		return nil, fmt.Errorf("variable %s: %w", frame.Variable.Name, err)
	}
	value, err := decodeValue(data, frame.Variable.DataType)
	if err != nil {
		return nil, fmt.Errorf("variable %s: %w", frame.Variable.Name, err)
	}
	return collection.ApplyTransform(value, frame.Variable), nil
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
		data, err := encodeValue(action.Value, action.Variable.DataType)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		err = runtime.pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
			session, ok := client.(*cipClient)
			if !ok {
				_ = client.Close()
				return false, fmt.Errorf("invalid ethernet/ip client type")
			}
			wireRequest, wireResponse, writeErr := session.WriteTag(ctx, address, data)
			traceID := c.traceRequest("write", wireRequest)
			c.traceResponse(traceID, "write", wireResponse, writeErr)
			if writeErr != nil {
				return collection.IsConnectionLost(writeErr), writeErr
			}
			return false, nil
		})
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func buildReadFrames(variables []*collection.Variable) ([]*cipFrame, error) {
	frames := make([]*cipFrame, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := parseAddress(variable.Address, variable.DataType)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		frames = append(frames, &cipFrame{Variable: variable, Address: address})
	}
	return frames, nil
}

var _ collection.Collector = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
