package iec101

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type transport interface {
	io.ReadWriteCloser
	SetReadTimeout(time.Duration) error
}

type tcpTransport struct{ net.Conn }
type serialTransport struct{ serial.Port }

func (t *tcpTransport) SetReadTimeout(d time.Duration) error {
	return t.SetReadDeadline(time.Now().Add(d))
}
func (t *serialTransport) SetReadTimeout(d time.Duration) error { return t.Port.SetReadTimeout(d) }

type collectorRuntime struct {
	mu      sync.Mutex
	port    transport
	calcs   []*collection.Calculation
	calcErr error
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
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
	var port transport
	if c.Link == 1 {
		conn, dialErr := net.DialTimeout("tcp", c.Location, time.Duration(c.Timeout)*time.Millisecond)
		if dialErr != nil {
			c.SetCollectStatus(collection.CollectStatusUnconnected)
			return dialErr
		}
		port = &tcpTransport{conn}
	} else {
		location := c.Location
		if location == "" {
			location = c.Device
		}
		sp, openErr := serial.Open(location, serialMode(c))
		if openErr != nil {
			c.SetCollectStatus(collection.CollectStatusUnconnected)
			return openErr
		}
		port = &serialTransport{sp}
	}
	calculations, calcErr := collection.CompileCalculations(c.GetCollectionMeta().GetVariables())
	runtime := &collectorRuntime{port: port, calcs: calculations, calcErr: calcErr}
	c.runtime = runtime
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) Start(ctx context.Context) error {
	runtime := c.runtime
	if runtime == nil || runtime.port == nil {
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
	port := runtime.port
	runtime.port = nil
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if port != nil {
		_ = port.Close()
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
	if runtime == nil || runtime.port == nil {
		return false
	}
	values, err := c.interrogate(runtime.port)
	if err != nil {
		c.GetCollectionMeta().ReportError(err)
		if collection.IsConnectionLost(err) {
			return collection.MarkConnectionLost(c, err)
		}
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	for _, variable := range c.GetCollectionMeta().GetVariables() {
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

func (c *Collection) interrogate(port transport) (map[uint32]any, error) {
	linkAddress := uint8(c.CA)
	if err := c.writeReset(port, linkAddress); err != nil {
		return nil, err
	}
	asdu := buildInterrogationASDU(c.CA, c.CALength, c.IOALength)
	if err := c.writeVariable(port, 0x53, linkAddress, asdu); err != nil {
		return nil, err
	}
	values := make(map[uint32]any)
	deadline := time.Now().Add(time.Duration(c.Timeout) * time.Millisecond)
	for time.Now().Before(deadline) {
		if err := port.SetReadTimeout(200); err != nil {
			return values, err
		}
		_, response, err := readLinkFrame(port)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return values, err
		}
		if len(response) == 0 {
			continue
		}
		items, termination, err := parseASDU(response, c.CALength, c.IOALength)
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
	return values, collection.ConnectionLost(fmt.Errorf("iec101 interrogation timeout"))
}

func (c *Collection) writeReset(port transport, link uint8) error {
	if err := port.SetReadTimeout(200); err != nil {
		return err
	}
	if _, err := port.Write(buildFixedFrame(0x40, link)); err != nil {
		return collection.ConnectionLost(err)
	}
	_, _, _ = readLinkFrame(port)
	return nil
}

func (c *Collection) writeVariable(port transport, ctrl, link uint8, asdu []byte) error {
	if err := port.SetReadTimeout(time.Duration(c.Timeout) * time.Millisecond); err != nil {
		return err
	}
	_, err := port.Write(buildVariableFrame(ctrl, link, asdu))
	return collection.ConnectionLost(err)
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.port == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		ioa, err := parseIOA(action.Variable.Address)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
		}
		var asdu []byte
		if isBoolType(action.Variable.DataType) {
			value, ok := action.Value.(bool)
			if !ok {
				return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
			}
			asdu = buildSingleCommandASDU(c.CA, c.CALength, c.IOALength, ioa, value)
		} else {
			value, ok := numericFloat(action.Value)
			if !ok {
				return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
			}
			asdu = buildSetpointASDU(c.CA, c.CALength, c.IOALength, ioa, value)
		}
		if err := c.writeVariable(runtime.port, 0x53, uint8(c.CA), asdu); err != nil {
			return collection.DeviceActionError(action, err)
		}
		if err := c.waitCommandAck(runtime.port, asdu[0], time.Now().Add(time.Duration(c.Timeout)*time.Millisecond)); err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func (c *Collection) waitCommandAck(port transport, wantType byte, deadline time.Time) error {
	for time.Now().Before(deadline) {
		if err := port.SetReadTimeout(200 * time.Millisecond); err != nil {
			return err
		}
		_, asdu, err := readLinkFrame(port)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return err
		}
		if len(asdu) < 3 || asdu[0] != wantType {
			continue
		}
		cot := asdu[2] & 0x3f
		if cot == 7 || cot == 10 {
			return nil
		}
	}
	return fmt.Errorf("iec101 command confirmation timeout")
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

func serialMode(c *Collection) *serial.Mode {
	baud := c.Baud
	if baud < 1200 {
		baud = 9600
	}
	data := c.Data
	if data < 5 || data > 8 {
		data = 8
	}
	parity := serial.NoParity
	switch c.Parity {
	case 1:
		parity = serial.OddParity
	case 2:
		parity = serial.EvenParity
	}
	stop := serial.OneStopBit
	if c.StopBits == 1 {
		stop = serial.TwoStopBits
	}
	return &serial.Mode{BaudRate: baud, DataBits: data, Parity: parity, StopBits: stop}
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") || strings.Contains(text, "deadline")
}

func coerceValue(value any, dataType string) (any, bool) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "bit":
		switch v := value.(type) {
		case bool:
			return v, true
		case float32:
			return v != 0, true
		}
	case "int16":
		if v, ok := value.(float32); ok {
			return int16(v), true
		}
	case "int32":
		if v, ok := value.(float32); ok {
			return int32(v), true
		}
	case "float32":
		if v, ok := value.(float32); ok {
			return v, true
		}
	case "float64":
		if v, ok := value.(float32); ok {
			return float64(v), true
		}
	}
	return nil, false
}

func numericFloat(value any) (float32, bool) {
	switch v := value.(type) {
	case float32:
		return v, true
	case float64:
		return float32(v), true
	case int16:
		return float32(v), true
	case int32:
		return float32(v), true
	case int64:
		return float32(v), true
	}
	return 0, false
}

func isBoolType(dataType string) bool {
	t := strings.ToLower(strings.TrimSpace(dataType))
	return t == "bool" || t == "bit"
}

var _ collection.CollectorFactory = (*CollectorFactory)(nil)
var _ collection.Collector = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
