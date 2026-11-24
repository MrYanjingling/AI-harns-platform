package fx

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

	pool    *connpool.Pool
	points  []*fxPoint
	calcs   []*collection.Calculation
	calcErr error
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("FX collector is nil")
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
	run := &collectorRuntime{points: buildPoints(variables), calcs: calcs, calcErr: calcErr}
	c.runtime = run
	if len(run.points) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	mode := serial.Mode{BaudRate: c.BaudRate, DataBits: c.DataBits, Parity: serialParity(c.Parity), StopBits: serialStopBits(c.StopBits)}
	pool, err := newSerialPool(ctx, mode, c.Device)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	run.pool = pool
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func buildPoints(variables []*collection.Variable) []*fxPoint {
	points := make([]*fxPoint, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || strings.TrimSpace(variable.Address) == "" {
			continue
		}
		point, err := parseAddress(variable)
		if err != nil {
			continue
		}
		points = append(points, point)
	}
	return points
}

func (c *Collection) Start(ctx context.Context) error {
	run := c.runtime
	if run == nil {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	run.mu.Lock()
	if run.started {
		run.mu.Unlock()
		return nil
	}
	if len(run.points) == 0 {
		run.mu.Unlock()
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	if run.pool == nil {
		run.mu.Unlock()
		return fmt.Errorf("FX runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	run.cancel = cancel
	run.started = true
	run.wg.Add(1)
	run.mu.Unlock()
	go c.loop(runCtx)
	return nil
}

func (c *Collection) Stop(ctx context.Context) error {
	run := c.runtime
	if run == nil {
		c.SetCollectStatus(collection.CollectStatusStopped)
		return nil
	}
	run.mu.Lock()
	cancel := run.cancel
	run.cancel = nil
	run.started = false
	pool := run.pool
	run.pool = nil
	run.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	run.wg.Wait()
	if pool != nil {
		_ = pool.Close()
	}
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) loop(ctx context.Context) {
	run := c.runtime
	if run == nil {
		return
	}
	defer run.wg.Done()
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
	run := c.runtime
	if run == nil || run.pool == nil {
		return false
	}
	failed := false
	var cycleErr error
	for _, point := range run.points {
		value, err := c.readPoint(ctx, point)
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
		c.GetCollectionMeta().SetVariableValue(point.variable.Name, collection.ApplyTransform(value, point.variable))
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
	if run.calcErr != nil {
		c.GetCollectionMeta().ReportError(run.calcErr)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	if err := collection.Calculate(c.GetCollectionMeta(), run.calcs); err != nil {
		c.GetCollectionMeta().ReportError(err)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.GetCollectionMeta().ReportError(nil)
	return true
}

func (c *Collection) readPoint(ctx context.Context, point *fxPoint) (any, error) {
	if point.bitDevice {
		data, err := c.readBytes(ctx, point.byteAddr, 1)
		if err != nil {
			return nil, err
		}
		return data[0]&(1<<point.bit) != 0, nil
	}
	data, err := c.readBytes(ctx, point.wordStart, point.bytes)
	if err != nil {
		return nil, err
	}
	if point.wordBit >= 0 {
		if len(data) < 2 {
			return nil, fmt.Errorf("FX word bit read is too short")
		}
		word := uint16(data[0]) | uint16(data[1])<<8
		return word&(1<<point.wordBit) != 0, nil
	}
	return decodeWord(data, point.dataType)
}

func (c *Collection) readBytes(ctx context.Context, start uint16, bytes int) ([]byte, error) {
	attempts := c.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		request := buildReadRequest(start, bytes)
		response, err := c.exchange(ctx, request, 2*bytes+4)
		if err != nil {
			return nil, err
		}
		data, err := parseReadResponse(response, bytes)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("FX read failed")
	}
	return nil, lastErr
}

func (c *Collection) writeBytes(ctx context.Context, start uint16, data []byte) error {
	attempts := c.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		request := buildWriteRequest(start, data)
		response, err := c.exchange(ctx, request, 1)
		if err != nil {
			return err
		}
		if err := parseWriteAck(response); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("FX write failed")
	}
	return lastErr
}

func (c *Collection) writeBit(ctx context.Context, point *fxPoint, value bool) error {
	attempts := c.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		request := buildBitRequest(value, point.bitAddr)
		response, err := c.exchange(ctx, request, 1)
		if err != nil {
			return err
		}
		if err := parseWriteAck(response); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("FX bit write failed")
	}
	return lastErr
}

func (c *Collection) exchange(ctx context.Context, request []byte, responseLen int) ([]byte, error) {
	run := c.runtime
	if run == nil || run.pool == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("FX runtime is not connected"))
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	client, err := run.pool.Get(ctx)
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	serialConn, ok := client.(*serialClient)
	if !ok {
		run.pool.Put(client, true)
		return nil, fmt.Errorf("invalid FX serial client")
	}
	serialConn.mu.Lock()
	defer serialConn.mu.Unlock()
	if _, err := serialConn.port.Write(request); err != nil {
		run.pool.Put(client, true)
		return nil, collection.ConnectionLost(err)
	}
	if err := serialConn.port.SetReadTimeout(time.Duration(c.TimeoutMs) * time.Millisecond); err != nil {
		run.pool.Put(client, true)
		return nil, collection.ConnectionLost(err)
	}
	response := make([]byte, responseLen)
	if err := collection.ReadFullWithTimeout(serialConn.port, response); err != nil {
		run.pool.Put(client, true)
		return nil, collection.ConnectionLost(err)
	}
	run.pool.Put(client, false)
	return response, nil
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	run := c.runtime
	if run == nil || run.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		point, err := parseAddress(action.Variable)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
		}
		value, err := valueAsBool(action.Value)
		if point.bitDevice {
			if err != nil {
				return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
			}
			if err := c.writeBit(ctx, point, value); err != nil {
				return collection.DeviceActionError(action, err)
			}
		} else if point.wordBit >= 0 {
			if err != nil {
				return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
			}
			data, err := c.readBytes(ctx, point.wordStart, 2)
			if err != nil {
				return collection.DeviceActionError(action, err)
			}
			word := uint16(data[0]) | uint16(data[1])<<8
			if value {
				word |= 1 << point.wordBit
			} else {
				word &^= 1 << point.wordBit
			}
			if err := c.writeBytes(ctx, point.wordStart, []byte{byte(word), byte(word >> 8)}); err != nil {
				return collection.DeviceActionError(action, err)
			}
		} else {
			data, err := encodeWord(point, action.Value)
			if err != nil {
				return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
			}
			if err := c.writeBytes(ctx, point.wordStart, data); err != nil {
				return collection.DeviceActionError(action, err)
			}
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

func (c *Collection) SetFrameTracer(tracer collection.FrameTracer) { c.frameTracer = tracer }

var (
	_ collection.Collector     = (*Collection)(nil)
	_ collection.ActionHandler = (*Collection)(nil)
)
