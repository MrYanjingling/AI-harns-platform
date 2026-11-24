// Package mewtocol implements Panasonic Mewtocol ASCII collection over TCP.
package mewtocol

import (
	"encoding/json"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "Panasonic"
	DriverModel = "Mewtocol"

	DefaultPort      = 2000
	DefaultUnit      = 1
	DefaultTimeoutMs = 15000
)

// ProtocolConfig contains Mewtocol TCP configuration.
type ProtocolConfig struct {
	Location  string `json:"location"`
	Port      int    `json:"port"`
	Unit      int    `json:"unit"`
	TimeoutMs int    `json:"timeoutMs"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.Unit == 0 {
		c.Unit = DefaultUnit
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
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

// Collection is one Mewtocol collector configuration.
type Collection struct {
	collection.CollectionMeta

	Location  string `json:"location"`
	Port      int    `json:"port"`
	Unit      int    `json:"unit"`
	TimeoutMs int    `json:"timeoutMs"`

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
		Location:  strings.TrimSpace(req.ProtocolConfig.Location),
		Port:      req.ProtocolConfig.Port,
		Unit:      req.ProtocolConfig.Unit,
		TimeoutMs: req.ProtocolConfig.TimeoutMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = strings.TrimSpace(req.ProtocolConfig.Location)
	c.Port = req.ProtocolConfig.Port
	c.Unit = req.ProtocolConfig.Unit
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location:  c.Location,
		Port:      c.Port,
		Unit:      c.Unit,
		TimeoutMs: c.TimeoutMs,
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
	c.Location = strings.TrimSpace(config.Location)
	c.Port = config.Port
	c.Unit = config.Unit
	c.TimeoutMs = config.TimeoutMs
	return nil
}

var _ collection.Collector = (*Collection)(nil)
var _ collection.ActionHandler = (*Collection)(nil)
var _ collection.FrameTracerAware = (*Collection)(nil)
