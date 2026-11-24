package comli

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
)

type collectorRuntime struct {
	mu sync.Mutex

	pool     *connpool.Pool
	commands []*readCommand
	calcs    []*collection.Calculation
	calcErr  error

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	stamp   byte
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("COMLI collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Device) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: device is required", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	calcs, calcErr := collection.CompileCalculations(variables)
	runtime := &collectorRuntime{
		commands: buildReadCommands(variables),
		calcs:    calcs,
		calcErr:  calcErr,
		stamp:    '2',
	}
	c.runtime = runtime
	if len(runtime.commands) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	mode := serial.Mode{
		BaudRate: c.BaudRate,
		DataBits: c.DataBits,
		Parity:   serialParity(c.Parity),
		StopBits: serialStopBits(c.StopBits),
	}
	pool, err := newSerialPool(ctx, mode, c.Device)
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
	if len(runtime.commands) == 0 {
		runtime.mu.Unlock()
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	if runtime.pool == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("COMLI runtime is not connected")
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
	runtime.pool = nil
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if pool != nil {
		_ = pool.Close()
	}
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
	var cycleErr error
	for _, command := range runtime.commands {
		data, err := c.readCommand(ctx, command)
		if err != nil {
			if collection.IsConnectionLost(err) {
				return collection.MarkConnectionLost(c, err)
			}
			failed = true
			if cycleErr == nil {
				cycleErr = err
			}
			continue
		}
		for _, point := range command.points {
			value, err := decodePointValue(point, command, data)
			if err != nil {
				failed = true
				if cycleErr == nil {
					cycleErr = err
				}
				continue
			}
			c.GetCollectionMeta().SetVariableValue(point.variable.Name, collection.ApplyTransform(value, point.variable))
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
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
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

func (c *Collection) readCommand(ctx context.Context, command *readCommand) ([]byte, error) {
	attempts := c.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		stamp, response, err := c.exchangeRead(ctx, command)
		if err != nil {
			return nil, err
		}
		data, err := parseReadResponse(response, command.wireAddress(), byte(command.quantity()), stamp, messageReadRegisters)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("COMLI read request failed")
	}
	return nil, lastErr
}

func (c *Collection) exchangeRead(ctx context.Context, command *readCommand) (byte, []byte, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return 0, nil, collection.ConnectionLost(fmt.Errorf("COMLI runtime is not connected"))
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	stamp := runtime.nextStampLocked()
	request := buildReadRequest(command.identity, command.wireAddress(), byte(command.quantity()), stamp)
	traceID := c.traceRequest("read", request)
	started := time.Now()
	response, err := c.exchangeLocked(ctx, request, 13+command.quantity())
	c.traceResponse(traceID, "read", response, started, err)
	return stamp, response, err
}

func (c *Collection) writePoint(ctx context.Context, point *comliPoint, data []byte) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return collection.ConnectionLost(fmt.Errorf("COMLI runtime is not connected"))
	}
	messageType := messageWriteRegisters
	address := uint16((point.offset + 0x400) * 0x10)
	if point.area == areaBits {
		messageType = messageWriteBit
		address = point.offset
	}
	attempts := c.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		runtime.mu.Lock()
		stamp := runtime.nextStampLocked()
		request := buildWriteRequest(point.identity, messageType, address, stamp, data)
		traceID := c.traceRequest("write", request)
		started := time.Now()
		response, err := c.exchangeLocked(ctx, request, 8)
		c.traceResponse(traceID, "write", response, started, err)
		runtime.mu.Unlock()
		if err != nil {
			return err
		}
		if err := parseWriteAck(response, stamp); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("COMLI write request failed")
	}
	return lastErr
}

func (c *Collection) exchangeLocked(ctx context.Context, request []byte, responseLen int) ([]byte, error) {
	runtime := c.runtime
	client, err := runtime.pool.Get(ctx)
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	serialConn, ok := client.(*serialClient)
	if !ok {
		runtime.pool.Put(client, true)
		return nil, fmt.Errorf("invalid COMLI serial client")
	}
	serialConn.mu.Lock()
	defer serialConn.mu.Unlock()
	if _, err := serialConn.port.Write(request); err != nil {
		runtime.pool.Put(client, true)
		return nil, collection.ConnectionLost(err)
	}
	if err := serialConn.port.SetReadTimeout(time.Duration(c.TimeoutMs) * time.Millisecond); err != nil {
		runtime.pool.Put(client, true)
		return nil, collection.ConnectionLost(err)
	}
	response := make([]byte, responseLen)
	if err := collection.ReadFullWithTimeout(serialConn.port, response); err != nil {
		runtime.pool.Put(client, true)
		return nil, collection.ConnectionLost(err)
	}
	runtime.pool.Put(client, false)
	return response, nil
}

func (runtime *collectorRuntime) nextStampLocked() byte {
	switch runtime.stamp {
	case '1':
		runtime.stamp = '2'
	default:
		runtime.stamp = '1'
	}
	return runtime.stamp
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

func (c *Collection) traceResponse(id uint64, operation string, data []byte, started time.Time, err error) {
	if c.frameTracer == nil || id == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(id, operation, data, time.Since(started), err)
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		point, err := parseAddress(action.Variable)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		data, err := encodeWriteData(point, action.Value)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		if err := c.writePoint(ctx, point, data); err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func serialParity(value int) serial.Parity {
	switch value {
	case 1:
		return serial.OddParity
	case 2:
		return serial.EvenParity
	default:
		return serial.NoParity
	}
}

func serialStopBits(value int) serial.StopBits {
	if value == 2 {
		return serial.TwoStopBits
	}
	return serial.OneStopBit
}

var (
	_ collection.Collector     = (*Collection)(nil)
	_ collection.ActionHandler = (*Collection)(nil)
)
