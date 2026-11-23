package dnp3

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
)

type tagRef struct {
	variable *collection.Variable
	address  pointAddress
}

type collectorRuntime struct {
	mu      sync.Mutex
	tags    map[uint32][]*tagRef
	pool    *connpool.Pool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool

	last0123 time.Time
	last123  time.Time
	appSeq   byte
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("DNP3 collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
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
	if c.MasterID < 0 || c.MasterID > 65535 || c.SlaveID < 0 || c.SlaveID > 65535 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: master/slave id is invalid", collection.ErrInvalidConfig)
	}
	if c.Class0123Interval <= 0 {
		c.Class0123Interval = DefaultClass0123Interval
	}
	if c.Class123Interval <= 0 {
		c.Class123Interval = DefaultClass123Interval
	}
	tags := make(map[uint32][]*tagRef)
	for _, variable := range c.GetCollectionMeta().GetVariables() {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := parseAddress(variable.Address)
		if err != nil {
			c.SetCollectStatus(collection.CollectStatusError)
			return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
		}
		key := address.key()
		tags[key] = append(tags[key], &tagRef{variable: variable, address: address})
	}
	runtime := &collectorRuntime{tags: tags}
	c.runtime = runtime
	if len(tags) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		return newDNP3Client(ctx, c.endpoint(), c.MasterID, c.SlaveID, dnp3Timeout)
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
		return fmt.Errorf("DNP3 runtime is not connected")
	}
	runtime.started = true
	runtime.last0123 = time.Time{}
	runtime.last123 = time.Time{}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
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
			if !c.pollDue(ctx, runtime, now) {
				return
			}
		}
	}
}

func (c *Collection) pollDue(ctx context.Context, runtime *collectorRuntime, now time.Time) bool {
	due0123 := runtime.last0123.IsZero() || now.Sub(runtime.last0123) >= time.Duration(c.Class0123Interval)*time.Millisecond
	due123 := runtime.last123.IsZero() || now.Sub(runtime.last123) >= time.Duration(c.Class123Interval)*time.Millisecond
	ran := false
	if due0123 {
		if c.TimeSync == 1 {
			if err := c.pollTimeSync(ctx, runtime, now); err != nil {
				return c.handlePollError(err)
			}
		}
		if err := c.pollClass(ctx, runtime, []byte{1, 2, 3, 4}); err != nil {
			return c.handlePollError(err)
		}
		runtime.last0123 = now
		ran = true
	}
	if due123 {
		if err := c.pollClass(ctx, runtime, []byte{2, 3, 4}); err != nil {
			return c.handlePollError(err)
		}
		runtime.last123 = now
		ran = true
	}
	if ran {
		c.SetCollectStatus(collection.CollectStatusCollecting)
		c.GetCollectionMeta().ReportError(nil)
	}
	return true
}

func (c *Collection) handlePollError(err error) bool {
	if collection.IsConnectionLost(err) {
		c.GetCollectionMeta().ReportError(err)
		return collection.MarkConnectionLost(c, err)
	}
	c.GetCollectionMeta().ReportError(err)
	c.SetCollectStatus(collection.CollectStatusCollectingError)
	return true
}

func (c *Collection) pollTimeSync(ctx context.Context, runtime *collectorRuntime, now time.Time) error {
	client, err := runtime.pool.Get(ctx)
	if err != nil {
		return collection.ConnectionLost(err)
	}
	session, ok := client.(*dnp3Client)
	if !ok {
		runtime.pool.Put(client, true)
		return fmt.Errorf("invalid DNP3 client type")
	}
	app := buildTimeSync(runtime.nextAppSeq(), now)
	_, wireRequest, wireResponse, err := session.Exchange(ctx, app)
	runtime.pool.Put(client, err != nil)
	traceID := c.traceRequest("time-sync", wireRequest)
	c.traceResponse(traceID, "time-sync", wireResponse, err)
	if err != nil {
		return collection.ConnectionLost(err)
	}
	return nil
}

func (c *Collection) pollClass(ctx context.Context, runtime *collectorRuntime, variations []byte) error {
	client, err := runtime.pool.Get(ctx)
	if err != nil {
		return collection.ConnectionLost(err)
	}
	session, ok := client.(*dnp3Client)
	if !ok {
		runtime.pool.Put(client, true)
		return fmt.Errorf("invalid DNP3 client type")
	}
	app := buildClassPoll(runtime.nextAppSeq(), variations)
	response, wireRequest, wireResponse, err := session.Exchange(ctx, app)
	runtime.pool.Put(client, err != nil)
	traceID := c.traceRequest("read", wireRequest)
	c.traceResponse(traceID, "read", wireResponse, err)
	if err != nil {
		return collection.ConnectionLost(err)
	}
	values, err := parseAppResponse(response)
	if err != nil {
		return err
	}
	for _, item := range values {
		key := uint32(item.Group)<<24 | uint32(item.Variation)<<16 | uint32(item.Index)
		for _, ref := range runtime.tags[key] {
			c.GetCollectionMeta().SetVariableValue(ref.variable.Name, collection.ApplyTransform(item.Value, ref.variable))
		}
	}
	return nil
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		address, err := parseAddress(action.Variable.Address)
		if err != nil || !address.IsCROB {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		app, err := buildCROBRequest(address, action.Value)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].value", action.Index),
				MsgID: "validate.action_value_invalid",
				Args:  []interface{}{action.Variable.Name, err.Error()},
			}})
		}
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		session, ok := client.(*dnp3Client)
		if !ok {
			runtime.pool.Put(client, true)
			return collection.DeviceActionError(action, fmt.Errorf("invalid DNP3 client type"))
		}
		response, wireRequest, wireResponse, exchangeErr := session.Exchange(ctx, app)
		traceID := c.traceRequest("write", wireRequest)
		c.traceResponse(traceID, "write", wireResponse, exchangeErr)
		runtime.pool.Put(client, exchangeErr != nil)
		if exchangeErr != nil {
			return collection.DeviceActionError(action, exchangeErr)
		}
		if len(response) < 2 || (response[1] != appFunctionResponse && response[1] != appFunctionUnsolicited) {
			return collection.DeviceActionError(action, fmt.Errorf("unexpected DNP3 control response"))
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, action.Value)
	}
	return nil
}

func (r *collectorRuntime) nextAppSeq() byte {
	seq := r.appSeq
	r.appSeq = (r.appSeq + 1) & 0x0F
	return seq
}

func (c *Collection) endpoint() string {
	return fmt.Sprintf("%s:%d", c.Location, c.Port)
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

var _ collection.Collector = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
