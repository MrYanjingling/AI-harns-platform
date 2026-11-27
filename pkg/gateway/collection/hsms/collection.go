package hsms

import (
	"encoding/json"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "SECS/GEM"
	DriverModel = "HSMS"

	DefaultPort               = 5000
	DefaultDeviceID           = 0
	DefaultTimeoutMs          = 5000
	DefaultLinktestIntervalMs = 30000
	DefaultEstablishComm      = 1
)

type ProtocolConfig struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	DeviceID           int    `json:"deviceId"`
	EstablishComm      *int   `json:"establishComm,omitempty"` // 1=send S1F13 after Select (default), 0=skip
	TimeoutMs          int    `json:"timeoutMs"`               // T3/T6 reply timeout (ms)
	LinktestIntervalMs int    `json:"linktestIntervalMs"`      // 0=disabled, default 30000
}

func (c *ProtocolConfig) Normalize() {
	c.Host = strings.TrimSpace(c.Host)
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = DefaultPort
	}
	if c.DeviceID < 0 || c.DeviceID > 65535 {
		c.DeviceID = DefaultDeviceID
	}
	if c.EstablishComm == nil {
		v := DefaultEstablishComm
		c.EstablishComm = &v
	} else if *c.EstablishComm != 0 && *c.EstablishComm != 1 {
		v := DefaultEstablishComm
		c.EstablishComm = &v
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.LinktestIntervalMs < 0 {
		c.LinktestIntervalMs = DefaultLinktestIntervalMs
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

	Host               string `json:"host"`
	Port               int    `json:"port"`
	DeviceID           int    `json:"deviceId"`
	EstablishComm      int    `json:"establishComm"`
	TimeoutMs          int    `json:"timeoutMs"`
	LinktestIntervalMs int    `json:"linktestIntervalMs"`

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
	}
	item.applyNormalized(cfg)
	return item, nil
}

func (c *Collection) applyNormalized(cfg ProtocolConfig) {
	c.Host = cfg.Host
	c.Port = cfg.Port
	c.DeviceID = cfg.DeviceID
	if cfg.EstablishComm != nil {
		c.EstablishComm = *cfg.EstablishComm
	} else {
		c.EstablishComm = DefaultEstablishComm
	}
	c.TimeoutMs = cfg.TimeoutMs
	c.LinktestIntervalMs = cfg.LinktestIntervalMs
}

func (c *Collection) ApplyConfig(raw json.RawMessage) error {
	var cfg ProtocolConfig
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return err
		}
	}
	cfg.Normalize()
	c.applyNormalized(cfg)
	return nil
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	ec := c.EstablishComm
	cfg := ProtocolConfig{
		Host:               c.Host,
		Port:               c.Port,
		DeviceID:           c.DeviceID,
		EstablishComm:      &ec,
		TimeoutMs:          c.TimeoutMs,
		LinktestIntervalMs: c.LinktestIntervalMs,
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
