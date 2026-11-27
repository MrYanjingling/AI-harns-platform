package cclinkie

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "Mitsubishi"
	DriverModel = "CC-Link IE Field Basic"

	DefaultPort             = 61450
	DefaultOccupiedStations = 1
	DefaultGroupNo          = 1
	DefaultTimeoutMs        = 3000
)

type ProtocolConfig struct {
	Location         string `json:"location"`
	OccupiedStations int    `json:"occupiedStations"`
	GroupNo          int    `json:"groupNo"`
	TimeoutMs        int    `json:"timeoutMs"`
}

func (c *ProtocolConfig) Normalize() {
	c.Location = strings.TrimSpace(c.Location)
	if c.OccupiedStations < 1 || c.OccupiedStations > 4 {
		c.OccupiedStations = DefaultOccupiedStations
	}
	if c.GroupNo < 1 || c.GroupNo > 64 {
		c.GroupNo = DefaultGroupNo
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
}

func (c *ProtocolConfig) endpoint() (string, string) {
	loc := strings.TrimSpace(c.Location)
	if loc == "" {
		return "", ""
	}
	if host, port, err := net.SplitHostPort(loc); err == nil {
		return host, net.JoinHostPort(host, port)
	}
	return loc, net.JoinHostPort(loc, strconv.Itoa(DefaultPort))
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

	Location         string `json:"location"`
	OccupiedStations int    `json:"occupiedStations"`
	GroupNo          int    `json:"groupNo"`
	TimeoutMs        int    `json:"timeoutMs"`

	runtime *collectorRuntime
}

func NewCollection(req *CreateReq) (*Collection, error) {
	cfg := req.ProtocolConfig
	cfg.Normalize()
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	item := &Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: strings.TrimSpace(req.Name)},
			DriverType:       DriverType,
			DriverModel:      DriverModel,
			CollectorCycleMs: req.CollectorCycleMs,
			CollectStatus:    collection.CollectStatusStopped,
			Variables:        req.Variables,
		},
		Location:         cfg.Location,
		OccupiedStations: cfg.OccupiedStations,
		GroupNo:          cfg.GroupNo,
		TimeoutMs:        cfg.TimeoutMs,
	}
	return item, nil
}

func (c *Collection) ApplyConfig(raw json.RawMessage) error {
	var cfg ProtocolConfig
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return err
		}
	}
	cfg.Normalize()
	c.Location = cfg.Location
	c.OccupiedStations = cfg.OccupiedStations
	c.GroupNo = cfg.GroupNo
	c.TimeoutMs = cfg.TimeoutMs
	return nil
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	cfg := ProtocolConfig{
		Location:         c.Location,
		OccupiedStations: c.OccupiedStations,
		GroupNo:          c.GroupNo,
		TimeoutMs:        c.TimeoutMs,
	}
	cfg.Normalize()
	return json.Marshal(cfg)
}

func (c *Collection) GetCollectionMeta() *collection.CollectionMeta { return &c.CollectionMeta }
func (c *Collection) SetCollectionMeta(meta *collection.CollectionMeta) {
	if meta != nil {
		c.CollectionMeta = *meta
	}
}
func (c *Collection) SetFrameTracer(tracer collection.FrameTracer) {}
