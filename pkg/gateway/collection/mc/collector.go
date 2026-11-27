package mc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
)

type runtime struct {
	mu sync.Mutex

	pool     *connpool.Pool
	commands []*command
	calcs    []*collection.Calculation
	calcErr  error

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	serial  uint16
}

type Collection struct {
	collection.CollectionMeta

	Info          Info   `json:"-"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	TransportMode int    `json:"transport_mode"`
	SrcPort       int    `json:"src-port"`
	TimeoutMs     int    `json:"timeout"`
	IntervalMs    int    `json:"interval"`
	Retry         int    `json:"retry"`

	runtime     *runtime
	frameTracer collection.FrameTracer
}

func NewCollection(info Info, req *CreateReq) (*Collection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       info.DriverType,
			DriverModel:      info.DriverModel,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Info:          info,
		Host:          req.ProtocolConfig.Host,
		Port:          req.ProtocolConfig.Port,
		TransportMode: req.ProtocolConfig.TransportMode,
		SrcPort:       req.ProtocolConfig.SrcPort,
		TimeoutMs:     req.ProtocolConfig.TimeoutMs,
		IntervalMs:    req.ProtocolConfig.IntervalMs,
		Retry:         req.ProtocolConfig.Retry,
	}, nil
}

func NewEmptyCollection(info Info) *Collection {
	return &Collection{Info: info}
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Host = req.ProtocolConfig.Host
	c.Port = req.ProtocolConfig.Port
	c.TransportMode = req.ProtocolConfig.TransportMode
	c.SrcPort = req.ProtocolConfig.SrcPort
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
	c.IntervalMs = req.ProtocolConfig.IntervalMs
	c.Retry = req.ProtocolConfig.Retry
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Host: c.Host, Port: c.Port, TransportMode: c.TransportMode, SrcPort: c.SrcPort, TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs, Retry: c.Retry}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.Host = config.Host
	c.Port = config.Port
	c.TransportMode = config.TransportMode
	c.SrcPort = config.SrcPort
	c.TimeoutMs = config.TimeoutMs
	c.IntervalMs = config.IntervalMs
	c.Retry = config.Retry
	return nil
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("MC collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Host) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: host is required", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	calcs, calcErr := collection.CompileCalculations(variables)
	run := &runtime{commands: buildCommandsForFamily(c.Info.Family, variables), calcs: calcs, calcErr: calcErr}
	c.runtime = run
	if len(run.commands) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := newPool(ctx, ProtocolConfig{Host: c.Host, Port: c.Port, TransportMode: c.TransportMode, SrcPort: c.SrcPort, TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs, Retry: c.Retry}, c.Info)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	run.pool = pool
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
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
	if len(run.commands) == 0 {
		run.mu.Unlock()
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	if run.pool == nil {
		run.mu.Unlock()
		return fmt.Errorf("MC runtime is not connected")
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
	for _, cmd := range run.commands {
		data, err := c.readCommand(ctx, cmd)
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
		for _, item := range cmd.points {
			value, err := decodePoint(item, cmd, data)
			if err != nil {
				failed = true
				if cycleErr == nil {
					cycleErr = err
				}
				continue
			}
			c.GetCollectionMeta().SetVariableValue(item.variable.Name, collection.ApplyTransform(value, item.variable))
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

func (c *Collection) readCommand(ctx context.Context, cmd *command) ([]byte, error) {
	attempts := c.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	expected := cmd.count * 2
	if !cmd.word {
		expected = (cmd.count + 1) / 2
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		serial, response, err := c.exchangeRead(ctx, cmd)
		if err != nil {
			return nil, err
		}
		var data []byte
		if c.Info.ASCII {
			data, err = parse3EASCIIResponse(response, cmd.word)
		} else {
			data, err = parseResponse(c.Info.Family, serial, response)
		}
		if err != nil {
			lastErr = err
			continue
		}
		if len(data) != expected {
			lastErr = fmt.Errorf("MC response data length %d, want %d", len(data), expected)
			continue
		}
		return data, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("MC read request failed")
	}
	return nil, lastErr
}

func (c *Collection) exchangeRead(ctx context.Context, cmd *command) (uint16, []byte, error) {
	run := c.runtime
	if run == nil || run.pool == nil {
		return 0, nil, collection.ConnectionLost(fmt.Errorf("MC runtime is not connected"))
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	serial := run.nextSerialLocked(c.Info.Family)
	var request []byte
	if c.Info.ASCII {
		request = build3EASCIIRequest(cmd.area, cmd.word, cmd.start, cmd.count, nil)
	} else {
		request = buildReadRequest(c.Info.Family, serial, cmd.area, cmd.word, cmd.start, cmd.count)
	}
	traceID := c.traceRequest("read", request)
	started := time.Now()
	response, err := c.exchangeLocked(ctx, request)
	c.traceResponse(traceID, "read", response, started, err)
	return serial, response, err
}

func (c *Collection) writeCommand(ctx context.Context, area byte, word bool, start uint32, count int, data []byte) error {
	run := c.runtime
	if run == nil || run.pool == nil {
		return collection.ConnectionLost(fmt.Errorf("MC runtime is not connected"))
	}
	attempts := c.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		run.mu.Lock()
		serial := run.nextSerialLocked(c.Info.Family)
		var request []byte
		if c.Info.ASCII {
			request = build3EASCIIRequest(area, word, start, count, data)
		} else {
			request = buildWriteRequest(c.Info.Family, serial, area, word, start, count, data)
		}
		traceID := c.traceRequest("write", request)
		started := time.Now()
		response, err := c.exchangeLocked(ctx, request)
		c.traceResponse(traceID, "write", response, started, err)
		run.mu.Unlock()
		if err != nil {
			return err
		}
		var parseErr error
		if c.Info.ASCII {
			_, parseErr = parse3EASCIIResponse(response, word)
		} else {
			_, parseErr = parseResponse(c.Info.Family, serial, response)
		}
		if parseErr != nil {
			lastErr = parseErr
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("MC write request failed")
	}
	return lastErr
}

func (c *Collection) exchangeLocked(ctx context.Context, request []byte) ([]byte, error) {
	run := c.runtime
	var response []byte
	err := run.pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
		transport, ok := client.(connpool.Transport)
		if !ok {
			_ = client.Close()
			return false, fmt.Errorf("invalid MC transport")
		}
		resp, exchangeErr := transport.Exchange(ctx, request)
		if exchangeErr != nil {
			return true, exchangeErr
		}
		response = resp
		return false, nil
	})
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	return response, nil
}

func (run *runtime) nextSerialLocked(family Family) uint16 {
	if family != Family4E {
		return 0
	}
	serial := run.serial
	run.serial++
	return serial
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
	run := c.runtime
	if run == nil || run.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		item, err := parseAddress(action.Variable)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
		}
		if item.word && (item.dataType == "BIT" || item.dataType == "BOOL") {
			if err := c.writeWordBit(ctx, item, action.Value); err != nil {
				return collection.DeviceActionError(action, err)
			}
		} else {
			data, err := encodePoint(item, action.Value)
			if err != nil {
				return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
			}
			if err := c.writeCommand(ctx, item.area, item.word, item.number, item.quantity, data); err != nil {
				return collection.DeviceActionError(action, err)
			}
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func (c *Collection) writeWordBit(ctx context.Context, item *point, value any) error {
	bit, err := valueAsBool(value)
	if err != nil {
		return err
	}
	cmd := &command{area: item.area, word: true, start: item.number, count: 1, points: []*point{item}}
	data, err := c.readCommand(ctx, cmd)
	if err != nil {
		return err
	}
	if len(data) < 2 {
		return fmt.Errorf("MC word bit read is too short")
	}
	word := uint16(data[0]) | uint16(data[1])<<8
	if bit {
		word |= 1 << item.bit
	} else {
		word &^= 1 << item.bit
	}
	write := []byte{byte(word), byte(word >> 8)}
	return c.writeCommand(ctx, item.area, true, item.number, 1, write)
}

var (
	_ collection.Collector     = (*Collection)(nil)
	_ collection.ActionHandler = (*Collection)(nil)
)
