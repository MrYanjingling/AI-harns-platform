// Package modbustcp defines the Modbus TCP collection model.
package modbustcp

import (
	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "modbus"
	DriverModel = "modbusTcp"

	DefaultConcurrency = 1
	MaxConcurrency     = 3
	DefaultTimeoutMs   = 3000
)

// ProtocolConfig contains Modbus TCP protocol fields.
type ProtocolConfig struct {
	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress"`
	Concurrency     int    `json:"concurrency"`
	TimeoutMs       int    `json:"timeoutMs"`
}

// Normalize applies default connection and timeout values.
func (c *ProtocolConfig) Normalize() {
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConcurrency
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
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
	PositionAddress uint   `json:"positionAddress"`
	Concurrency     int    `json:"concurrency"`
	TimeoutMs       int    `json:"timeoutMs"`

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
		Concurrency:     req.ProtocolConfig.Concurrency,
		TimeoutMs:       req.ProtocolConfig.TimeoutMs,
	}, nil
}
