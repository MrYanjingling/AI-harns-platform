package fins

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

const frameRetryCount = 3

type collectorRuntime struct {
	mu        sync.Mutex
	frames    []*readFrame
	calcs     []*collection.Calculation
	transport transport
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	started   bool
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("fins collector is nil")
	}
	if c.runtime != nil {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Location) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: location is required", collection.ErrInvalidConfig)
	}
	if c.Transport != TransportTCP && c.Transport != TransportUDP {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: transport must be tcp or udp", collection.ErrInvalidConfig)
	}
	if c.Type < 0 || c.Type > 7 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: type must be between 0 and 7", collection.ErrInvalidConfig)
	}
	if c.MaxSize < MinMaxSize || c.MaxSize > MaxMaxSize {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: maxSize must be between %d and %d", collection.ErrInvalidConfig, MinMaxSize, MaxMaxSize)
	}
	if c.Expires <= 0 {
		c.Expires = DefaultExpiresMs
	}
	variables := c.GetCollectionMeta().GetVariables()
	frames, err := buildReadFrames(variables, c.MaxSize)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calculations, err := collection.CompileCalculations(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	runtime := &collectorRuntime{frames: frames, calcs: calculations}
	c.runtime = runtime
	if len(frames) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	timeout := time.Duration(c.Expires) * time.Millisecond
	var conn transport
	if c.Transport == TransportUDP {
		conn, err = newUDPTransport(ctx, c.Location, c.Port, timeout)
	} else {
		conn, err = newTCPTransport(ctx, c.endpoint(), timeout)
	}
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.transport = conn
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
	if runtime.transport == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("fins runtime is not connected")
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
	conn := runtime.transport
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if conn != nil {
		_ = conn.Close()
	}
	runtime.mu.Lock()
	if runtime.transport == conn {
		runtime.transport = nil
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
	if runtime == nil || runtime.transport == nil {
		return false
	}
	failed := false
	var cycleErr error
	for _, frame := range runtime.frames {
		values, err := c.readFrame(ctx, frame)
		if err != nil {
			failed = true
			if cycleErr == nil {
				cycleErr = err
			}
			continue
		}
		for _, item := range values {
			if item.Variable != nil {
				c.GetCollectionMeta().SetVariableValue(item.Variable.Name, item.Value)
			}
		}
	}
	if ctx.Err() != nil {
		return false
	}
	if failed {
		c.GetCollectionMeta().ReportError(cycleErr)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.SetCollectStatus(collection.CollectStatusCollecting)
	if err := collection.Calculate(c.GetCollectionMeta(), runtime.calcs); err != nil {
		c.GetCollectionMeta().ReportError(err)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.GetCollectionMeta().ReportError(nil)
	return true
}

func (c *Collection) readFrame(ctx context.Context, frame *readFrame) ([]decodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.transport == nil {
		return nil, fmt.Errorf("fins runtime is not connected")
	}
	var lastErr error
	for attempt := 0; attempt < frameRetryCount; attempt++ {
		traceID := c.traceRequest("read", frame.Request)
		started := time.Now()
		response, err := runtime.transport.Exchange(ctx, frame.Request)
		c.traceResponse(traceID, "read", response, started, err)
		if err != nil {
			lastErr = err
			continue
		}
		payload, err := validateReadResponse(response, frame)
		if err != nil {
			lastErr = err
			continue
		}
		return decodeFrame(frame, payload)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("fins frame failed")
	}
	return nil, lastErr
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

func (c *Collection) traceResponse(traceID uint64, operation string, data []byte, started time.Time, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, time.Since(started), err)
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.transport == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		request, err := buildWriteRequest(action.Variable, action.Value)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		traceID := c.traceRequest("write", request)
		started := time.Now()
		response, exchangeErr := runtime.transport.Exchange(ctx, request)
		c.traceResponse(traceID, "write", response, started, exchangeErr)
		if exchangeErr != nil {
			return exchangeErr
		}
		if err := validateWriteResponse(response); err != nil {
			return err
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func buildWriteRequest(variable *collection.Variable, value any) ([]byte, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	parsed, err := parseAddress(variable.Address, variable.DataType, variable.BitsValue())
	if err != nil {
		return nil, err
	}
	if parsed.stringLength > 0 {
		return nil, fmt.Errorf("fins string write is not supported")
	}
	data, err := encodeValue(value, variable.DataType, parsed)
	if err != nil {
		return nil, err
	}
	count := parsed.length
	if parsed.isBit {
		count = 1
	}
	return newWriteRequest(parsed.areaCode(), parsed.start, parsed.bit, count, data), nil
}

var _ collection.Collector = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
