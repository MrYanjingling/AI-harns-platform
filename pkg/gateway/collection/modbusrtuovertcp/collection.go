// Package modbusrtuovertcp implements Modbus RTU over TCP collection.
package modbusrtuovertcp

import "harns-platform/pkg/gateway/collection"

const (
	DriverType  = "modbus"
	DriverModel = "modbusRtuOverTcp"

	DefaultTCPCount = 1
	MaxTCPCount     = 3
)

type ProtocolConfig struct {
	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
	TCPCount        int    `json:"tcpCount"`
}

func (c *ProtocolConfig) Normalize() {
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

type ModbusRtuOverTcpCollection struct {
	collection.CollectionMeta

	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
	TCPCount        int    `json:"tcpCount"`

	runtime      *collectorRuntime
	activeBackup bool
	frameTracer  collection.FrameTracer
}

func NewCollection(req *CreateReq) (*ModbusRtuOverTcpCollection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &ModbusRtuOverTcpCollection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       DriverType,
			DriverModel:      DriverModel,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location:        req.ProtocolConfig.Location,
		BackupLocation:  req.ProtocolConfig.BackupLocation,
		MemoryLayout:    req.ProtocolConfig.MemoryLayout,
		PositionAddress: req.ProtocolConfig.PositionAddress,
		TCPCount:        req.ProtocolConfig.TCPCount,
	}, nil
}

func (c *ModbusRtuOverTcpCollection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.BackupLocation = req.ProtocolConfig.BackupLocation
	c.MemoryLayout = req.ProtocolConfig.MemoryLayout
	c.PositionAddress = req.ProtocolConfig.PositionAddress
	c.TCPCount = req.ProtocolConfig.TCPCount
}

var _ collection.Collector = (*ModbusRtuOverTcpCollection)(nil)
