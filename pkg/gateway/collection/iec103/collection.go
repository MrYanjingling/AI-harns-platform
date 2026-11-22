// Package iec103 implements IEC 60870-5-103 collection.
package iec103

import "harns-platform/pkg/gateway/collection"

const (
	DriverType  = "IEC"
	DriverModel = "IEC103"
)

type ProtocolConfig struct {
	Location string `json:"location"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Link     int    `json:"link"`
	Interval int    `json:"interval"`
	Timeout  int    `json:"timeout"`
	Device   string `json:"device"`
	Baud     int    `json:"baud"`
	Parity   int    `json:"parity"`
	Data     int    `json:"data"`
	StopBits int    `json:"stop"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Link != 1 {
		c.Link = 0
	}
	if c.Timeout == 0 {
		c.Timeout = 3000
	}
	if c.Baud == 0 {
		c.Baud = 9600
	}
	if c.Data == 0 {
		c.Data = 8
	}
	if c.Location == "" {
		if c.Link == 1 {
			c.Location = c.Host
		} else {
			c.Location = c.Device
		}
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
	Link        int    `json:"link"`
	Interval    int    `json:"interval"`
	Timeout     int    `json:"timeout"`
	Device      string `json:"device"`
	Baud        int    `json:"baud"`
	Parity      int    `json:"parity"`
	Data        int    `json:"data"`
	StopBits    int    `json:"stop"`
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
			CollectStatus: collection.CollectStatusStopped, CollectorCycleMs: req.CollectorCycleMs,
			Variables: req.Variables,
		},
		Location: req.ProtocolConfig.Location, Port: req.ProtocolConfig.Port, Link: req.ProtocolConfig.Link,
		Interval: req.ProtocolConfig.Interval, Timeout: req.ProtocolConfig.Timeout,
		Device: req.ProtocolConfig.Device, Baud: req.ProtocolConfig.Baud, Parity: req.ProtocolConfig.Parity,
		Data: req.ProtocolConfig.Data, StopBits: req.ProtocolConfig.StopBits,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location, c.Port, c.Link = req.ProtocolConfig.Location, req.ProtocolConfig.Port, req.ProtocolConfig.Link
	c.Interval, c.Timeout = req.ProtocolConfig.Interval, req.ProtocolConfig.Timeout
	c.Device, c.Baud, c.Parity = req.ProtocolConfig.Device, req.ProtocolConfig.Baud, req.ProtocolConfig.Parity
	c.Data, c.StopBits = req.ProtocolConfig.Data, req.ProtocolConfig.StopBits
}

var _ collection.Collector = (*Collection)(nil)
