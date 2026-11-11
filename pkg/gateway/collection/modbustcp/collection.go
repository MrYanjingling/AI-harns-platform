// Package modbustcp defines the Modbus TCP collection model.
package modbustcp

import (
	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "modbus"
	DriverModel = "modbusTcp"

	DefaultTCPCount = 1
	MaxTCPCount     = 3
)

// ProtocolConfig contains Modbus TCP protocol fields.
type ProtocolConfig struct {
	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
	TCPCount        int    `json:"tcpCount"`
}

// Normalize applies the default TCP connection count.
func (c *ProtocolConfig) Normalize() {
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
}

// CreateReq creates a Modbus TCP collection.
type CreateReq struct {
	Name             string                 `json:"name"`
	CollectorCycleMs uint64                 `json:"collectorCycleMs"`
	Variables        []*collection.Variable `json:"variables"`
	ProtocolConfig   ProtocolConfig         `json:"protocolConfig"`
}

// ModbusTcpCollection is one Modbus TCP collector configuration.
type ModbusTcpCollection struct {
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

var _ collection.Collector = (*ModbusTcpCollection)(nil)

// NewCollection builds a Modbus TCP collection from a create request.
func NewCollection(req *CreateReq) (*ModbusTcpCollection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &ModbusTcpCollection{
		CollectionMeta: collection.CollectionMeta{
			Meta: collection.Meta{
				Name: req.Name,
			},
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
