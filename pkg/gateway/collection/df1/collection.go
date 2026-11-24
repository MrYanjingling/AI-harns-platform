package df1

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "Allen-Bradley"
	DriverModel = "DF1"

	DefaultBaudRate   = 9600
	DefaultDataBits   = 8
	DefaultParity     = "even"
	DefaultStopBits   = "1"
	DefaultTimeoutMs  = 3000
	DefaultIntervalMs = 30
)

type ProtocolConfig struct {
	Location   string `json:"location"`
	BaudRate   int    `json:"baudRate"`
	DataBits   int    `json:"dataBits"`
	Parity     string `json:"parity"`
	StopBits   string `json:"stopBits"`
	TimeoutMs  int    `json:"timeoutMs"`
	IntervalMs int    `json:"intervalMs"`
}

func (c *ProtocolConfig) Normalize() {
	if c.BaudRate == 0 {
		c.BaudRate = DefaultBaudRate
	}
	if c.DataBits == 0 {
		c.DataBits = DefaultDataBits
	}
	if c.Parity == "" {
		c.Parity = DefaultParity
	}
	if c.StopBits == "" {
		c.StopBits = DefaultStopBits
	}
	if c.TimeoutMs == 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.IntervalMs == 0 {
		c.IntervalMs = DefaultIntervalMs
	}
}

type CreateReq struct {
	Name             string                 `json:"name"`
	CollectorCycleMs uint64                 `json:"collectorCycleMs"`
	Variables        []*collection.Variable `json:"variables"`
	ProtocolConfig   ProtocolConfig         `json:"protocolConfig"`
}

type UpdateReq struct {
	Name             string         `json:"name"`
	CollectorCycleMs uint64         `json:"collectorCycleMs"`
	ProtocolConfig   ProtocolConfig `json:"protocolConfig"`
}

type Collection struct {
	collection.CollectionMeta

	Location   string `json:"location"`
	BaudRate   int    `json:"baudRate"`
	DataBits   int    `json:"dataBits"`
	Parity     string `json:"parity"`
	StopBits   string `json:"stopBits"`
	TimeoutMs  int    `json:"timeoutMs"`
	IntervalMs int    `json:"intervalMs"`

	runtime     *collectorRuntime
	frameTracer collection.FrameTracer
}

func NewCollection(req *CreateReq) (*Collection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta: collection.Meta{Name: req.Name}, DriverType: DriverType, DriverModel: DriverModel,
			CollectStatus: collection.CollectStatusStopped, CollectorCycleMs: req.CollectorCycleMs, Variables: req.Variables,
		},
		Location: req.ProtocolConfig.Location, BaudRate: req.ProtocolConfig.BaudRate,
		DataBits: req.ProtocolConfig.DataBits, Parity: req.ProtocolConfig.Parity,
		StopBits: req.ProtocolConfig.StopBits, TimeoutMs: req.ProtocolConfig.TimeoutMs, IntervalMs: req.ProtocolConfig.IntervalMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location, c.BaudRate, c.DataBits = req.ProtocolConfig.Location, req.ProtocolConfig.BaudRate, req.ProtocolConfig.DataBits
	c.Parity, c.StopBits = req.ProtocolConfig.Parity, req.ProtocolConfig.StopBits
	c.TimeoutMs, c.IntervalMs = req.ProtocolConfig.TimeoutMs, req.ProtocolConfig.IntervalMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Location: c.Location, BaudRate: c.BaudRate, DataBits: c.DataBits, Parity: c.Parity, StopBits: c.StopBits, TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.Location, c.BaudRate, c.DataBits = config.Location, config.BaudRate, config.DataBits
	c.Parity, c.StopBits = config.Parity, config.StopBits
	c.TimeoutMs, c.IntervalMs = config.TimeoutMs, config.IntervalMs
	return nil
}

var _ collection.Collector = (*Collection)(nil)
