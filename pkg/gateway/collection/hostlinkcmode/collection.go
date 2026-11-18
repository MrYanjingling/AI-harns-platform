// Package hostlinkcmode implements Omron HostLink C-Mode serial collection.
package hostlinkcmode

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "omron"
	DriverModel = "hostlinkCmode"

	DefaultBaudRate     = 9600
	DefaultDataBits     = 8
	DefaultTimeoutMs    = 3000
	DefaultIntervalMs   = 20
	DefaultMaxReadWords = 30
)

type ProtocolConfig struct {
	Location     string `json:"location"`
	BaudRate     int    `json:"baudRate"`
	DataBits     int    `json:"dataBits"`
	Parity       string `json:"parity"`
	StopBits     string `json:"stopBits"`
	TimeoutMs    int    `json:"timeoutMs"`
	IntervalMs   int    `json:"intervalMs"`
	MaxReadWords int    `json:"maxReadWords"`
}

func (c *ProtocolConfig) Normalize() {
	if c.BaudRate == 0 {
		c.BaudRate = DefaultBaudRate
	}
	if c.DataBits == 0 {
		c.DataBits = DefaultDataBits
	}
	if c.Parity == "" {
		c.Parity = "noParity"
	}
	if c.StopBits == "" {
		c.StopBits = "1"
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.IntervalMs <= 0 {
		c.IntervalMs = DefaultIntervalMs
	}
	if c.MaxReadWords <= 0 {
		c.MaxReadWords = DefaultMaxReadWords
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

	Location     string `json:"location"`
	BaudRate     int    `json:"baudRate"`
	DataBits     int    `json:"dataBits"`
	Parity       string `json:"parity"`
	StopBits     string `json:"stopBits"`
	TimeoutMs    int    `json:"timeoutMs"`
	IntervalMs   int    `json:"intervalMs"`
	MaxReadWords int    `json:"maxReadWords"`

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
		Location:     req.ProtocolConfig.Location,
		BaudRate:     req.ProtocolConfig.BaudRate,
		DataBits:     req.ProtocolConfig.DataBits,
		Parity:       req.ProtocolConfig.Parity,
		StopBits:     req.ProtocolConfig.StopBits,
		TimeoutMs:    req.ProtocolConfig.TimeoutMs,
		IntervalMs:   req.ProtocolConfig.IntervalMs,
		MaxReadWords: req.ProtocolConfig.MaxReadWords,
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
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
	c.IntervalMs = req.ProtocolConfig.IntervalMs
	c.MaxReadWords = req.ProtocolConfig.MaxReadWords
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location:     c.Location,
		BaudRate:     c.BaudRate,
		DataBits:     c.DataBits,
		Parity:       c.Parity,
		StopBits:     c.StopBits,
		TimeoutMs:    c.TimeoutMs,
		IntervalMs:   c.IntervalMs,
		MaxReadWords: c.MaxReadWords,
	}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.Location = config.Location
	c.BaudRate = config.BaudRate
	c.DataBits = config.DataBits
	c.Parity = config.Parity
	c.StopBits = config.StopBits
	c.TimeoutMs = config.TimeoutMs
	c.IntervalMs = config.IntervalMs
	c.MaxReadWords = config.MaxReadWords
	return nil
}

var _ collection.Collector = (*Collection)(nil)
