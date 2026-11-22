package iec102

import (
	"context"
	"encoding/json"
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
		conn, dialErr := net.DialTimeout("tcp", c.Location, time.Duration(c.Timeout)*time.Millisecond)
		if dialErr != nil {
			c.SetCollectStatus(collection.CollectStatusUnconnected)
			return dialErr
		}
		port = &tcpTransport{conn}
	} else {
		sp, openErr := serial.Open(c.Location, serialMode(c))
		if openErr != nil {
			c.SetCollectStatus(collection.CollectStatusUnconnected)
			return openErr
		}
		port = &serialTransport{sp}
	}
	c.runtime = &collectorRuntime{port: port}
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
	if runtime.cancel != nil {
		runtime.mu.Unlock()
		return nil
	}
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
	defer runtime.wg.Done()
	cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
	if cycle <= 0 {
		cycle = time.Second
	}
	for {
		if !c.poll(ctx, runtime.port) {
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
	if _, err := port.Write(buildFixedFrame(0x5B, uint16(c.LA))); err != nil {
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
var _ = strings.TrimSpace
