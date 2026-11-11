// Package modbusrtu implements Modbus RTU serial collection.
package modbusrtu

import (
	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "modbus"
	DriverModel = "modbusRtu"
)

type ProtocolConfig struct {
	Location        string `json:"location"`
	BaudRate        int    `json:"baudRate"`
	DataBits        int    `json:"dataBits"`
	Parity          string `json:"parity"`
	StopBits        string `json:"stopBits"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
}

func (c *ProtocolConfig) Normalize() {
	if c.BaudRate == 0 {
		c.BaudRate = 9600
	}
	if c.DataBits == 0 {
		c.DataBits = 8
	}
	if c.Parity == "" {
		c.Parity = "noParity"
	}
	if c.StopBits == "" {
		c.StopBits = "1"
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

type ModbusRtuCollection struct {
	collection.CollectionMeta

	Location        string `json:"location"`
	BaudRate        int    `json:"baudRate"`
	DataBits        int    `json:"dataBits"`
	Parity          string `json:"parity"`
	StopBits        string `json:"stopBits"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`

	runtime     *collectorRuntime
	frameTracer collection.FrameTracer
}

func NewCollection(req *CreateReq) (*ModbusRtuCollection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &ModbusRtuCollection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       DriverType,
			DriverModel:      DriverModel,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location:        req.ProtocolConfig.Location,
		BaudRate:        req.ProtocolConfig.BaudRate,
		DataBits:        req.ProtocolConfig.DataBits,
		Parity:          req.ProtocolConfig.Parity,
		StopBits:        req.ProtocolConfig.StopBits,
		MemoryLayout:    req.ProtocolConfig.MemoryLayout,
		PositionAddress: req.ProtocolConfig.PositionAddress,
	}, nil
}

func (c *ModbusRtuCollection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.BaudRate = req.ProtocolConfig.BaudRate
	c.DataBits = req.ProtocolConfig.DataBits
	c.Parity = req.ProtocolConfig.Parity
	c.StopBits = req.ProtocolConfig.StopBits
	c.MemoryLayout = req.ProtocolConfig.MemoryLayout
	c.PositionAddress = req.ProtocolConfig.PositionAddress
}

var _ collection.Collector = (*ModbusRtuCollection)(nil)
