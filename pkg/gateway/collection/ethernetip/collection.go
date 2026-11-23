// Package ethernetip implements EtherNet/IP CIP collection.
//
// The driver follows the Neuron EtherNet/IP plugin: it registers an
// encapsulation session, opens a class-3 connection with Forward Open and
// exchanges each tag over connected SendUnitData messaging. Tag addresses are
// CIP symbolic paths, one ANSI extended-symbol segment per dot-separated name.
package ethernetip

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "ethernetIp"
	DriverModel = "ethernetIp"

	DefaultPort = 44818
)

type ProtocolConfig struct {
	Location string `json:"location"`
	Port     int    `json:"port"`
	// Slot is kept for compatibility with Neuron's EtherNet/IP driver
	// configuration. The generic Forward Open used here does not carry a slot;
	// 5500-style Large Forward Open is a separate plugin in Neuron.
	Slot int `json:"slot"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
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
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Slot = req.ProtocolConfig.Slot
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Location: c.Location, Port: c.Port, Slot: c.Slot}
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
	return nil
}

var _ collection.Collector = (*Collection)(nil)
