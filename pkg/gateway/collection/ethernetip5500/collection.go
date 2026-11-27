// Package ethernetip5500 implements EtherNet/IP ControlLogix 5500 CIP collection.
package ethernetip5500

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "Allen-Bradley"
	DriverModel = "EtherNet/IP-ControlLogic 55xx"

	DefaultPort     = 44818
	DefaultSlot     = 0
	DefaultInterval = 20
	MaxSlot         = 255
	MinInterval     = 20
	MaxInterval     = 60000
)

// ProtocolConfig contains EtherNet/IP ControlLogix 5500 connection options.
type ProtocolConfig struct {
	Location string `json:"location"`
	Port     int    `json:"port"`
	Slot     int    `json:"slot"`
	Interval int    `json:"interval"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.Interval == 0 {
		c.Interval = DefaultInterval
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

	Location string `json:"location"`
	Port     int    `json:"port"`
	Slot     int    `json:"slot"`
	Interval int    `json:"interval"`

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
		Location: req.ProtocolConfig.Location,
		Port:     req.ProtocolConfig.Port,
		Slot:     req.ProtocolConfig.Slot,
		Interval: req.ProtocolConfig.Interval,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Slot = req.ProtocolConfig.Slot
	c.Interval = req.ProtocolConfig.Interval
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location: c.Location,
		Port:     c.Port,
		Slot:     c.Slot,
		Interval: c.Interval,
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
	c.Port = config.Port
	c.Slot = config.Slot
	c.Interval = config.Interval
	return nil
}

var _ collection.Collector = (*Collection)(nil)
