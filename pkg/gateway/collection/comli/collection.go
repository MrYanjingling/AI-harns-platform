// Package comli implements ABB COMLI serial collection.
package comli

import (
	"encoding/json"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "ABB"
	DriverModel = "COMLI"

	DefaultBaudRate   = 9600
	DefaultDataBits   = 8
	DefaultParity     = 0
	DefaultStopBits   = 1
	DefaultTimeoutMs  = 3000
	DefaultIntervalMs = 20
)

type ProtocolConfig struct {
	Device     string `json:"device"`
	BaudRate   int    `json:"baud"`
	DataBits   int    `json:"data"`
	Parity     int    `json:"parity"`
	StopBits   int    `json:"stop"`
	TimeoutMs  int    `json:"timeout"`
	IntervalMs int    `json:"interval"`
	Retry      int    `json:"retry"`
}

func (c *ProtocolConfig) Normalize() {
	c.Device = strings.TrimSpace(c.Device)
	if c.BaudRate <= 0 {
		c.BaudRate = DefaultBaudRate
	}
	if c.DataBits <= 0 {
		c.DataBits = DefaultDataBits
	}
	if c.Parity < 0 || c.Parity > 2 {
		c.Parity = DefaultParity
	}
	if c.StopBits <= 0 {
		c.StopBits = DefaultStopBits
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.IntervalMs <= 0 {
		c.IntervalMs = DefaultIntervalMs
	}
	if c.Retry < 0 {
		c.Retry = 0
	}
	if c.Retry > 10 {
		c.Retry = 10
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

	Device     string `json:"device"`
	BaudRate   int    `json:"baud"`
	DataBits   int    `json:"data"`
	Parity     int    `json:"parity"`
	StopBits   int    `json:"stop"`
	TimeoutMs  int    `json:"timeout"`
	IntervalMs int    `json:"interval"`
	Retry      int    `json:"retry"`

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
		Device:     req.ProtocolConfig.Device,
		BaudRate:   req.ProtocolConfig.BaudRate,
		DataBits:   req.ProtocolConfig.DataBits,
		Parity:     req.ProtocolConfig.Parity,
		StopBits:   req.ProtocolConfig.StopBits,
		TimeoutMs:  req.ProtocolConfig.TimeoutMs,
		IntervalMs: req.ProtocolConfig.IntervalMs,
		Retry:      req.ProtocolConfig.Retry,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Device = req.ProtocolConfig.Device
	c.BaudRate = req.ProtocolConfig.BaudRate
	c.DataBits = req.ProtocolConfig.DataBits
	c.Parity = req.ProtocolConfig.Parity
	c.StopBits = req.ProtocolConfig.StopBits
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
	c.IntervalMs = req.ProtocolConfig.IntervalMs
	c.Retry = req.ProtocolConfig.Retry
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Device:     c.Device,
		BaudRate:   c.BaudRate,
		DataBits:   c.DataBits,
		Parity:     c.Parity,
		StopBits:   c.StopBits,
		TimeoutMs:  c.TimeoutMs,
		IntervalMs: c.IntervalMs,
		Retry:      c.Retry,
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
	c.Device = config.Device
	c.BaudRate = config.BaudRate
	c.DataBits = config.DataBits
	c.Parity = config.Parity
	c.StopBits = config.StopBits
	c.TimeoutMs = config.TimeoutMs
	c.IntervalMs = config.IntervalMs
	c.Retry = config.Retry
	return nil
}
