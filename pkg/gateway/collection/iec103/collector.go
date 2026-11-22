package iec103

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	mu     sync.Mutex
	port   transport
	cancel context.CancelFunc
	wg     sync.WaitGroup
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
	var port transport
	if c.Link == 1 {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.Location, c.Port), time.Duration(c.Timeout)*time.Millisecond)
		if err != nil {
			c.SetCollectStatus(collection.CollectStatusUnconnected)
			return err
		}
		port = &tcpTransport{conn}
	} else {
		sp, err := serial.Open(c.Location, serialMode(c))
		if err != nil {
			c.SetCollectStatus(collection.CollectStatusUnconnected)
			return err
		}
		port = &serialTransport{sp}
	}
	c.runtime = &collectorRuntime{port: port}
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) Start(ctx context.Context) error {
	r := c.runtime
	if r == nil || r.port == nil {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	r.mu.Lock()
	if r.cancel != nil {
		r.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.wg.Add(1)
	r.mu.Unlock()
	go c.loop(runCtx)
	return nil
}

func (c *Collection) Stop(ctx context.Context) error {
	r := c.runtime
	if r == nil {
		c.SetCollectStatus(collection.CollectStatusStopped)
		return nil
	}
	r.mu.Lock()
	cancel := r.cancel
	r.cancel = nil
	port := r.port
	r.port = nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	r.wg.Wait()
	if port != nil {
		_ = port.Close()
	}
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) loop(ctx context.Context) {
	r := c.runtime
	defer r.wg.Done()
	cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
	if cycle <= 0 {
		cycle = time.Second
	}
	for {
		if !c.poll(ctx, r.port) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(cycle):
		}
	}
}

func (c *Collection) poll(ctx context.Context, port transport) bool {
	if port == nil {
		return false
	}
	if err := port.SetReadTimeout(200 * time.Millisecond); err != nil {
		return c.failConnection(err)
	}
	if _, err := port.Write(buildFixedFrame(0x5B, uint8(c.Port))); err != nil {
		return c.failConnection(err)
	}
	for {
		asdu, err := readLinkFrame(port)
		if err != nil {
			if collection.IsConnectionLost(err) && !collection.IsTimeout(err) {
				return c.failConnection(err)
			}
			break
		}
		if len(asdu) == 0 {
			continue
		}
		values, err := parseASDU(asdu)
		if err != nil {
			continue
		}
		for _, variable := range c.GetCollectionMeta().GetVariables() {
			if variable == nil || variable.IsCalculated {
				continue
			}
			if value, ok := values[variable.Address]; ok {
				c.GetCollectionMeta().SetVariableValue(variable.Name, collection.ApplyTransform(value, variable))
			}
		}
	}
	c.SetCollectStatus(collection.CollectStatusCollecting)
	return true
}

func (c *Collection) failConnection(err error) bool {
	err = collection.ConnectionLost(err)
	c.GetCollectionMeta().ReportError(err)
	return collection.MarkConnectionLost(c, err)
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	return validation.New([]validation.FieldIssue{{Field: "actions", MsgID: "validate.action_not_supported"}})
}
func (c *Collection) SetFrameTracer(tracer collection.FrameTracer) {}
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
	if c.Parity == 1 {
		parity = serial.OddParity
	} else if c.Parity == 2 {
		parity = serial.EvenParity
	}
	stop := serial.OneStopBit
	if c.StopBits == 1 {
		stop = serial.TwoStopBits
	}
	return &serial.Mode{BaudRate: baud, DataBits: data, Parity: parity, StopBits: stop}
}

var _ collection.CollectorFactory = (*CollectorFactory)(nil)
var _ collection.Collector = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
