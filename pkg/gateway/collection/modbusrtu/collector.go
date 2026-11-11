package modbusrtu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/collection/modbus"
	"harns-platform/pkg/gateway/validation"
)

const (
	frameRetryCount = 3
	serialTimeout   = time.Second
)

type wireFrame struct {
	frame       *modbus.Frame
	request     []byte
	responseLen int
}

type collectorRuntime struct {
	mu     sync.Mutex
	frames []*wireFrame
	calcs  []*collection.Calculation
	pool   *connpool.Pool

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

type CollectorFactory struct{}

func (f *CollectorFactory) DriverType() string  { return DriverType }
func (f *CollectorFactory) DriverModel() string { return DriverModel }

func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &ModbusRtuCollection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *ModbusRtuCollection) Connect(ctx context.Context) error {
	if c.runtime != nil {
		_ = c.Stop(ctx)
	}
	layout := strings.ToUpper(strings.TrimSpace(c.MemoryLayout))
	switch layout {
	case "ABCD", "BADC", "CDAB", "DCBA":
		c.MemoryLayout = layout
	default:
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: memoryLayout must be ABCD, BADC, CDAB or DCBA", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	frames, err := modbus.BuildFrames(variables, c.PositionAddress)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calculations, err := collection.CompileCalculations(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	wireFrames := buildWireFrames(frames)
	runtime := &collectorRuntime{frames: wireFrames, calcs: calculations}
	c.runtime = runtime
	if len(wireFrames) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := newSerialPool(ctx, serial.Mode{
		BaudRate: c.BaudRate,
		DataBits: c.DataBits,
		Parity:   parseParity(c.Parity),
		StopBits: parseStopBits(c.StopBits),
	}, c.Location)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.pool = pool
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *ModbusRtuCollection) Start(ctx context.Context) error {
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
		return fmt.Errorf("modbusRtu serial port is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
	runtime.wg.Add(1)
	runtime.mu.Unlock()

	go c.loop(runCtx)
	return nil
}

func (c *ModbusRtuCollection) Stop(ctx context.Context) error {
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

func (c *ModbusRtuCollection) loop(ctx context.Context) {
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

func (c *ModbusRtuCollection) poll(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return false
	}
	failed := false
	for _, frame := range runtime.frames {
		values, err := c.collectFrame(ctx, frame)
		if err != nil {
			failed = true
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
func (c *ModbusRtuCollection) SetFrameTracer(tracer collection.FrameTracer) {
	c.frameTracer = tracer
}

func (c *ModbusRtuCollection) collectFrame(ctx context.Context, frame *wireFrame) ([]modbus.DecodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return nil, fmt.Errorf("modbusRtu runtime is not connected")
	}
	var lastErr error
	for attempt := 0; attempt < frameRetryCount; attempt++ {
		response := make([]byte, frame.responseLen)
		traceID := c.traceRequest("read", frame.request)
		started := time.Now()
		exchangeErr := c.exchange(ctx, runtime, frame.request, response)
		c.traceResponse(traceID, "read", response, started, exchangeErr)
		if exchangeErr != nil {
			lastErr = exchangeErr
			continue
		}
		payload, err := modbus.ValidateRTUResponse(response, frame.frame)
		if err != nil {
			lastErr = err
			continue
		}
		return modbus.DecodeFrame(frame.frame, payload, c.MemoryLayout)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("modbusRtu frame failed")
	}
	return nil, lastErr
}

func (c *ModbusRtuCollection) exchange(ctx context.Context, runtime *collectorRuntime, request, response []byte) error {
	client, err := runtime.pool.Get(ctx)
	if err != nil {
		return err
	}
	serialConn := client.(*serialClient)
	serialConn.mu.Lock()
	_, err = serialConn.port.Write(request)
	if err == nil {
		if err = serialConn.port.SetReadTimeout(serialTimeout); err == nil {
			_, err = io.ReadFull(serialConn.port, response)
		}
	}
	serialConn.mu.Unlock()
	runtime.pool.Put(client, err != nil)
	return err
}

func buildWireFrames(frames []*modbus.Frame) []*wireFrame {
	result := make([]*wireFrame, 0, len(frames))
	for _, frame := range frames {
		result = append(result, &wireFrame{
			frame:       frame,
			request:     modbus.BuildRTURequest(frame),
			responseLen: modbus.RTUResponseLength(frame),
		})
	}
	return result
}

func parseParity(value string) serial.Parity {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "oddparity":
		return serial.OddParity
	case "evenparity":
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

// DeliverAction writes collection variables through the serial port.
func (c *ModbusRtuCollection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		write, err := modbus.BuildWrite(action.Variable, action.Value, c.PositionAddress, c.MemoryLayout)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		request := modbus.BuildRTUWriteRequest(write)
		traceID := c.traceRequest("write", request)
		response := make([]byte, 8)
		started := time.Now()
		exchangeErr := c.exchange(ctx, runtime, request, response)
		c.traceResponse(traceID, "write", response, started, exchangeErr)
		if exchangeErr != nil {
			return exchangeErr
		}
		if err := modbus.ValidateRTUWriteResponse(response, write.Slave, write.Function); err != nil {
			return err
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func (c *ModbusRtuCollection) traceRequest(operation string, data []byte) uint64 {
	if c.frameTracer == nil || len(data) == 0 {
		return 0
	}
	traceID := c.frameTracer.NextTraceID()
	c.frameTracer.TraceRequest(traceID, operation, data)
	return traceID
}

func (c *ModbusRtuCollection) traceResponse(traceID uint64, operation string, data []byte, started time.Time, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, time.Since(started), err)
}

var _ collection.CollectorFactory = (*CollectorFactory)(nil)
