package modbustcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbus"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/utils/binutil"
)

var (
	backupHoldDuration    = 5 * time.Minute
	failbackProbeInterval = 5 * time.Minute
)

const (
	failbackSuccessCount = 3
	frameRetryCount      = 3
)

// CollectorFactory creates Modbus TCP collectors.
type CollectorFactory struct{}

// DriverType returns the Modbus driver type.
func (f *CollectorFactory) DriverType() string { return DriverType }

// DriverModel returns the Modbus TCP driver model.
func (f *CollectorFactory) DriverModel() string { return DriverModel }

// NewCollector creates a Modbus TCP collector.
func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &ModbusTcpCollection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

// Connect builds frames and initializes the TCP connection pool.
func (c *ModbusTcpCollection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("modbusTcp collector is nil")
	}
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
	frames, err := buildWireFrames(variables, c.PositionAddress)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calculations, err := compileCalculations(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	runtime := &collectorRuntime{
		frames: frames,
		calcs:  calculations,
	}
	c.runtime = runtime
	if len(frames) == 0 {
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

// Start starts the polling loop.
func (c *ModbusTcpCollection) Start(ctx context.Context) error {
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
		return fmt.Errorf("modbusTcp runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
	runtime.wg.Add(1)
	runtime.mu.Unlock()

	go c.loop(runCtx)
	return nil
}

// Stop stops the polling loop and closes the connection pool.
func (c *ModbusTcpCollection) Stop(ctx context.Context) error {
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

func (c *ModbusTcpCollection) loop(ctx context.Context) {
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
		elapsed := time.Since(started)
		delay := cycle - elapsed
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

func (c *ModbusTcpCollection) poll(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return false
	}
	frames := runtime.frames
	results := make(chan frameResult, len(frames))
	var wait sync.WaitGroup
	for _, frame := range frames {
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
	if err := c.calculate(); err != nil {
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.maybeFailback(ctx)
	return true
}

// SetFrameTracer injects live frame tracing.
func (c *ModbusTcpCollection) SetFrameTracer(tracer collection.FrameTracer) {
	c.frameTracer = tracer
}

func (c *ModbusTcpCollection) collectFrame(ctx context.Context, frame *wireFrame) ([]modbus.DecodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return nil, fmt.Errorf("modbusTcp runtime is not connected")
	}
	var lastErr error
	for attempt := 0; attempt < frameRetryCount; attempt++ {
		transaction := c.nextTransactionID()
		request := append([]byte(nil), frame.request...)
		binutil.WriteUint16BigEndian(request, transaction)
		traceID := c.traceRequest("read", request)
		response := make([]byte, frame.responseLen)
		started := time.Now()
		exchangeErr := runtime.pool.exchange(ctx, request, response)
		c.traceResponse(traceID, "read", response, started, exchangeErr)
		if exchangeErr != nil {
			lastErr = exchangeErr
			continue
		}
		payload, err := validateTCPReadResponse(response, frame.frame, transaction)
		if err != nil {
			lastErr = err
			continue
		}
		return modbus.DecodeFrame(frame.frame, payload, c.MemoryLayout)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("modbusTcp frame failed")
	}
	return nil, lastErr
}

func (c *ModbusTcpCollection) nextTransactionID() uint16 {
	if c.runtime == nil {
		return 1
	}
	return uint16(atomic.AddUint32(&c.runtime.tx, 1))
}

func (c *ModbusTcpCollection) recordCycleFailure(ctx context.Context) {
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

func (c *ModbusTcpCollection) maybeFailback(ctx context.Context) {
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

func (c *ModbusTcpCollection) probePrimary(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil || len(runtime.frames) == 0 || c.Location == "" {
		return false
	}
	frame := runtime.frames[0]
	transaction := c.nextTransactionID()
	request := append([]byte(nil), frame.request...)
	binutil.WriteUint16BigEndian(request, transaction)
	response := make([]byte, frame.responseLen)
	if err := runtime.pool.probe(ctx, c.Location, request, response); err != nil {
		return false
	}
	_, err := validateTCPReadResponse(response, frame.frame, transaction)
	return err == nil
}

// DeliverAction writes collection variables through the active TCP pool.
func (c *ModbusTcpCollection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
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
		transaction := c.nextTransactionID()
		request := modbus.BuildTCPWriteRequest(write.Slave, transaction, write.PDU())
		traceID := c.traceRequest("write", request)
		response := make([]byte, 12)
		started := time.Now()
		exchangeErr := runtime.pool.exchange(ctx, request, response)
		c.traceResponse(traceID, "write", response, started, exchangeErr)
		if exchangeErr != nil {
			return exchangeErr
		}
		if err := modbus.ValidateTCPWriteResponse(response, transaction, write.Slave, write.Function); err != nil {
			return err
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func (c *ModbusTcpCollection) traceRequest(operation string, data []byte) uint64 {
	if c.frameTracer == nil || len(data) == 0 {
		return 0
	}
	traceID := c.frameTracer.NextTraceID()
	c.frameTracer.TraceRequest(traceID, operation, data)
	return traceID
}

func (c *ModbusTcpCollection) traceResponse(traceID uint64, operation string, data []byte, started time.Time, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, time.Since(started), err)
}

var _ collection.Collector = (*ModbusTcpCollection)(nil)
var _ collection.CollectorFactory = (*CollectorFactory)(nil)
