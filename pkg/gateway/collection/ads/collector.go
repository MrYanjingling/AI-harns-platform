package ads

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
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
	pool    *connpool.Pool

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	last    time.Time

	endpoint   string
	target     [6]byte
	source     [6]byte
	targetPort uint16
	sourcePort uint16
	timeout    time.Duration
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("Beckhoff ADS collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Host) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: host is required", collection.ErrInvalidConfig)
	}
	if c.Port < 1 || c.Port > 65535 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: port is invalid", collection.ErrInvalidConfig)
	}
	if c.SrcADSPort < 1 || c.SrcADSPort > 65535 || c.DstADSPort < 1 || c.DstADSPort > 65535 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: ADS port is invalid", collection.ErrInvalidConfig)
	}
	target, err := resolveTargetNetID(c.Host, c.DstAMSNetID)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	source, err := resolveLocalNetID(c.SrcAMSNetID)
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
		endpoint:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		target:     target,
		source:     source,
		targetPort: uint16(c.DstADSPort),
		sourcePort: uint16(c.SrcADSPort),
		timeout:    time.Duration(c.TimeoutMs) * time.Millisecond,
	}
	hasWrite := false
	for _, variable := range variables {
		if variable != nil && !variable.IsCalculated && variable.AccessMode == collection.AccessModeWrite {
			hasWrite = true
			break
		}
	}
	c.runtime = runtime
	if len(frames) == 0 && !hasWrite {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		return newADSClient(ctx, runtime.endpoint, target, runtime.targetPort, source, runtime.sourcePort, runtime.timeout)
	})
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
	if runtime.pool == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("Beckhoff ADS runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
	runtime.last = time.Time{}
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

func (c *Collection) loop(ctx context.Context) {
	runtime := c.runtime
	if runtime == nil {
		return
	}
	defer runtime.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
			if cycle <= 0 {
				cycle = time.Second
			}
			if !runtime.startedAt(now, cycle) {
				continue
			}
			if err := c.poll(ctx, runtime); err != nil {
				if collection.IsConnectionLost(err) {
					c.GetCollectionMeta().ReportError(err)
					collection.MarkConnectionLost(c, err)
					return
				}
				c.GetCollectionMeta().ReportError(err)
				c.SetCollectStatus(collection.CollectStatusCollectingError)
				continue
			}
			c.SetCollectStatus(collection.CollectStatusCollecting)
			if runtime.calcErr != nil {
				c.GetCollectionMeta().ReportError(runtime.calcErr)
				c.SetCollectStatus(collection.CollectStatusCollectingError)
				continue
			}
			if err := collection.Calculate(c.GetCollectionMeta(), runtime.calcs); err != nil {
				c.GetCollectionMeta().ReportError(err)
				c.SetCollectStatus(collection.CollectStatusCollectingError)
				continue
			}
			c.GetCollectionMeta().ReportError(nil)
		}
	}
}

func (r *collectorRuntime) startedAt(now time.Time, cycle time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.last.IsZero() && now.Sub(r.last) < cycle {
		return false
	}
	r.last = now
	return true
}

func (c *Collection) poll(ctx context.Context, runtime *collectorRuntime) error {
	if runtime.pool == nil {
		return collection.ConnectionLost(fmt.Errorf("Beckhoff ADS runtime is not connected"))
	}
	succeeded := 0
	connectionFailures := 0
	protocolFailure := false
	var cycleErr error
	for index, frame := range runtime.frames {
		if index > 0 && c.IntervalMs > 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(c.IntervalMs) * time.Millisecond):
			}
		}
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.ConnectionLost(err)
		}
		value, err := c.readFrame(ctx, client, frame)
		runtime.pool.Put(client, err != nil && isTransportError(err))
		if err != nil {
			if cycleErr == nil {
				cycleErr = err
			}
			if isTransportError(err) {
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
		return nil
	}
	if cycleErr != nil {
		c.GetCollectionMeta().ReportError(cycleErr)
		if !collection.ApplyCycleFailure(c, cycleErr, succeeded, connectionFailures, protocolFailure) {
			return collection.ConnectionLost(cycleErr)
		}
		return cycleErr
	}
	return nil
}

func (c *Collection) readFrame(ctx context.Context, client connpool.Client, frame *adsFrame) (any, error) {
	conn, ok := client.(*adsClient)
	if !ok {
		return nil, fmt.Errorf("invalid Beckhoff ADS client type")
	}
	data, wireRequest, wireResponse, err := conn.read(ctx, frame.Address.IndexGroup, frame.Address.IndexOffset, frame.Length)
	traceID := c.traceRequest("read", wireRequest)
	c.traceResponse(traceID, "read", wireResponse, err)
	if err != nil {
		return nil, fmt.Errorf("variable %s: %w", frame.Variable.Name, err)
	}
	value, err := decodeValue(data, frame.Variable.DataType)
	if err != nil {
		return nil, fmt.Errorf("variable %s: %w", frame.Variable.Name, err)
	}
	return collection.ApplyTransform(value, frame.Variable), nil
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		if action.Variable == nil {
			continue
		}
		address, err := parseAddress(action.Variable.Address, action.Variable.DataType)
		if err != nil || address.ReadOnly {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
		}
		length, err := typeSize(action.Variable.DataType, address)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
		}
		value, err := encodeValue(action.Value, action.Variable.DataType, length)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
		}
		err = runtime.pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
			conn, ok := client.(*adsClient)
			if !ok {
				_ = client.Close()
				return false, collection.ConnectionLost(fmt.Errorf("invalid Beckhoff ADS client type"))
			}
			_, wireRequest, wireResponse, writeErr := conn.write(ctx, address.IndexGroup, address.IndexOffset, value)
			traceID := c.traceRequest("write", wireRequest)
			if traceID != 0 {
				if writeErr != nil {
					wireResponse = nil
				}
				c.frameTracer.TraceResponse(traceID, "write", wireResponse, 0, writeErr)
			}
			if writeErr != nil {
				if isTransportError(writeErr) {
					return true, collection.ConnectionLost(writeErr)
				}
				return false, writeErr
			}
			return false, nil
		})
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
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

func (c *Collection) traceResponse(traceID uint64, operation string, data []byte, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, 0, err)
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

func isTransportError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "timeout") || strings.Contains(message, "connection") || strings.Contains(message, "closed") || strings.Contains(message, "refused") || strings.Contains(message, "unreachable") || strings.Contains(message, "eof")
}

var _ collection.Collector = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
