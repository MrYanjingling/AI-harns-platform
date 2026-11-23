package mpi

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "siemens"
	DriverModel = "mpi"

	DefaultBaudRate   = 38400
	DefaultDataBits   = 8
	DefaultParity     = "odd"
	DefaultStopBits   = "1"
	DefaultStation    = 2
	DefaultTimeoutMs  = 3000
	DefaultIntervalMs = 20
)

type ProtocolConfig struct {
	Location   string `json:"location"`
	BaudRate   int    `json:"baudRate"`
	DataBits   int    `json:"dataBits"`
	Parity     string `json:"parity"`
	StopBits   string `json:"stopBits"`
	Station    int    `json:"station"`
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
	if c.Station == 0 {
		c.Station = DefaultStation
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
	Station    int    `json:"station"`
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
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       DriverType,
			DriverModel:      DriverModel,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location: req.ProtocolConfig.Location, BaudRate: req.ProtocolConfig.BaudRate,
		DataBits: req.ProtocolConfig.DataBits, Parity: req.ProtocolConfig.Parity,
		StopBits: req.ProtocolConfig.StopBits, Station: req.ProtocolConfig.Station,
		TimeoutMs: req.ProtocolConfig.TimeoutMs, IntervalMs: req.ProtocolConfig.IntervalMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.BaudRate = req.ProtocolConfig.BaudRate
	c.DataBits = req.ProtocolConfig.DataBits
	c.Parity = req.ProtocolConfig.Parity
	c.StopBits = req.ProtocolConfig.StopBits
	c.Station = req.ProtocolConfig.Station
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
	c.IntervalMs = req.ProtocolConfig.IntervalMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Location: c.Location, BaudRate: c.BaudRate, DataBits: c.DataBits, Parity: c.Parity,
		StopBits: c.StopBits, Station: c.Station, TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs}
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
	c.Parity, c.StopBits, c.Station = config.Parity, config.StopBits, config.Station
	c.TimeoutMs, c.IntervalMs = config.TimeoutMs, config.IntervalMs
	return nil
}

var _ collection.Collector = (*Collection)(nil)
