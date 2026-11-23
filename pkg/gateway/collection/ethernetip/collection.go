// Package ethernetip implements EtherNet/IP CIP collection.
package ethernetip

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "ethernetIp"
	DriverModel = "ethernetIp"

	DefaultPort        = 44818
	DefaultConcurrency = 1
	MaxConcurrency     = 3
)

type ProtocolConfig struct {
	Location    string `json:"location"`
	Port        int    `json:"port"`
	Concurrency int    `json:"concurrency"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConcurrency
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

	Location    string `json:"location"`
	Port        int    `json:"port"`
	Concurrency int    `json:"concurrency"`

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
		Location:    req.ProtocolConfig.Location,
		Port:        req.ProtocolConfig.Port,
		Concurrency: req.ProtocolConfig.Concurrency,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Concurrency = req.ProtocolConfig.Concurrency
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Location: c.Location, Port: c.Port, Concurrency: c.Concurrency}
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
	c.Concurrency = config.Concurrency
	return nil
}

var _ collection.Collector = (*Collection)(nil)
