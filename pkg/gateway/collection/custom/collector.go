package custom

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/pluginmgr"
	"harns-platform/pkg/gateway/validation"
)

const (
	frameRetryCount = 3
	defaultCycle    = time.Second
)

type transportClient struct {
	transport pluginmgr.Transport
}

func (c *transportClient) Available() bool {
	return c != nil && c.transport != nil && c.transport.Available()
}
func (c *transportClient) Close() error {
	if c == nil || c.transport == nil {
		return nil
	}
	return c.transport.Close()
}
func (c *transportClient) Exchange(ctx context.Context, request []byte) ([]byte, error) {
	if c == nil || c.transport == nil {
		return nil, fmt.Errorf("custom transport is not available")
	}
	return c.transport.Exchange(ctx, request)
}

type runtimeState struct {
	mu      sync.Mutex
	frames  []pluginmgr.ReadFrame
	calcs   []*collection.Calculation
	calcErr error
	pool    *connpool.Pool

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

// Collector is the host-owned runtime for one custom plugin driver.
type Collector struct {
	collection.CollectionMeta

	driver      pluginmgr.Driver
	config      json.RawMessage
	runtime     *runtimeState
	frameTracer collection.FrameTracer
}

func NewCollector(meta *collection.CollectionMeta, driver pluginmgr.Driver, config json.RawMessage) *Collector {
	item := &Collector{driver: driver, config: append(json.RawMessage(nil), config...)}
	item.SetCollectionMeta(meta)
	return item
}

func (c *Collector) Connect(ctx context.Context) (err error) {
	if c == nil || c.driver == nil {
		return fmt.Errorf("%w: custom driver is nil", collection.ErrInvalidConfig)
	}
	if c.runtime != nil {
		_ = c.Stop(ctx)
	}
	if err = safeError(func() error { return c.driver.ValidateConfig(c.config) }); err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	var kind pluginmgr.TransportKind
	kind, err = safeValue(func() (pluginmgr.TransportKind, error) {
		return c.driver.TransportKind(c.config)
	})
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	variables := c.GetCollectionMeta().GetVariables()
	frames, err := safeValue(func() ([]pluginmgr.ReadFrame, error) {
		return c.driver.BuildReadFrames(variables, c.config)
	})
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calculations, calcErr := collection.CompileCalculations(variables)
	runtime := &runtimeState{frames: frames, calcs: calculations, calcErr: calcErr}
	c.runtime = runtime
	if len(frames) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	size := commonConcurrency(c.config)
	switch kind {
	case pluginmgr.TransportTCP, pluginmgr.TransportUDP:
	case pluginmgr.TransportSerial, pluginmgr.TransportCustom:
		size = 1
	default:
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: unknown transport kind %q", collection.ErrInvalidConfig, kind)
	}
	if size < 1 {
		size = 1
	}
	pool, err := connpool.NewPool(ctx, size, func(ctx context.Context) (connpool.Client, error) {
		transport, err := safeValue(func() (pluginmgr.Transport, error) {
			return c.driver.NewTransport(ctx, c.config)
		})
		if err != nil {
			return nil, err
		}
		return &transportClient{transport: transport}, nil
	})
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.pool = pool
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collector) Start(ctx context.Context) error {
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
		return fmt.Errorf("custom runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
	runtime.wg.Add(1)
	runtime.mu.Unlock()
	go c.loop(runCtx)
	return nil
}

func (c *Collector) Stop(ctx context.Context) error {
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

func (c *Collector) loop(ctx context.Context) {
	runtime := c.runtime
	if runtime == nil {
		return
	}
	defer runtime.wg.Done()
	cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
	if cycle <= 0 {
		cycle = defaultCycle
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
	values []pluginmgr.DecodedValue
	err    error
}

func (c *Collector) poll(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return false
	}
	results := make(chan frameResult, len(runtime.frames))
	var wait sync.WaitGroup
	for _, frame := range runtime.frames {
		wait.Add(1)
		go func(item pluginmgr.ReadFrame) {
			defer wait.Done()
			values, err := c.readFrame(ctx, item)
			results <- frameResult{values: values, err: err}
		}(frame)
	}
	wait.Wait()
	close(results)

	failed := false
	succeeded := 0
	connectionFailures := 0
	protocolFailure := false
	var cycleErr error
	for result := range results {
		for _, item := range result.values {
			if item.Variable != nil {
				c.GetCollectionMeta().SetVariableValue(item.Variable.Name, item.Value)
			}
		}
		if result.err == nil || len(result.values) > 0 {
			succeeded++
		}
		if result.err != nil {
			failed = true
			if cycleErr == nil {
				cycleErr = result.err
			}
			if collection.IsConnectionLost(result.err) {
				connectionFailures++
			} else {
				protocolFailure = true
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

func (c *Collector) readFrame(ctx context.Context, frame pluginmgr.ReadFrame) ([]pluginmgr.DecodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("custom runtime is not connected"))
	}
	var lastErr error
	for attempt := 0; attempt < frameRetryCount; attempt++ {
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return nil, err
		}
		transport, ok := client.(*transportClient)
		if !ok {
			runtime.pool.Put(client, true)
			return nil, fmt.Errorf("invalid custom transport type")
		}
		traceID := c.traceRequest("read", frame.Request)
		started := time.Now()
		response, err := safeValue(func() ([]byte, error) {
			return transport.Exchange(ctx, frame.Request)
		})
		c.traceResponse(traceID, "read", response, started, err)
		if err != nil {
			runtime.pool.Put(client, true)
			lastErr = err
			continue
		}
		runtime.pool.Put(client, false)
		values, err := safeValue(func() ([]pluginmgr.DecodedValue, error) {
			return c.driver.DecodeRead(frame, response)
		})
		if err != nil {
			return values, err
		}
		return values, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("custom driver frame failed")
	}
	return nil, lastErr
}

func (c *Collector) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		request, err := safeValue(func() ([]byte, error) {
			return c.driver.BuildWrite(action, c.config)
		})
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		transport, ok := client.(*transportClient)
		if !ok {
			runtime.pool.Put(client, true)
			return collection.DeviceActionError(action, fmt.Errorf("invalid custom transport type"))
		}
		traceID := c.traceRequest("write", request)
		started := time.Now()
		response, err := safeValue(func() ([]byte, error) {
			return transport.Exchange(ctx, request)
		})
		c.traceResponse(traceID, "write", response, started, err)
		if err != nil {
			runtime.pool.Put(client, true)
			return collection.DeviceActionError(action, err)
		}
		runtime.pool.Put(client, false)
		if err := safeError(func() error { return c.driver.DecodeWrite(response) }); err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func (c *Collector) SetFrameTracer(tracer collection.FrameTracer) {
	c.frameTracer = tracer
}

func (c *Collector) traceRequest(operation string, data []byte) uint64 {
	if c.frameTracer == nil || len(data) == 0 {
		return 0
	}
	traceID := c.frameTracer.NextTraceID()
	c.frameTracer.TraceRequest(traceID, operation, data)
	return traceID
}

func (c *Collector) traceResponse(traceID uint64, operation string, data []byte, started time.Time, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, time.Since(started), err)
}

type commonConfig struct {
	Concurrency int `json:"concurrency"`
}

func commonConcurrency(config json.RawMessage) int {
	var value commonConfig
	if len(config) > 0 {
		_ = json.Unmarshal(config, &value)
	}
	if value.Concurrency < 1 {
		return 1
	}
	return value.Concurrency
}

func safeError(fn func() error) (err error) {
	defer recoverAsError(&err)
	return fn()
}

func safeValue[T any](fn func() (T, error)) (value T, err error) {
	defer recoverAsError(&err)
	return fn()
}

func recoverAsError(err *error) {
	if recovered := recover(); recovered != nil {
		*err = fmt.Errorf("custom driver panic: %v", recovered)
	}
}

var _ collection.Collector = (*Collector)(nil)
var _ collection.ActionHandler = (*Collector)(nil)
var _ collection.FrameTracerAware = (*Collector)(nil)

type CreateReq struct {
	Name             string                 `json:"name"`
	CollectorCycleMs uint64                 `json:"collectorCycleMs"`
	Variables        []*collection.Variable `json:"variables"`
	ProtocolConfig   json.RawMessage        `json:"protocolConfig"`
}

func (r *CreateReq) Validate() error {
	issues := make([]validation.FieldIssue, 0, 2)
	if strings.TrimSpace(r.Name) == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.collection_name_required"})
	}
	if r.CollectorCycleMs == 0 {
		issues = append(issues, validation.FieldIssue{Field: "collectorCycleMs", MsgID: "validate.collection_cycle_invalid"})
	}
	if err := collection.NormalizeVariables(r.Variables); err != nil {
		return err
	}
	if len(issues) > 0 {
		return validation.New(issues)
	}
	return nil
}

type Factory struct {
	driverType  string
	driverModel string
	driver      pluginmgr.Driver
}

func NewFactory(driverType, driverModel string, driver pluginmgr.Driver) *Factory {
	return &Factory{driverType: driverType, driverModel: driverModel, driver: driver}
}

func (f *Factory) DriverType() string  { return f.driverType }
func (f *Factory) DriverModel() string { return f.driverModel }

func (f *Factory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	return NewCollector(meta, f.driver, config), nil
}

func (f *Factory) DecodeCreate(body []byte) (collection.CreateRequest, error) {
	var req CreateReq
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (f *Factory) BuildCreate(raw collection.CreateRequest) (collection.Collector, json.RawMessage, error) {
	req, ok := raw.(*CreateReq)
	if !ok {
		return nil, nil, fmt.Errorf("invalid custom driver create request")
	}
	if err := safeError(func() error { return f.driver.ValidateConfig(req.ProtocolConfig) }); err != nil {
		return nil, nil, validation.New([]validation.FieldIssue{{
			Field: "protocolConfig", MsgID: "validate.collection_field_invalid", Args: []interface{}{err.Error()},
		}})
	}
	meta := &collection.CollectionMeta{
		Meta:             collection.Meta{Name: req.Name},
		DriverType:       f.driverType,
		DriverModel:      f.driverModel,
		CollectStatus:    collection.CollectStatusStopped,
		CollectorCycleMs: req.CollectorCycleMs,
		Variables:        req.Variables,
	}
	return NewCollector(meta, f.driver, req.ProtocolConfig), req.ProtocolConfig, nil
}

var _ collection.CollectorFactory = (*Factory)(nil)
var _ collection.CreateHandler = (*Factory)(nil)
