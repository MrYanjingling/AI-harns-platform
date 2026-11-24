package mtconnect

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "CNC"
	DriverModel = "MTConnect"

	DefaultPort       = 5000
	DefaultTimeoutMs  = 5000
	DefaultIntervalMs = 20
)

type ProtocolConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	NSPrefix   string `json:"nsPrefix"`
	NSURI      string `json:"nsUri"`
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
	NSPrefix   string `json:"nsPrefix"`
	NSURI      string `json:"nsUri"`
	TimeoutMs  int    `json:"timeoutMs"`
	IntervalMs int    `json:"intervalMs"`

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
			Meta: collection.Meta{Name: req.Name}, DriverType: DriverType, DriverModel: DriverModel,
			CollectStatus: collection.CollectStatusStopped, CollectorCycleMs: req.CollectorCycleMs, Variables: req.Variables,
		},
		Host: req.ProtocolConfig.Host, Port: req.ProtocolConfig.Port,
		NSPrefix: req.ProtocolConfig.NSPrefix, NSURI: req.ProtocolConfig.NSURI,
		TimeoutMs: req.ProtocolConfig.TimeoutMs, IntervalMs: req.ProtocolConfig.IntervalMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Host, c.Port = req.ProtocolConfig.Host, req.ProtocolConfig.Port
	c.NSPrefix, c.NSURI = req.ProtocolConfig.NSPrefix, req.ProtocolConfig.NSURI
	c.TimeoutMs, c.IntervalMs = req.ProtocolConfig.TimeoutMs, req.ProtocolConfig.IntervalMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Host: c.Host, Port: c.Port, NSPrefix: c.NSPrefix, NSURI: c.NSURI, TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.Host, c.Port = config.Host, config.Port
	c.NSPrefix, c.NSURI = config.NSPrefix, config.NSURI
	c.TimeoutMs, c.IntervalMs = config.TimeoutMs, config.IntervalMs
	return nil
}

var _ collection.Collector = (*Collection)(nil)
