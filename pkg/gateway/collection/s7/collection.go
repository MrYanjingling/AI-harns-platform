// Package s7 implements Siemens S7-1500 collection.
package s7

import "harns-platform/pkg/gateway/collection"

const (
	DriverType  = "s7"
	DriverModel = "s71500"

	DefaultPort     = 102
	DefaultTCPCount = 1
	MaxTCPCount     = 3
)

type ProtocolConfig struct {
	Location string `json:"location"`
	Port     int    `json:"port"`
	Rack     uint8  `json:"rack"`
	Slot     uint8  `json:"slot"`
	TCPCount int    `json:"tcpCount"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
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

type S7Collection struct {
	collection.CollectionMeta

	Location string `json:"location"`
	Port     int    `json:"port"`
	Rack     uint8  `json:"rack"`
	Slot     uint8  `json:"slot"`
	TCPCount int    `json:"tcpCount"`

	runtime     *collectorRuntime
	frameTracer collection.FrameTracer
}

func NewCollection(req *CreateReq) (*S7Collection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &S7Collection{
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
		Rack:     req.ProtocolConfig.Rack,
		Slot:     req.ProtocolConfig.Slot,
		TCPCount: req.ProtocolConfig.TCPCount,
	}, nil
}

func (c *S7Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Rack = req.ProtocolConfig.Rack
	c.Slot = req.ProtocolConfig.Slot
	c.TCPCount = req.ProtocolConfig.TCPCount
}

var _ collection.Collector = (*S7Collection)(nil)
