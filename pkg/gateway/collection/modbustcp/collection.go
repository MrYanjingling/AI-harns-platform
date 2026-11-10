// Package modbustcp defines the Modbus TCP collection model.
package modbustcp

import (
	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "modbus"
	DriverModel = "modbusTcp"
)

// CreateReq creates a Modbus TCP collection.
type CreateReq struct {
	Name             string                 `json:"name"`
	CollectorCycleMs uint64                 `json:"collectorCycleMs"`
	Variables        []*collection.Variable `json:"variables"`

	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
}

// ModbusTcpCollection is one Modbus TCP collector configuration.
type ModbusTcpCollection struct {
	collection.CollectionMeta

	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
}

var _ collection.Collector = (*ModbusTcpCollection)(nil)

// NewCollection builds a Modbus TCP collection from a create request.
func NewCollection(req *CreateReq) (*ModbusTcpCollection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
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
		Location:        req.Location,
		BackupLocation:  req.BackupLocation,
		MemoryLayout:    req.MemoryLayout,
		PositionAddress: req.PositionAddress,
	}, nil
}
