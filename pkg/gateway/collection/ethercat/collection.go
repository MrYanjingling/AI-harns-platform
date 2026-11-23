// Package ethercat implements EtherCAT process-data collection through a
// Beckhoff TwinCAT ADS gateway. EtherCAT itself is not an IP protocol; TwinCAT
// exposes the mapped process image through ADS index groups, which this driver
// reads and writes. Addressing, index-group validation and data types follow
// Neuron's ADS plugin.
package ethercat

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "ethercat"
	DriverModel = "ethercat"

	DefaultPort         = 48898
	DefaultAMSPort      = 851
	DefaultLocalAMSPort = 851
	DefaultTimeoutMs    = 3000

	MinADSPort = 800
	MaxADSPort = 899
)

type ProtocolConfig struct {
	Location      string `json:"location"`
	Port          int    `json:"port"`
	AMSNetID      string `json:"amsNetId"`
	AMSPort       int    `json:"amsPort"`
	LocalAMSNetID string `json:"localAmsNetId"`
	LocalAMSPort  int    `json:"localAmsPort"`
	TimeoutMs     int    `json:"timeoutMs"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.AMSPort == 0 {
		c.AMSPort = DefaultAMSPort
	}
	if c.LocalAMSPort == 0 {
		c.LocalAMSPort = DefaultLocalAMSPort
	}
	if c.TimeoutMs == 0 {
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

type Collection struct {
	collection.CollectionMeta

	Location      string `json:"location"`
	Port          int    `json:"port"`
	AMSNetID      string `json:"amsNetId"`
	AMSPort       int    `json:"amsPort"`
	LocalAMSNetID string `json:"localAmsNetId"`
	LocalAMSPort  int    `json:"localAmsPort"`
	TimeoutMs     int    `json:"timeoutMs"`

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
		Location:      req.ProtocolConfig.Location,
		Port:          req.ProtocolConfig.Port,
		AMSNetID:      req.ProtocolConfig.AMSNetID,
		AMSPort:       req.ProtocolConfig.AMSPort,
		LocalAMSNetID: req.ProtocolConfig.LocalAMSNetID,
		LocalAMSPort:  req.ProtocolConfig.LocalAMSPort,
		TimeoutMs:     req.ProtocolConfig.TimeoutMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.AMSNetID = req.ProtocolConfig.AMSNetID
	c.AMSPort = req.ProtocolConfig.AMSPort
	c.LocalAMSNetID = req.ProtocolConfig.LocalAMSNetID
	c.LocalAMSPort = req.ProtocolConfig.LocalAMSPort
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location:      c.Location,
		Port:          c.Port,
		AMSNetID:      c.AMSNetID,
		AMSPort:       c.AMSPort,
		LocalAMSNetID: c.LocalAMSNetID,
		LocalAMSPort:  c.LocalAMSPort,
		TimeoutMs:     c.TimeoutMs,
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
	c.AMSNetID = config.AMSNetID
	c.AMSPort = config.AMSPort
	c.LocalAMSNetID = config.LocalAMSNetID
	c.LocalAMSPort = config.LocalAMSPort
	c.TimeoutMs = config.TimeoutMs
	return nil
}

var _ collection.Collector = (*Collection)(nil)
