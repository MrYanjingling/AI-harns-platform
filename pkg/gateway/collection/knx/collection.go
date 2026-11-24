package knx

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "KNX"
	DriverModel = "KNXnet/IP"

	DefaultDiscoveryIP   = "224.0.23.12"
	DefaultDiscoveryPort = 3671
	DefaultTimeoutMs     = 3000
	DefaultIntervalMs    = 20
)

type ProtocolConfig struct {
	DiscoveryIP   string `json:"discoveryIp"`
	DiscoveryPort int    `json:"discoveryPort"`
	TimeoutMs     int    `json:"timeoutMs"`
	IntervalMs    int    `json:"intervalMs"`
}

func (c *ProtocolConfig) Normalize() {
	if c.DiscoveryIP == "" {
		c.DiscoveryIP = DefaultDiscoveryIP
	}
	if c.DiscoveryPort == 0 {
		c.DiscoveryPort = DefaultDiscoveryPort
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

	DiscoveryIP   string `json:"discoveryIp"`
	DiscoveryPort int    `json:"discoveryPort"`
	TimeoutMs     int    `json:"timeoutMs"`
	IntervalMs    int    `json:"intervalMs"`

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
		DiscoveryIP: req.ProtocolConfig.DiscoveryIP, DiscoveryPort: req.ProtocolConfig.DiscoveryPort,
		TimeoutMs: req.ProtocolConfig.TimeoutMs, IntervalMs: req.ProtocolConfig.IntervalMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.DiscoveryIP = req.ProtocolConfig.DiscoveryIP
	c.DiscoveryPort = req.ProtocolConfig.DiscoveryPort
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
	c.IntervalMs = req.ProtocolConfig.IntervalMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{DiscoveryIP: c.DiscoveryIP, DiscoveryPort: c.DiscoveryPort,
		TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.DiscoveryIP, c.DiscoveryPort = config.DiscoveryIP, config.DiscoveryPort
	c.TimeoutMs, c.IntervalMs = config.TimeoutMs, config.IntervalMs
	return nil
}

var _ collection.Collector = (*Collection)(nil)
