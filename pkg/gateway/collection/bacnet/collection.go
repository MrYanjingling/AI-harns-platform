package bacnet

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "BACnet"
	DriverModel = "BACnet"

	DefaultPort       = 47808
	DefaultTimeoutMs  = 3000
	DefaultIntervalMs = 20
)

type ProtocolConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	SrcPort    int    `json:"srcPort"`
	BBMD       int    `json:"bbmd"`
	TimeoutMs  int    `json:"timeoutMs"`
	IntervalMs int    `json:"intervalMs"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.TimeoutMs == 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.IntervalMs == 0 {
		c.IntervalMs = DefaultIntervalMs
	}
	if c.BBMD != 1 {
		c.BBMD = 0
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

	Host       string `json:"host"`
	Port       int    `json:"port"`
	SrcPort    int    `json:"srcPort"`
	BBMD       int    `json:"bbmd"`
	TimeoutMs  int    `json:"timeoutMs"`
	IntervalMs int    `json:"intervalMs"`

	runtime *collectorRuntime
}

func NewCollection(req *CreateReq) (*Collection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta: collection.Meta{Name: req.Name}, DriverType: DriverType, DriverModel: DriverModel,
			CollectStatus: collection.CollectStatusStopped, CollectorCycleMs: req.CollectorCycleMs, Variables: req.Variables,
		},
		Host: req.ProtocolConfig.Host, Port: req.ProtocolConfig.Port, SrcPort: req.ProtocolConfig.SrcPort,
		BBMD: req.ProtocolConfig.BBMD, TimeoutMs: req.ProtocolConfig.TimeoutMs, IntervalMs: req.ProtocolConfig.IntervalMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Host, c.Port, c.SrcPort = req.ProtocolConfig.Host, req.ProtocolConfig.Port, req.ProtocolConfig.SrcPort
	c.BBMD, c.TimeoutMs, c.IntervalMs = req.ProtocolConfig.BBMD, req.ProtocolConfig.TimeoutMs, req.ProtocolConfig.IntervalMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Host: c.Host, Port: c.Port, SrcPort: c.SrcPort, BBMD: c.BBMD, TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.Host, c.Port, c.SrcPort = config.Host, config.Port, config.SrcPort
	c.BBMD, c.TimeoutMs, c.IntervalMs = config.BBMD, config.TimeoutMs, config.IntervalMs
	return nil
}
