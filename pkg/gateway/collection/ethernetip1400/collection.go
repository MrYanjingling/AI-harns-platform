// Package ethernetip1400 implements EtherNet/IP MicroLogix 1400 PCCC collection.
package ethernetip1400

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "Allen-Bradley"
	DriverModel = "ethernetIp1400"

	DefaultPort           = 44818
	DefaultCIPSize        = 500
	DefaultLogixArraySize = 120
)

// ProtocolConfig contains EtherNet/IP MicroLogix 1400 options.
type ProtocolConfig struct {
	Location       string `json:"location"`
	Port           int    `json:"port"`
	CIPSize        int    `json:"cipSize"`
	LogixArraySize int    `json:"logixArraySize"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.CIPSize == 0 {
		c.CIPSize = DefaultCIPSize
	}
	if c.LogixArraySize == 0 {
		c.LogixArraySize = DefaultLogixArraySize
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

	Location       string `json:"location"`
	Port           int    `json:"port"`
	CIPSize        int    `json:"cipSize"`
	LogixArraySize int    `json:"logixArraySize"`

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
		Location:       req.ProtocolConfig.Location,
		Port:           req.ProtocolConfig.Port,
		CIPSize:        req.ProtocolConfig.CIPSize,
		LogixArraySize: req.ProtocolConfig.LogixArraySize,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.CIPSize = req.ProtocolConfig.CIPSize
	c.LogixArraySize = req.ProtocolConfig.LogixArraySize
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location:       c.Location,
		Port:           c.Port,
		CIPSize:        c.CIPSize,
		LogixArraySize: c.LogixArraySize,
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
	c.CIPSize = config.CIPSize
	c.LogixArraySize = config.LogixArraySize
	return nil
}

var _ collection.Collector = (*Collection)(nil)
