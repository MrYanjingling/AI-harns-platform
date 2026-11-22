// Package iec101 implements IEC 60870-5-101 serial/TCP collection.
package iec101

import "harns-platform/pkg/gateway/collection"

const (
	DriverType  = "IEC"
	DriverModel = "IEC101"
)

type ProtocolConfig struct {
	Location  string `json:"location"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Link      int    `json:"link"`
	CA        int    `json:"ca"`
	CALength  int    `json:"caLength"`
	COTLength int    `json:"cotLength"`
	IOALength int    `json:"ioaLength"`
	Interval  int    `json:"interval"`
	Timeout   int    `json:"timeout"`
	Device    string `json:"device"`
	Baud      int    `json:"baud"`
	Parity    int    `json:"parity"`
	Data      int    `json:"data"`
	StopBits  int    `json:"stop"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Link != 1 {
		c.Link = 0
	}
	if c.CA == 0 {
		c.CA = 1
	}
	if c.CALength == 0 {
		c.CALength = 1
	}
	if c.COTLength == 0 {
		c.COTLength = 1
	}
	if c.IOALength == 0 {
		c.IOALength = 3
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

	Location  string `json:"location"`
	Port      int    `json:"port"`
	Link      int    `json:"link"`
	CA        int    `json:"ca"`
	CALength  int    `json:"caLength"`
	COTLength int    `json:"cotLength"`
	IOALength int    `json:"ioaLength"`
	Interval  int    `json:"interval"`
	Timeout   int    `json:"timeout"`
	Device    string `json:"device"`
	Baud      int    `json:"baud"`
	Parity    int    `json:"parity"`
	Data      int    `json:"data"`
	StopBits  int    `json:"stop"`

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
		Location: req.ProtocolConfig.Location, Port: req.ProtocolConfig.Port,
		Link: req.ProtocolConfig.Link, CA: req.ProtocolConfig.CA,
		CALength: req.ProtocolConfig.CALength, COTLength: req.ProtocolConfig.COTLength,
		IOALength: req.ProtocolConfig.IOALength, Interval: req.ProtocolConfig.Interval,
		Timeout: req.ProtocolConfig.Timeout, Device: req.ProtocolConfig.Device,
		Baud: req.ProtocolConfig.Baud, Parity: req.ProtocolConfig.Parity,
		Data: req.ProtocolConfig.Data, StopBits: req.ProtocolConfig.StopBits,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Link = req.ProtocolConfig.Link
	c.CA = req.ProtocolConfig.CA
	c.CALength = req.ProtocolConfig.CALength
	c.COTLength = req.ProtocolConfig.COTLength
	c.IOALength = req.ProtocolConfig.IOALength
	c.Interval = req.ProtocolConfig.Interval
	c.Timeout = req.ProtocolConfig.Timeout
	c.Device = req.ProtocolConfig.Device
	c.Baud = req.ProtocolConfig.Baud
	c.Parity = req.ProtocolConfig.Parity
	c.Data = req.ProtocolConfig.Data
	c.StopBits = req.ProtocolConfig.StopBits
}

var _ collection.Collector = (*Collection)(nil)
