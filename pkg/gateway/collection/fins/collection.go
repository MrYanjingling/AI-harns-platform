// Package fins implements Omron FINS collection over TCP and UDP.
package fins

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType = "omron"
	ModelFins  = "fins"

	TransportTCP = "tcp"
	TransportUDP = "udp"

	DefaultPort          = 9600
	DefaultExpiresMs     = 500
	DefaultSlidingWindow = 1
	DefaultMaxSize       = 64

	MinMaxSize = 64
	MaxMaxSize = 1024
)

type ProtocolConfig struct {
	Location      string `json:"location"`
	Port          int    `json:"port"`
	Transport     string `json:"transport"`
	Type          int    `json:"type"`
	Expires       int    `json:"expires"`
	SlidingWindow int    `json:"slidingWindow"`
	MaxSize       int    `json:"maxSize"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	c.Transport = normalizeTransport(c.Transport)
	if c.Expires <= 0 {
		c.Expires = DefaultExpiresMs
	}
	if c.SlidingWindow <= 0 {
		c.SlidingWindow = DefaultSlidingWindow
	}
	if c.MaxSize <= 0 {
		c.MaxSize = DefaultMaxSize
	}
}

func normalizeTransport(value string) string {
	switch value {
	case TransportUDP:
		return TransportUDP
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

	Location      string `json:"location"`
	Port          int    `json:"port"`
	Transport     string `json:"transport"`
	Type          int    `json:"type"`
	Expires       int    `json:"expires"`
	SlidingWindow int    `json:"slidingWindow"`
	MaxSize       int    `json:"maxSize"`

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
			DriverModel:      ModelFins,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location:      req.ProtocolConfig.Location,
		Port:          req.ProtocolConfig.Port,
		Transport:     req.ProtocolConfig.Transport,
		Type:          req.ProtocolConfig.Type,
		Expires:       req.ProtocolConfig.Expires,
		SlidingWindow: req.ProtocolConfig.SlidingWindow,
		MaxSize:       req.ProtocolConfig.MaxSize,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Transport = req.ProtocolConfig.Transport
	c.Type = req.ProtocolConfig.Type
	c.Expires = req.ProtocolConfig.Expires
	c.SlidingWindow = req.ProtocolConfig.SlidingWindow
	c.MaxSize = req.ProtocolConfig.MaxSize
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location:      c.Location,
		Port:          c.Port,
		Transport:     c.Transport,
		Type:          c.Type,
		Expires:       c.Expires,
		SlidingWindow: c.SlidingWindow,
		MaxSize:       c.MaxSize,
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
	c.SlidingWindow = config.SlidingWindow
	c.MaxSize = config.MaxSize
	return nil
}

var _ collection.Collector = (*Collection)(nil)
