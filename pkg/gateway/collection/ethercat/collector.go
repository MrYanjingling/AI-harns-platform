package ethercat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type adsFrame struct {
	Variable *collection.Variable
	Address  parsedAddress
	Length   int
}

type collectorRuntime struct {
	mu      sync.Mutex
	frames  []*adsFrame
	calcs   []*collection.Calculation
	calcErr error
	client  *adsClient

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool

	endpoint   string
	target     [6]byte
	source     [6]byte
	targetPort uint16
	sourcePort uint16
	timeout    time.Duration
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("ethercat collector is nil")
	}
	if c.runtime != nil {
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
	if c.AMSPort < MinADSPort || c.AMSPort > MaxADSPort {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: amsPort must be between %d and %d", collection.ErrInvalidConfig, MinADSPort, MaxADSPort)
	}
	if c.LocalAMSPort < MinADSPort || c.LocalAMSPort > MaxADSPort {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: localAmsPort must be between %d and %d", collection.ErrInvalidConfig, MinADSPort, MaxADSPort)
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	target, err := resolveTargetNetID(c.Location, c.AMSNetID)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	source, err := resolveLocalNetID(c.LocalAMSNetID)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	variables := c.GetCollectionMeta().GetVariables()
	frames, err := buildReadFrames(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calculations, calcErr := collection.CompileCalculations(variables)
	runtime := &collectorRuntime{
		frames:     frames,
		calcs:      calculations,
		calcErr:    calcErr,
		endpoint:   fmt.Sprintf("%s:%d", c.Location, c.Port),
		target:     target,
		source:     source,
		targetPort: uint16(c.AMSPort),
		sourcePort: uint16(c.LocalAMSPort),
		timeout:    time.Duration(c.TimeoutMs) * time.Millisecond,
	}
	if len(frames) == 0 {
		c.runtime = runtime
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	client, err := newADSClient(ctx, runtime.endpoint, target, runtime.targetPort, source, runtime.sourcePort, runtime.timeout)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.client = client
	c.runtime = runtime
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
	if runtime.client == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("ethercat runtime is not connected")
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
	client := runtime.client
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if client != nil {
		_ = client.Close()
	}
	runtime.mu.Lock()
	if runtime.client == client {
		runtime.client = nil
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
	if runtime == nil || runtime.client == nil {
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

func (c *Collection) readFrame(ctx context.Context, frame *adsFrame) (any, error) {
	runtime := c.runtime
	if runtime == nil || runtime.client == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("ethercat runtime is not connected"))
	}
	client := runtime.client
	data, wireRequest, wireResponse, err := client.read(ctx, frame.Address.IndexGroup, frame.Address.IndexOffset, frame.Length)
	traceID := c.traceRequest("read", wireRequest)
	c.traceResponse(traceID, wireResponse, err)
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

func (c *Collection) traceResponse(traceID uint64, data []byte, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, "read", data, 0, err)
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.client == nil {
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
		if address.ReadOnly {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		length, err := typeSize(action.Variable.DataType, address)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		value, err := encodeValue(action.Value, action.Variable.DataType, length)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		_, wireRequest, wireResp, err := runtime.client.write(ctx, address.IndexGroup, address.IndexOffset, value)
		traceID := c.traceRequest("write", wireRequest)
		if c.frameTracer != nil && traceID != 0 {
			if err != nil {
				wireResp = nil
			}
			c.frameTracer.TraceResponse(traceID, "write", wireResp, 0, err)
		}
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func buildReadFrames(variables []*collection.Variable) ([]*adsFrame, error) {
	frames := make([]*adsFrame, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := parseAddress(variable.Address, variable.DataType)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		if address.WriteOnly {
			return nil, fmt.Errorf("variable %s: ads index group 0x%X is not readable", variable.Name, address.IndexGroup)
		}
		length, err := typeSize(variable.DataType, address)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		frames = append(frames, &adsFrame{Variable: variable, Address: address, Length: length})
	}
	return frames, nil
}

var _ collection.Collector = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
