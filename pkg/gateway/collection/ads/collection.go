package ads

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "TwinCAT"
	DriverModel = "Beckhoff ADS"

	DefaultPort       = 48898
	DefaultADSPort    = 851
	DefaultTimeoutMs  = 5000
	DefaultIntervalMs = 20
)

type ProtocolConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	SrcAMSNetID string `json:"srcAmsNetId"`
	SrcADSPort  int    `json:"srcAdsPort"`
	DstAMSNetID string `json:"dstAmsNetId"`
	DstADSPort  int    `json:"dstAdsPort"`
	TimeoutMs   int    `json:"timeoutMs"`
	IntervalMs  int    `json:"intervalMs"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.SrcADSPort == 0 {
		c.SrcADSPort = DefaultADSPort
	}
	if c.DstADSPort == 0 {
		c.DstADSPort = DefaultADSPort
	}
	if c.TimeoutMs == 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.IntervalMs == 0 {
		c.IntervalMs = DefaultIntervalMs
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

	Host        string `json:"host"`
	Port        int    `json:"port"`
	SrcAMSNetID string `json:"srcAmsNetId"`
	SrcADSPort  int    `json:"srcAdsPort"`
	DstAMSNetID string `json:"dstAmsNetId"`
	DstADSPort  int    `json:"dstAdsPort"`
	TimeoutMs   int    `json:"timeoutMs"`
	IntervalMs  int    `json:"intervalMs"`

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
		Host:        req.ProtocolConfig.Host,
		Port:        req.ProtocolConfig.Port,
		SrcAMSNetID: req.ProtocolConfig.SrcAMSNetID,
		SrcADSPort:  req.ProtocolConfig.SrcADSPort,
		DstAMSNetID: req.ProtocolConfig.DstAMSNetID,
		DstADSPort:  req.ProtocolConfig.DstADSPort,
		TimeoutMs:   req.ProtocolConfig.TimeoutMs,
		IntervalMs:  req.ProtocolConfig.IntervalMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Host = req.ProtocolConfig.Host
	c.Port = req.ProtocolConfig.Port
	c.SrcAMSNetID = req.ProtocolConfig.SrcAMSNetID
	c.SrcADSPort = req.ProtocolConfig.SrcADSPort
	c.DstAMSNetID = req.ProtocolConfig.DstAMSNetID
	c.DstADSPort = req.ProtocolConfig.DstADSPort
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
	c.IntervalMs = req.ProtocolConfig.IntervalMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Host: c.Host, Port: c.Port, SrcAMSNetID: c.SrcAMSNetID, SrcADSPort: c.SrcADSPort,
		DstAMSNetID: c.DstAMSNetID, DstADSPort: c.DstADSPort, TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.Host = config.Host
	c.Port = config.Port
	c.SrcAMSNetID = config.SrcAMSNetID
	c.SrcADSPort = config.SrcADSPort
	c.DstAMSNetID = config.DstAMSNetID
	c.DstADSPort = config.DstADSPort
	c.TimeoutMs = config.TimeoutMs
	c.IntervalMs = config.IntervalMs
	return nil
}
