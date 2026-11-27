// Package fetchwrite implements a Siemens S5 Fetch/Write TCP collector using connpool.
package fetchwrite

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "siemens"
	DriverModel = "FetchWrite"

	DefaultPort        = 102
	DefaultTimeoutMs   = 3000
	DefaultConcurrency = 1
	MaxConcurrency     = 4
)

type ProtocolConfig struct {
	Location    string `json:"location"`
	Port        int    `json:"port"`
	TimeoutMs   int    `json:"timeoutMs"`
	Concurrency int    `json:"concurrency"`
}

func (c *ProtocolConfig) Normalize() {
	c.Location = strings.TrimSpace(c.Location)
	if host, portText, err := net.SplitHostPort(c.Location); err == nil {
		if parsed, convErr := strconv.Atoi(portText); convErr == nil && parsed > 0 && parsed <= 65535 {
			c.Location = strings.TrimSpace(host)
			if c.Port <= 0 {
				c.Port = parsed
			}
		}
	}
	if c.Port <= 0 {
		c.Port = DefaultPort
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.Concurrency <= 0 {
		c.Concurrency = DefaultConcurrency
	}
	if c.Concurrency > MaxConcurrency {
		c.Concurrency = MaxConcurrency
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
	TimeoutMs   int    `json:"timeoutMs"`
	Concurrency int    `json:"concurrency"`

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
		Location:    req.ProtocolConfig.Location,
		Port:        req.ProtocolConfig.Port,
		TimeoutMs:   req.ProtocolConfig.TimeoutMs,
		Concurrency: req.ProtocolConfig.Concurrency,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
	c.Concurrency = req.ProtocolConfig.Concurrency
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location:    c.Location,
		Port:        c.Port,
		TimeoutMs:   c.TimeoutMs,
		Concurrency: c.Concurrency,
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
	c.TimeoutMs = config.TimeoutMs
	c.Concurrency = config.Concurrency
	return nil
}

func (c *Collection) endpoint() string {
	port := c.Port
	if port <= 0 {
		port = DefaultPort
	}
	return net.JoinHostPort(strings.TrimSpace(c.Location), strconv.Itoa(port))
}

var _ collection.Collector = (*Collection)(nil)
