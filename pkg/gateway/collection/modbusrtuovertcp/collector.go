package modbusrtuovertcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbus"
	"harns-platform/pkg/gateway/validation"
)

var (
	backupHoldDuration    = 5 * time.Minute
	failbackProbeInterval = 5 * time.Minute
)

const (
	failbackSuccessCount = 3
	frameRetryCount      = 3
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
	pool   *tcpPool

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool

	failureStreak int
	backupSince   time.Time
	probeSuccess  int
	lastProbe     time.Time
}

type CollectorFactory struct{}

func (f *CollectorFactory) DriverType() string  { return DriverType }
func (f *CollectorFactory) DriverModel() string { return DriverModel }

func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &ModbusRtuOverTcpCollection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *ModbusRtuOverTcpCollection) Connect(ctx context.Context) error {
	if c.runtime != nil {
		_ = c.Stop(ctx)
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
	if c.TCPCount < 1 || c.TCPCount > MaxTCPCount {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: tcpCount must be between 1 and %d", collection.ErrInvalidConfig, MaxTCPCount)
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
	wireFrames := make([]*wireFrame, 0, len(frames))
	for _, frame := range frames {
		wireFrames = append(wireFrames, &wireFrame{
			frame:       frame,
			request:     modbus.BuildRTURequest(frame),
			responseLen: modbus.RTUResponseLength(frame),
		})
	}
	runtime := &collectorRuntime{frames: wireFrames, calcs: calculations}
	c.runtime = runtime
	if len(wireFrames) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool := newTCPPool(c.Location, c.BackupLocation, c.TCPCount)
	if c.activeBackup {
		pool.preferBackup()
	}
	if err := pool.connect(ctx); err != nil {
		c.activeBackup = pool.isBackup()
		pool.close()
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.pool = pool
	c.activeBackup = pool.isBackup()
	if pool.isBackup() {
		runtime.backupSince = time.Now()
	}
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *ModbusRtuOverTcpCollection) Start(ctx context.Context) error {
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
		return fmt.Errorf("modbusRtuOverTcp runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
	runtime.wg.Add(1)
	runtime.mu.Unlock()

	go c.loop(runCtx)
	return nil
}

func (c *ModbusRtuOverTcpCollection) Stop(ctx context.Context) error {
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
		pool.close()
	}
	runtime.mu.Lock()
	if runtime.pool == pool {
		runtime.pool = nil
	}
	runtime.mu.Unlock()
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *ModbusRtuOverTcpCollection) loop(ctx context.Context) {
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
	values []modbus.DecodedValue
	err    error
}

func (c *ModbusRtuOverTcpCollection) poll(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return false
	}
	results := make(chan frameResult, len(runtime.frames))
	var wait sync.WaitGroup
	for _, frame := range runtime.frames {
		wait.Add(1)
		go func(item *wireFrame) {
			defer wait.Done()
			values, err := c.collectFrame(ctx, item)
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
		c.recordCycleFailure(ctx)
		return true
	}
	c.SetCollectStatus(collection.CollectStatusCollecting)
	runtime.mu.Lock()
	runtime.failureStreak = 0
	runtime.mu.Unlock()
	if err := collection.Calculate(c.GetCollectionMeta(), runtime.calcs); err != nil {
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.maybeFailback(ctx)
	return true
}

// SetFrameTracer injects live frame tracing.
func (c *ModbusRtuOverTcpCollection) SetFrameTracer(tracer collection.FrameTracer) {
	c.frameTracer = tracer
}

func (c *ModbusRtuOverTcpCollection) collectFrame(ctx context.Context, frame *wireFrame) ([]modbus.DecodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return nil, fmt.Errorf("modbusRtuOverTcp runtime is not connected")
	}
	var lastErr error
	for attempt := 0; attempt < frameRetryCount; attempt++ {
		response := make([]byte, frame.responseLen)
		traceID := c.traceRequest("read", frame.request)
		started := time.Now()
		exchangeErr := runtime.pool.exchange(ctx, frame.request, response)
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
		lastErr = fmt.Errorf("modbusRtuOverTcp frame failed")
	}
	return nil, lastErr
}

func (c *ModbusRtuOverTcpCollection) recordCycleFailure(ctx context.Context) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return
	}
	runtime.mu.Lock()
	runtime.failureStreak++
	streak := runtime.failureStreak
	activeBackup := runtime.pool.isBackup()
	runtime.mu.Unlock()
	if streak < 3 {
		return
	}
	runtime.mu.Lock()
	runtime.failureStreak = 0
	runtime.mu.Unlock()
	if activeBackup || c.BackupLocation == "" {
		return
	}
	if err := runtime.pool.switchTo(ctx, c.BackupLocation); err == nil {
		c.activeBackup = true
		runtime.mu.Lock()
		runtime.backupSince = time.Now()
		runtime.probeSuccess = 0
		runtime.lastProbe = time.Time{}
		runtime.mu.Unlock()
	}
}

func (c *ModbusRtuOverTcpCollection) maybeFailback(ctx context.Context) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil || !runtime.pool.isBackup() {
		return
	}
	now := time.Now()
	runtime.mu.Lock()
	if runtime.backupSince.IsZero() {
		runtime.backupSince = now
	}
	if now.Sub(runtime.backupSince) < backupHoldDuration {
		runtime.mu.Unlock()
		return
	}
	if !runtime.lastProbe.IsZero() && now.Sub(runtime.lastProbe) < failbackProbeInterval {
		runtime.mu.Unlock()
		return
	}
	runtime.lastProbe = now
	runtime.mu.Unlock()

	if !c.probePrimary(ctx) {
		runtime.mu.Lock()
		runtime.probeSuccess = 0
		runtime.mu.Unlock()
		return
	}
	runtime.mu.Lock()
	runtime.probeSuccess++
	success := runtime.probeSuccess
	runtime.mu.Unlock()
	if success < failbackSuccessCount {
		return
	}
	if err := runtime.pool.switchTo(ctx, c.Location); err == nil {
		c.activeBackup = false
		runtime.mu.Lock()
		runtime.backupSince = time.Time{}
		runtime.probeSuccess = 0
		runtime.lastProbe = time.Time{}
		runtime.mu.Unlock()
		return
	}
	runtime.mu.Lock()
	runtime.probeSuccess = 0
	runtime.mu.Unlock()
}

func (c *ModbusRtuOverTcpCollection) probePrimary(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil || len(runtime.frames) == 0 || c.Location == "" {
		return false
	}
	frame := runtime.frames[0]
	response := make([]byte, frame.responseLen)
	if err := runtime.pool.probe(ctx, c.Location, frame.request, response); err != nil {
		return false
	}
	_, err := modbus.ValidateRTUResponse(response, frame.frame)
	return err == nil
}

// DeliverAction writes collection variables through the active TCP pool.
func (c *ModbusRtuOverTcpCollection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
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
		exchangeErr := runtime.pool.exchange(ctx, request, response)
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

func (c *ModbusRtuOverTcpCollection) traceRequest(operation string, data []byte) uint64 {
	if c.frameTracer == nil || len(data) == 0 {
		return 0
	}
	traceID := c.frameTracer.NextTraceID()
	c.frameTracer.TraceRequest(traceID, operation, data)
	return traceID
}

func (c *ModbusRtuOverTcpCollection) traceResponse(traceID uint64, operation string, data []byte, started time.Time, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, time.Since(started), err)
}

var _ collection.Collector = (*ModbusRtuOverTcpCollection)(nil)
var _ collection.CollectorFactory = (*CollectorFactory)(nil)
