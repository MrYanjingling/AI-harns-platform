package knd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "CNC"
	DriverModel = "KND"

	DefaultPort       = 8000
	DefaultTimeoutMs  = 5000
	DefaultIntervalMs = 20
)

type ProtocolConfig struct {
	Location   string `json:"location,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	TimeoutMs  int    `json:"timeoutMs"`
	IntervalMs int    `json:"intervalMs"`
}

func (c *ProtocolConfig) Normalize() {
	if c.TimeoutMs == 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.IntervalMs == 0 {
		c.IntervalMs = DefaultIntervalMs
	}
	if strings.TrimSpace(c.Location) == "" && strings.TrimSpace(c.Host) != "" {
		port := c.Port
		if port == 0 {
			port = DefaultPort
		}
		c.Location = formatEndpoint(c.Host, port)
	}
}

func (c *ProtocolConfig) endpoint() (host string, port int, location string, err error) {
	c.Normalize()
	value := strings.TrimSpace(c.Location)
	if value == "" && strings.TrimSpace(c.Host) != "" {
		port = c.Port
		if port == 0 {
			port = DefaultPort
		}
		return strings.TrimSpace(c.Host), port, formatEndpoint(c.Host, port), nil
	}
	return parseEndpoint(value)
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

	Location   string `json:"location"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
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
	host, port, location, err := req.ProtocolConfig.endpoint()
	if err != nil {
		return nil, err
	}
	return &Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       DriverType,
			DriverModel:      DriverModel,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location: location, Host: host, Port: port,
		TimeoutMs: req.ProtocolConfig.TimeoutMs, IntervalMs: req.ProtocolConfig.IntervalMs,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	if host, port, location, err := req.ProtocolConfig.endpoint(); err == nil {
		c.Location, c.Host, c.Port = location, host, port
	} else {
		c.Location, c.Host, c.Port = strings.TrimSpace(req.ProtocolConfig.Location), "", 0
	}
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
	c.IntervalMs = req.ProtocolConfig.IntervalMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	location := strings.TrimSpace(c.Location)
	if location == "" {
		location = formatEndpoint(c.Host, c.Port)
	}
	config := ProtocolConfig{Location: location, TimeoutMs: c.TimeoutMs, IntervalMs: c.IntervalMs}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	host, port, location, err := config.endpoint()
	if err != nil {
		return err
	}
	c.Location, c.Host, c.Port = location, host, port
	c.TimeoutMs, c.IntervalMs = config.TimeoutMs, config.IntervalMs
	return nil
}

func parseEndpoint(value string) (string, int, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, "", fmt.Errorf("KND HTTP address is empty")
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" {
		return "", 0, "", fmt.Errorf("invalid KND HTTP address %q", value)
	}
	if parsed.Scheme != "http" {
		return "", 0, "", fmt.Errorf("KND HTTP address must use http")
	}
	port := DefaultPort
	if parsed.Port() != "" {
		port, err = strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return "", 0, "", fmt.Errorf("invalid KND HTTP port %q", parsed.Port())
		}
	}
	host := parsed.Hostname()
	return host, port, formatEndpoint(host, port), nil
}

func formatEndpoint(host string, port int) string {
	host = strings.TrimSpace(host)
	if port == 0 {
		port = DefaultPort
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}

var _ collection.Collector = (*Collection)(nil)
