// Package dnp3 implements DNP 3.0 TCP master collection.
package dnp3

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "dnp3"
	DriverModel = "dnp3"

	DefaultPort              = 20000
	DefaultMasterID          = 1
	DefaultSlaveID           = 2
	DefaultClass0123Interval = 20000
	DefaultClass123Interval  = 1000
)

type ProtocolConfig struct {
	Location          string `json:"location"`
	Port              int    `json:"port"`
	MasterID          int    `json:"masterId"`
	SlaveID           int    `json:"slaveId"`
	Class0123Interval int    `json:"class0123Interval"`
	Class123Interval  int    `json:"class123Interval"`
	TimeSync          int    `json:"timeSync"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.MasterID == 0 {
		c.MasterID = DefaultMasterID
	}
	if c.SlaveID == 0 {
		c.SlaveID = DefaultSlaveID
	}
	if c.Class0123Interval == 0 {
		c.Class0123Interval = DefaultClass0123Interval
	}
	if c.Class123Interval == 0 {
		c.Class123Interval = DefaultClass123Interval
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

	Location          string `json:"location"`
	Port              int    `json:"port"`
	MasterID          int    `json:"masterId"`
	SlaveID           int    `json:"slaveId"`
	Class0123Interval int    `json:"class0123Interval"`
	Class123Interval  int    `json:"class123Interval"`
	TimeSync          int    `json:"timeSync"`

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
		Location:          req.ProtocolConfig.Location,
		Port:              req.ProtocolConfig.Port,
		MasterID:          req.ProtocolConfig.MasterID,
		SlaveID:           req.ProtocolConfig.SlaveID,
		Class0123Interval: req.ProtocolConfig.Class0123Interval,
		Class123Interval:  req.ProtocolConfig.Class123Interval,
		TimeSync:          req.ProtocolConfig.TimeSync,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.MasterID = req.ProtocolConfig.MasterID
	c.SlaveID = req.ProtocolConfig.SlaveID
	c.Class0123Interval = req.ProtocolConfig.Class0123Interval
	c.Class123Interval = req.ProtocolConfig.Class123Interval
	c.TimeSync = req.ProtocolConfig.TimeSync
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location:          c.Location,
		Port:              c.Port,
		MasterID:          c.MasterID,
		SlaveID:           c.SlaveID,
		Class0123Interval: c.Class0123Interval,
		Class123Interval:  c.Class123Interval,
		TimeSync:          c.TimeSync,
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
	c.MasterID = config.MasterID
	c.SlaveID = config.SlaveID
	c.Class0123Interval = config.Class0123Interval
	c.Class123Interval = config.Class123Interval
	c.TimeSync = config.TimeSync
	return nil
}

var _ collection.Collector = (*Collection)(nil)
