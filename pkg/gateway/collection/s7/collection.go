// Package s7 implements Siemens S7-1500 collection.
package s7

import (
	"encoding/json"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "s7"
	DriverModel = "s71500"

	DefaultPort           = 102
	DefaultTCPCount       = 1
	DefaultPDUSize        = 480
	DefaultConnectionType = 1
	MaxTCPCount           = 3
)

// FlexibleInt accepts both JSON numbers and numeric strings.
type FlexibleInt int

func (n *FlexibleInt) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*n = 0
		return nil
	}
	if strings.HasPrefix(raw, `"`) {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		raw = strings.TrimSpace(text)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return err
	}
	*n = FlexibleInt(value)
	return nil
}

func (n FlexibleInt) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(n))), nil
}

type ProtocolConfig struct {
	Location       string      `json:"location"`
	Port           int         `json:"port"`
	Rack           uint8       `json:"rack"`
	Slot           uint8       `json:"slot"`
	PDUSize        FlexibleInt `json:"pduSize"`
	ConnectionType FlexibleInt `json:"connectionType"`
	TCPCount       int         `json:"tcpCount"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.PDUSize == 0 {
		c.PDUSize = DefaultPDUSize
	}
	if c.ConnectionType == 0 {
		c.ConnectionType = DefaultConnectionType
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
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

type S7Collection struct {
	collection.CollectionMeta

	Location       string `json:"location"`
	Port           int    `json:"port"`
	Rack           uint8  `json:"rack"`
	Slot           uint8  `json:"slot"`
	PDUSize        int    `json:"pduSize"`
	ConnectionType uint8  `json:"connectionType"`
	TCPCount       int    `json:"tcpCount"`

	runtime     *collectorRuntime
	frameTracer collection.FrameTracer
}

func NewCollection(req *CreateReq) (*S7Collection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &S7Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       DriverType,
			DriverModel:      DriverModel,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location:       req.ProtocolConfig.Location,
		Port:           req.ProtocolConfig.Port,
		Rack:           req.ProtocolConfig.Rack,
		Slot:           req.ProtocolConfig.Slot,
		PDUSize:        int(req.ProtocolConfig.PDUSize),
		ConnectionType: uint8(req.ProtocolConfig.ConnectionType),
		TCPCount:       req.ProtocolConfig.TCPCount,
	}, nil
}

func (c *S7Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Rack = req.ProtocolConfig.Rack
	c.Slot = req.ProtocolConfig.Slot
	c.PDUSize = int(req.ProtocolConfig.PDUSize)
	c.ConnectionType = uint8(req.ProtocolConfig.ConnectionType)
	c.TCPCount = req.ProtocolConfig.TCPCount
}

var _ collection.Collector = (*S7Collection)(nil)
