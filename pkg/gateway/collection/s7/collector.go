package s7

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/utils/binutil"
)

const frameRetryCount = 3

type collectorRuntime struct {
	mu      sync.Mutex
	frames  []*s7Frame
	calcs   []*collection.Calculation
	pool    *connpool.Pool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

type CollectorFactory struct {
	model string
}

func (f *CollectorFactory) DriverType() string { return DriverType }
func (f *CollectorFactory) DriverModel() string {
	if f == nil || f.model == "" {
		return DriverModel
	}
	return f.model
}

func NewS7200CollectorFactory() *CollectorFactory {
	return &CollectorFactory{model: DriverModelS7200}
}

func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &S7Collection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *S7Collection) Connect(ctx context.Context) error {
	if c.runtime != nil {
		_ = c.Stop(ctx)
	}
	if c.PDUSize == 0 {
		c.PDUSize = DefaultPDUSize
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
	if c.TCPCount < 1 || c.TCPCount > MaxTCPCount {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: tcpCount must be between 1 and %d", collection.ErrInvalidConfig, MaxTCPCount)
	}
	if c.PDUSize != 480 && c.PDUSize != 960 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: pduSize must be 480 or 960", collection.ErrInvalidConfig)
	}
	var cotp []byte
	if c.GetDriverModel() == DriverModelS7200 {
		if c.LocalTSAP == 0 {
			c.LocalTSAP = DefaultLocalTSAP
		}
		if c.RemoteTSAP == 0 {
			c.RemoteTSAP = DefaultRemoteTSAP
		}
		cotp = newS7200COTPConnectMessage(c.LocalTSAP, c.RemoteTSAP)
	} else {
		if c.ConnectionType == 0 {
			c.ConnectionType = DefaultConnectionType
		}
		if c.ConnectionType < 1 || c.ConnectionType > 3 {
			c.SetCollectStatus(collection.CollectStatusError)
			return fmt.Errorf("%w: connectionType must be 1, 2 or 3", collection.ErrInvalidConfig)
		}
		cotp = newCOTPConnectMessage(c.Rack, c.Slot, c.ConnectionType)
	}
	variables := c.GetCollectionMeta().GetVariables()
	calculations, err := collection.CompileCalculations(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	pool, err := connpool.NewPool(ctx, c.TCPCount, func(ctx context.Context) (connpool.Client, error) {
		return newS7Client(ctx, c.endpoint(), cotp, c.PDUSize)
	})
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	client, err := pool.Get(ctx)
	if err != nil {
		_ = pool.Close()
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	pdu := client.(*s7Client).pdu
	pool.Put(client, false)
	frames, err := buildFrames(variables, pdu)
	if err != nil {
		_ = pool.Close()
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	runtime := &collectorRuntime{frames: frames, calcs: calculations, pool: pool}
	c.runtime = runtime
	if len(frames) == 0 {
		_ = pool.Close()
		runtime.pool = nil
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *S7Collection) Start(ctx context.Context) error {
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
		return fmt.Errorf("s7 runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
	runtime.wg.Add(1)
	runtime.mu.Unlock()
	go c.loop(runCtx)
	return nil
}

func (c *S7Collection) Stop(ctx context.Context) error {
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

func (c *S7Collection) loop(ctx context.Context) {
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

type frameResult struct {
	values []decodedValue
	err    error
}

func (c *S7Collection) poll(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return false
	}
	results := make(chan frameResult, len(runtime.frames))
	var wait sync.WaitGroup
	for _, frame := range runtime.frames {
		wait.Add(1)
		go func(item *s7Frame) {
			defer wait.Done()
			values, err := c.readFrame(ctx, item)
			results <- frameResult{values: values, err: err}
		}(frame)
	}
	wait.Wait()
	close(results)

	failed := false
	for result := range results {
		if result.err != nil {
			failed = true
			continue
		}
		for _, item := range result.values {
			if item.variable != nil {
				c.GetCollectionMeta().SetVariableValue(item.variable.Name, item.value)
			}
		}
	}
	if ctx.Err() != nil {
		return false
	}
	if failed {
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.SetCollectStatus(collection.CollectStatusCollecting)
	if err := collection.Calculate(c.GetCollectionMeta(), runtime.calcs); err != nil {
		c.SetCollectStatus(collection.CollectStatusCollectingError)
	}
	return true
}

// SetFrameTracer injects live frame tracing.
func (c *S7Collection) SetFrameTracer(tracer collection.FrameTracer) {
	c.frameTracer = tracer
}

func (c *S7Collection) readFrame(ctx context.Context, frame *s7Frame) ([]decodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return nil, fmt.Errorf("s7 runtime is not connected")
	}
	var lastErr error
	for attempt := 0; attempt < frameRetryCount; attempt++ {
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return nil, err
		}
		session := client.(*s7Client)
		traceID := c.traceRequest("read", frame.Request)
		started := time.Now()
		response, exchangeErr := session.ask(frame.Request)
		c.traceResponse(traceID, "read", response, started, exchangeErr)
		runtime.pool.Put(client, exchangeErr != nil)
		if exchangeErr != nil {
			lastErr = exchangeErr
			continue
		}
		payload, err := validateResponse(response, frame)
		if err != nil {
			lastErr = err
			continue
		}
		return decodeFrame(frame, payload)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("s7 frame failed")
	}
	return nil, lastErr
}

func validateResponse(response []byte, frame *s7Frame) ([]byte, error) {
	if len(response) < 21 {
		return nil, fmt.Errorf("s7 response is too short")
	}
	dataLength := int(binutil.ParseUint16(response[15:]))
	if dataLength != frame.DataLength {
		return nil, fmt.Errorf("s7 response data length mismatch")
	}
	if response[17] != 0 || response[18] != 0 {
		return nil, fmt.Errorf("s7 response error %d/%d", response[17], response[18])
	}
	payload := response[21:]
	if len(payload) < dataLength {
		return nil, fmt.Errorf("s7 response payload is too short")
	}
	offset := 0
	for index, item := range frame.Items {
		if offset+4 > len(payload) {
			return nil, fmt.Errorf("s7 response item %d header is too short", index)
		}
		if payload[offset] != 0xFF {
			return nil, fmt.Errorf("s7 response item %d failed", index)
		}
		if payload[offset+1] != 2 {
			return nil, fmt.Errorf("s7 response item %d has invalid transport %d", index, payload[offset+1])
		}
		lengthBits := int(binutil.ParseUint16(payload[offset+2:]))
		expectedBits := int(item.Length) * 8
		if lengthBits != expectedBits {
			return nil, fmt.Errorf("s7 response item %d length mismatch: got %d bits, want %d", index, lengthBits, expectedBits)
		}
		dataBytes := (lengthBits + 7) / 8
		if offset+4+dataBytes > len(payload) {
			return nil, fmt.Errorf("s7 response item %d data is too short", index)
		}
		offset += 4 + dataBytes
	}
	if offset != len(payload) {
		return nil, fmt.Errorf("s7 response payload length mismatch")
	}
	return payload, nil
}

func (c *S7Collection) endpoint() string {
	return fmt.Sprintf("%s:%d", c.Location, c.Port)
}

// DeliverAction writes collection variables through the S7 session pool.
func (c *S7Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
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
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return err
		}
		session := client.(*s7Client)
		traceID := c.traceRequest("write", request)
		started := time.Now()
		response, exchangeErr := session.ask(request)
		c.traceResponse(traceID, "write", response, started, exchangeErr)
		runtime.pool.Put(client, exchangeErr != nil)
		if exchangeErr != nil {
			return exchangeErr
		}
		if len(response) < 22 || response[21] != 255 {
			return fmt.Errorf("s7 write failed")
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func (c *S7Collection) traceRequest(operation string, data []byte) uint64 {
	if c.frameTracer == nil || len(data) == 0 {
		return 0
	}
	traceID := c.frameTracer.NextTraceID()
	c.frameTracer.TraceRequest(traceID, operation, data)
	return traceID
}

func (c *S7Collection) traceResponse(traceID uint64, operation string, data []byte, started time.Time, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, time.Since(started), err)
}

var _ collection.Collector = (*S7Collection)(nil)
var _ collection.CollectorFactory = (*CollectorFactory)(nil)
