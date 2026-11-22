package hostlinkcmode

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

const frameRetryCount = 3

type serialPort interface {
	io.ReadWriteCloser
	SetReadTimeout(time.Duration) error
}

var openSerialPort = func(location string, mode *serial.Mode) (serialPort, error) {
	return serial.Open(location, mode)
}

type collectorRuntime struct {
	mu      sync.Mutex
	frames  []*hostlinkFrame
	calcs   []*collection.Calculation
	calcErr error
	port    serialPort

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("hostlink collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Location) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: location is required", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	frames, err := buildReadFrames(variables, c.MaxReadWords)
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
	if err := ctx.Err(); err != nil {
		return err
	}
	port, err := openSerialPort(c.Location, &serial.Mode{
		BaudRate: c.BaudRate,
		DataBits: c.DataBits,
		Parity:   parseParity(c.Parity),
		StopBits: parseStopBits(c.StopBits),
	})
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.port = port
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
	if runtime.port == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("hostlink serial port is not connected")
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
	port := runtime.port
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if port != nil {
		_ = port.Close()
	}
	runtime.mu.Lock()
	if runtime.port == port {
		runtime.port = nil
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
	if runtime == nil || runtime.port == nil {
		return false
	}
	failed := false
	succeeded := 0
	connectionFailures := 0
	protocolFailure := false
	var cycleErr error
	for _, frame := range runtime.frames {
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
			} else {
				protocolFailure = true
			}
		}
		if c.IntervalMs > 0 {
			select {
			case <-ctx.Done():
				return false
			case <-time.After(time.Duration(c.IntervalMs) * time.Millisecond):
			}
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

func (c *Collection) readFrame(ctx context.Context, frame *hostlinkFrame) ([]decodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.port == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("hostlink runtime is not connected"))
	}
	var lastErr error
	for attempt := 0; attempt < frameRetryCount; attempt++ {
		traceID := c.traceRequest("read", frame.Request)
		started := time.Now()
		raw, err := c.exchange(ctx, frame.Request)
		c.traceResponse(traceID, "read", raw, started, err)
		if err != nil {
			lastErr = err
			continue
		}
		parsed, err := parseResponse(raw, frame.Address)
		if err != nil {
			lastErr = err
			continue
		}
		values, err := decodeFrame(frame, parsed)
		if err != nil {
			return values, err
		}
		return values, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("hostlink frame failed")
	}
	return nil, lastErr
}

func (c *Collection) exchange(ctx context.Context, request []byte) ([]byte, error) {
	runtime := c.runtime
	if runtime == nil || runtime.port == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("hostlink runtime is not connected"))
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := runtime.port.Write(request); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if err := runtime.port.SetReadTimeout(time.Duration(c.TimeoutMs) * time.Millisecond); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	response := make([]byte, 0, 64)
	buffer := make([]byte, 1)
	for len(response) < 4096 {
		read, err := runtime.port.Read(buffer)
		if err != nil {
			return nil, collection.ConnectionLost(err)
		}
		if read == 0 {
			return nil, collection.ConnectionLost(collection.ErrReadTimeout)
		}
		response = append(response, buffer[0])
		if buffer[0] == '\r' {
			return response, nil
		}
	}
	return nil, fmt.Errorf("hostlink response is too long")
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
	if runtime == nil || runtime.port == nil {
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
		data, err := encodeResponseData(action.Value, address, action.Variable.DataType)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		request := buildWriteRequest(address, data)
		traceID := c.traceRequest("write", request)
		started := time.Now()
		response, exchangeErr := c.exchange(ctx, request)
		c.traceResponse(traceID, "write", response, started, exchangeErr)
		if exchangeErr != nil {
			return collection.DeviceActionError(action, exchangeErr)
		}
		if _, err := parseResponse(response, address); err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func parseParity(value string) serial.Parity {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "odd", "oddparity":
		return serial.OddParity
	case "even", "evenparity":
		return serial.EvenParity
	default:
		return serial.NoParity
	}
}

func parseStopBits(value string) serial.StopBits {
	switch strings.TrimSpace(value) {
	case "1.5":
		return serial.OnePointFiveStopBits
	case "2":
		return serial.TwoStopBits
	default:
		return serial.OneStopBit
	}
}

var _ collection.Collector = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
