package iec104

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type collectorRuntime struct {
	mu      sync.Mutex
	client  *client
	calcs   []*collection.Calculation
	calcErr error
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

type client struct {
	conn    net.Conn
	mu      sync.Mutex
	ca      uint16
	timeout time.Duration
	tx      uint16
	rx      uint16
}

type inFrame struct {
	u    byte
	asdu []byte
}

type CollectorFactory struct{}

func (f *CollectorFactory) DriverType() string  { return DriverType }
func (f *CollectorFactory) DriverModel() string { return DriverModel }

func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, configJson json.RawMessage) (collection.Collector, error) {
	item := &Collection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(configJson); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *Collection) Connect(ctx context.Context) error {
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if c.Timeout <= 0 {
		c.Timeout = 5
	}
	address := fmt.Sprintf("%s:%d", c.Location, c.Port)
	dialer := net.Dialer{Timeout: time.Duration(c.Timeout) * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	client := &client{conn: conn, ca: uint16(c.CA), timeout: time.Duration(c.Timeout) * time.Second}
	if err := client.startDT(); err != nil {
		_ = conn.Close()
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	calculations, calcErr := collection.CompileCalculations(c.GetCollectionMeta().GetVariables())
	c.runtime = &collectorRuntime{client: client, calcs: calculations, calcErr: calcErr}
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) Start(ctx context.Context) error {
	runtime := c.runtime
	if runtime == nil || runtime.client == nil {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	runtime.mu.Lock()
	if runtime.started {
		runtime.mu.Unlock()
		return nil
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
	runtime.client = nil
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if client != nil {
		_ = client.close()
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
	if runtime == nil || runtime.client == nil {
		return false
	}
	if !c.Active {
		c.SetCollectStatus(collection.CollectStatusCollecting)
		return true
	}
	values, err := c.interrogate(ctx, runtime.client)
	if err != nil {
		c.GetCollectionMeta().ReportError(err)
		if collection.IsConnectionLost(err) {
			return collection.MarkConnectionLost(c, err)
		}
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	vars := c.GetCollectionMeta().GetVariables()
	for _, variable := range vars {
		if variable == nil || variable.IsCalculated {
			continue
		}
		ioa, err := parseIOA(variable.Address)
		if err != nil {
			continue
		}
		if value, ok := values[ioa]; ok {
			if converted, ok := coerceValue(value, variable.DataType); ok {
				c.GetCollectionMeta().SetVariableValue(variable.Name, collection.ApplyTransform(converted, variable))
			}
		}
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

func (c *Collection) interrogate(ctx context.Context, client *client) (map[uint32]any, error) {
	return client.interrogate()
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.client == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		ioa, err := parseIOA(action.Variable.Address)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
		}
		var asdu []byte
		switch {
		case isBoolType(action.Variable.DataType):
			value, ok := action.Value.(bool)
			if !ok {
				return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
			}
			asdu = buildSingleCommandASDU(uint16(c.CA), ioa, value)
		default:
			value, ok := numericFloat(action.Value)
			if !ok {
				return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
			}
			asdu = buildSetpointASDU(uint16(c.CA), ioa, value)
		}
		if err := runtime.client.executeCommand(asdu, ioa); err != nil {
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

func (c *Collection) traceResponse(traceID uint64, operation string, data []byte, started time.Time, err error) {
	if c.frameTracer == nil || traceID == 0 {
		return
	}
	if err != nil {
		data = nil
	}
	c.frameTracer.TraceResponse(traceID, operation, data, time.Since(started), err)
}

func newClient(conn net.Conn, ca uint16, timeout time.Duration) *client {
	return &client{conn: conn, ca: ca, timeout: timeout}
}

func (c *client) close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *client) startDT() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	if _, err := c.conn.Write(buildUFrame(0x07)); err != nil {
		return err
	}
	frame, err := c.readFrameLocked(time.Now().Add(c.timeout))
	if err != nil {
		return err
	}
	if frame.u != 0x0b {
		return fmt.Errorf("iec104 STARTDT confirmation is missing")
	}
	return nil
}

func (c *client) interrogate() (map[uint32]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.sendILocked(buildInterrogationASDU(c.ca)); err != nil {
		return nil, err
	}
	values := make(map[uint32]any)
	deadline := time.Now().Add(c.timeout)
	for time.Now().Before(deadline) {
		frame, err := c.readFrameLocked(deadline)
		if err != nil {
			return values, err
		}
		if frame.u != 0 || len(frame.asdu) == 0 {
			continue
		}
		items, termination, err := parseASDU(frame.asdu, c.ca)
		if err != nil {
			return values, err
		}
		for ioa, value := range items {
			values[ioa] = value
		}
		if termination {
			return values, nil
		}
	}
	return values, collection.ConnectionLost(fmt.Errorf("iec104 interrogation timeout"))
}

func (c *client) executeCommand(asdu []byte, ioa uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.sendILocked(asdu); err != nil {
		return err
	}
	deadline := time.Now().Add(c.timeout)
	for time.Now().Before(deadline) {
		frame, err := c.readFrameLocked(deadline)
		if err != nil {
			return err
		}
		if frame.u != 0 || len(frame.asdu) < 9 || frame.asdu[0] != asdu[0] {
			continue
		}
		cot := frame.asdu[2] & 0x3f
		if (cot == 7 || cot == 10) && readUint24LE(frame.asdu[6:]) == ioa {
			return nil
		}
	}
	return fmt.Errorf("iec104 command confirmation timeout")
}

func (c *client) sendILocked(asdu []byte) error {
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	frame := buildIFrame(asdu, c.tx, c.rx)
	c.tx = (c.tx + 1) & 0x7fff
	_, err := c.conn.Write(frame)
	return err
}

func (c *client) sendS() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sendSLocked()
}

func (c *client) sendSLocked() error {
	if _, err := c.conn.Write(buildSFrame(c.rx)); err != nil {
		return err
	}
	return nil
}

func (c *client) readFrame(deadline time.Time) (inFrame, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readFrameLocked(deadline)
}

func (c *client) readFrameLocked(deadline time.Time) (inFrame, error) {
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return inFrame{}, err
	}
	head := make([]byte, 2)
	if _, err := io.ReadFull(c.conn, head); err != nil {
		return inFrame{}, err
	}
	if head[0] != startByte || head[1] < 4 {
		return inFrame{}, fmt.Errorf("invalid iec104 frame header")
	}
	body := make([]byte, int(head[1]))
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return inFrame{}, err
	}
	if len(body) < 4 {
		return inFrame{}, fmt.Errorf("invalid iec104 frame length")
	}
	ctrl := body[:4]
	switch ctrl[0] & 0x03 {
	case 0x01:
		c.rx = (uint16(ctrl[2])>>1 | uint16(ctrl[3])<<7) + 1
		return inFrame{}, nil
	case 0x03:
		return inFrame{u: ctrl[0]}, nil
	default:
		c.rx = (uint16(ctrl[0])>>1 | uint16(ctrl[1])<<7) + 1
		if err := c.sendSLocked(); err != nil {
			return inFrame{}, err
		}
		return inFrame{asdu: body[4:]}, nil
	}
}

func coerceValue(value any, dataType string) (any, bool) {
	switch dataTypeLower(dataType) {
	case "bool", "bit":
		switch v := value.(type) {
		case bool:
			return v, true
		case float32:
			return v != 0, true
		case float64:
			return v != 0, true
		case int16:
			return v != 0, true
		case int32:
			return v != 0, true
		}
	case "int16":
		switch v := value.(type) {
		case float32:
			return int16(v), true
		case float64:
			return int16(v), true
		case int16:
			return v, true
		case int32:
			return int16(v), true
		}
	case "int32":
		switch v := value.(type) {
		case float32:
			return int32(v), true
		case float64:
			return int32(v), true
		case int16:
			return int32(v), true
		case int32:
			return v, true
		}
	case "float32":
		switch v := value.(type) {
		case float32:
			return v, true
		case float64:
			return float32(v), true
		}
	case "float64":
		switch v := value.(type) {
		case float32:
			return float64(v), true
		case float64:
			return v, true
		}
	}
	return nil, false
}

func numericFloat(value any) (float32, bool) {
	switch v := value.(type) {
	case float64:
		return float32(v), true
	case float32:
		return v, true
	case int16:
		return float32(v), true
	case int32:
		return float32(v), true
	case int64:
		return float32(v), true
	default:
		return 0, false
	}
}

func isBoolType(dataType string) bool {
	return dataTypeLower(dataType) == "bool" || dataTypeLower(dataType) == "bit"
}

func dataTypeLower(dataType string) string {
	return strings.ToLower(strings.TrimSpace(dataType))
}
