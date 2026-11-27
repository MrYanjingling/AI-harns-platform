// Package fins implements Omron FINS collection over TCP and UDP.
package fins

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType        = "omron"
	ModelFins         = "fins"
	ModelHostlinkFins = "hostlinkFins"

	TransportTCP      = "tcp"
	TransportUDP      = "udp"
	TransportHostlink = "hostlink"

	DefaultPort        = 9600
	DefaultExpiresMs   = 500
	DefaultConcurrency = 1
	DefaultMaxSize     = 64

	MinMaxSize     = 64
	MaxMaxSize     = 1024
	MaxConcurrency = 3
)

type ProtocolConfig struct {
	Location    string `json:"location"`
	Port        int    `json:"port"`
	Concurrency int    `json:"concurrency"`
	Transport   string `json:"transport"`
	Type        int    `json:"type"`
	Expires     int    `json:"expires"`
	MaxSize     int    `json:"maxSize"`
	Unit        int    `json:"unit,omitempty"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConcurrency
	}
	c.Transport = normalizeTransport(c.Transport)
	if c.Expires <= 0 {
		c.Expires = DefaultExpiresMs
	}
	if c.MaxSize <= 0 {
		c.MaxSize = DefaultMaxSize
	}
}

func normalizeTransport(value string) string {
	switch value {
	case TransportUDP:
		return TransportUDP
	case TransportHostlink:
		return TransportHostlink
	default:
		return TransportTCP
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

	Location    string `json:"location"`
	Port        int    `json:"port"`
	Transport   string `json:"transport"`
	Type        int    `json:"type"`
	Expires     int    `json:"expires"`
	Concurrency int    `json:"concurrency"`
	MaxSize     int    `json:"maxSize"`
	Unit        int    `json:"unit,omitempty"`

	runtime     *collectorRuntime
	frameTracer collection.FrameTracer
}

func NewCollection(req *CreateReq) (*Collection, error) {
	return NewCollectionForModel(ModelFins, req)
}

func NewCollectionForModel(model string, req *CreateReq) (*Collection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	if model == "" {
		model = ModelFins
	}
	if model == ModelHostlinkFins && req.ProtocolConfig.Transport == "" {
		req.ProtocolConfig.Transport = TransportHostlink
	}
	req.ProtocolConfig.Normalize()
	return &Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       DriverType,
			DriverModel:      model,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location:    req.ProtocolConfig.Location,
		Port:        req.ProtocolConfig.Port,
		Transport:   req.ProtocolConfig.Transport,
		Type:        req.ProtocolConfig.Type,
		Expires:     req.ProtocolConfig.Expires,
		Concurrency: req.ProtocolConfig.Concurrency,
		MaxSize:     req.ProtocolConfig.MaxSize,
		Unit:        req.ProtocolConfig.Unit,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	if c.DriverModel == ModelHostlinkFins && req.ProtocolConfig.Transport == "" {
		req.ProtocolConfig.Transport = TransportHostlink
	}
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Transport = req.ProtocolConfig.Transport
	c.Type = req.ProtocolConfig.Type
	c.Expires = req.ProtocolConfig.Expires
	c.Concurrency = req.ProtocolConfig.Concurrency
	c.MaxSize = req.ProtocolConfig.MaxSize
	c.Unit = req.ProtocolConfig.Unit
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location:    c.Location,
		Port:        c.Port,
		Transport:   c.Transport,
		Type:        c.Type,
		Expires:     c.Expires,
		Concurrency: c.Concurrency,
		MaxSize:     c.MaxSize,
		Unit:        c.Unit,
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
	c.Transport = config.Transport
	c.Type = config.Type
	c.Expires = config.Expires
	c.Concurrency = config.Concurrency
	c.MaxSize = config.MaxSize
	c.Unit = config.Unit
	return nil
}

var _ collection.Collector = (*Collection)(nil)
