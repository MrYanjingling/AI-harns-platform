package fxlinks

import (
	"encoding/json"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "Mitsubishi"
	DriverModel = "FX-Links"

	DefaultPort       = 8000
	DefaultBaudRate   = 9600
	DefaultDataBits   = 7
	DefaultParity     = "evenParity"
	DefaultStopBits   = "1"
	DefaultTimeoutMs  = 3000
	DefaultIntervalMs = 20
)

type ProtocolConfig struct {
	Link       int    `json:"link"` // 0=serial, 1=tcp
	Location   string `json:"location"`
	Port       int    `json:"port"`
	Station    int    `json:"station"`
	Format     int    `json:"format"`   // 1 or 4
	WaitTime   int    `json:"waitTime"` // 0..15 (x10ms)
	SumCheck   int    `json:"sumCheck"` // 1=enabled, 0=disabled
	BaudRate   int    `json:"baudRate"`
	DataBits   int    `json:"dataBits"`
	Parity     string `json:"parity"`
	StopBits   string `json:"stopBits"`
	TimeoutMs  int    `json:"timeoutMs"`
	IntervalMs int    `json:"intervalMs"`
}

func (c *ProtocolConfig) Normalize() {
	c.Location = strings.TrimSpace(c.Location)
	if c.Link != 0 && c.Link != 1 {
		c.Link = 1
	}
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = DefaultPort
	}
	if c.Station < 0 || c.Station > 15 {
		c.Station = 0
	}
	if c.Format != 1 && c.Format != 4 {
		c.Format = 1
	}
	if c.WaitTime < 0 || c.WaitTime > 15 {
		c.WaitTime = 0
	}
	if c.BaudRate <= 0 {
		c.BaudRate = DefaultBaudRate
	}
	if c.DataBits <= 0 {
		c.DataBits = DefaultDataBits
	}
	if strings.TrimSpace(c.Parity) == "" {
		c.Parity = DefaultParity
	}
	if strings.TrimSpace(c.StopBits) == "" {
		c.StopBits = DefaultStopBits
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.IntervalMs < 0 {
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

	Link       int    `json:"link"`
	Location   string `json:"location"`
	Port       int    `json:"port"`
	Station    int    `json:"station"`
	Format     int    `json:"format"`
	WaitTime   int    `json:"waitTime"`
	SumCheck   int    `json:"sumCheck"`
	BaudRate   int    `json:"baudRate"`
	DataBits   int    `json:"dataBits"`
	Parity     string `json:"parity"`
	StopBits   string `json:"stopBits"`
	TimeoutMs  int    `json:"timeoutMs"`
	IntervalMs int    `json:"intervalMs"`

	runtime *collectorRuntime
}

func NewCollection(req *CreateReq) (*Collection, error) {
	cfg := req.ProtocolConfig
	if cfg.Location != "" && cfg.Link == 0 && strings.Contains(cfg.Location, ".") && !strings.HasPrefix(cfg.Location, "/") {
		cfg.Link = 1
	}
	cfg.Normalize()
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	item := &Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: strings.TrimSpace(req.Name)},
			DriverType:       DriverType,
			DriverModel:      DriverModel,
			CollectorCycleMs: req.CollectorCycleMs,
			CollectStatus:    collection.CollectStatusStopped,
			Variables:        req.Variables,
		},
	}
	item.applyNormalized(cfg)
	return item, nil
}

func (c *Collection) applyNormalized(cfg ProtocolConfig) {
	c.Link = cfg.Link
	c.Location = cfg.Location
	c.Port = cfg.Port
	c.Station = cfg.Station
	c.Format = cfg.Format
	c.WaitTime = cfg.WaitTime
	c.SumCheck = cfg.SumCheck
	c.BaudRate = cfg.BaudRate
	c.DataBits = cfg.DataBits
	c.Parity = cfg.Parity
	c.StopBits = cfg.StopBits
	c.TimeoutMs = cfg.TimeoutMs
	c.IntervalMs = cfg.IntervalMs
}

func (c *Collection) ApplyConfig(raw json.RawMessage) error {
	var cfg ProtocolConfig
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return err
		}
	}
	cfg.Normalize()
	c.applyNormalized(cfg)
	return nil
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	cfg := ProtocolConfig{
		Link: c.Link, Location: c.Location, Port: c.Port, Station: c.Station,
		Format: c.Format, WaitTime: c.WaitTime, SumCheck: c.SumCheck,
		BaudRate: c.BaudRate, DataBits: c.DataBits, Parity: c.Parity, StopBits: c.StopBits,
		TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs,
	}
	cfg.Normalize()
	return json.Marshal(cfg)
}

func (c *Collection) GetCollectionMeta() *collection.CollectionMeta { return &c.CollectionMeta }
func (c *Collection) SetCollectionMeta(meta *collection.CollectionMeta) {
	if meta != nil {
		c.CollectionMeta = *meta
	}
}
func (c *Collection) SetFrameTracer(tracer collection.FrameTracer) {}
