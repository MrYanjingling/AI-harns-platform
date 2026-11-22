// Package iec104 implements IEC 60870-5-104 TCP collection.
package iec104

import "harns-platform/pkg/gateway/collection"

const (
	DriverType  = "IEC"
	DriverModel = "IEC104"
)

type ProtocolConfig struct {
	Location            string `json:"location"`
	Port                int    `json:"port"`
	CA                  int    `json:"ca"`
	Interval            int    `json:"interval"`
	SyncTimeInterval    int    `json:"syncTimeInterval"`
	RemotePulseInterval int    `json:"remotePulseInterval"`
	T0                  int    `json:"t0"`
	T1                  int    `json:"t1"`
	T2                  int    `json:"t2"`
	T3                  int    `json:"t3"`
	W                   int    `json:"w"`
	Timeout             int    `json:"timeout"`
	Active              bool   `json:"active"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = 2404
	}
	if c.CA == 0 {
		c.CA = 1
	}
	if c.Interval == 0 {
		c.Interval = 60
	}
	if c.T0 == 0 {
		c.T0 = 30
	}
	if c.T1 == 0 {
		c.T1 = 15
	}
	if c.T2 == 0 {
		c.T2 = 10
	}
	if c.T3 == 0 {
		c.T3 = 20
	}
	if c.W == 0 {
		c.W = 8
	}
	if c.Timeout == 0 {
		c.Timeout = 5
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

	Location            string `json:"location"`
	Port                int    `json:"port"`
	CA                  int    `json:"ca"`
	Interval            int    `json:"interval"`
	SyncTimeInterval    int    `json:"syncTimeInterval"`
	RemotePulseInterval int    `json:"remotePulseInterval"`
	T0                  int    `json:"t0"`
	T1                  int    `json:"t1"`
	T2                  int    `json:"t2"`
	T3                  int    `json:"t3"`
	W                   int    `json:"w"`
	Timeout             int    `json:"timeout"`
	Active              bool   `json:"active"`

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
		CA: req.ProtocolConfig.CA, Interval: req.ProtocolConfig.Interval,
		SyncTimeInterval:    req.ProtocolConfig.SyncTimeInterval,
		RemotePulseInterval: req.ProtocolConfig.RemotePulseInterval,
		T0:                  req.ProtocolConfig.T0, T1: req.ProtocolConfig.T1, T2: req.ProtocolConfig.T2, T3: req.ProtocolConfig.T3,
		W: req.ProtocolConfig.W, Timeout: req.ProtocolConfig.Timeout, Active: req.ProtocolConfig.Active,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.CA = req.ProtocolConfig.CA
	c.Interval = req.ProtocolConfig.Interval
	c.SyncTimeInterval = req.ProtocolConfig.SyncTimeInterval
	c.RemotePulseInterval = req.ProtocolConfig.RemotePulseInterval
	c.T0 = req.ProtocolConfig.T0
	c.T1 = req.ProtocolConfig.T1
	c.T2 = req.ProtocolConfig.T2
	c.T3 = req.ProtocolConfig.T3
	c.W = req.ProtocolConfig.W
	c.Timeout = req.ProtocolConfig.Timeout
	c.Active = req.ProtocolConfig.Active
}

var _ collection.Collector = (*Collection)(nil)
