// Package s7 implements Siemens S7-1500 and S7-200 collections.
package s7

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType       = "s7"
	DriverModel      = "s71500"
	DriverModelS7200 = "s7-200"
	DriverS7200      = "s7-200"

	DefaultPort           = 102
	DefaultConcurrency    = 1
	DefaultPDUSize        = 480
	DefaultConnectionType = 1
	DefaultLocalTSAP      = 0x1000
	DefaultRemoteTSAP     = 0x1000
	MaxConcurrency        = 3
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

// FlexibleTSAP accepts a hexadecimal TSAP string such as 0x1000.
type FlexibleTSAP uint16

func (n *FlexibleTSAP) UnmarshalJSON(data []byte) error {
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
	if raw == "" {
		*n = 0
		return nil
	}
	if !strings.HasPrefix(strings.ToLower(raw), "0x") {
		return fmt.Errorf("tsap must be a hexadecimal string like 0x1000")
	}
	value, err := strconv.ParseUint(raw[2:], 16, 16)
	if err != nil {
		return fmt.Errorf("invalid tsap %q", raw)
	}
	*n = FlexibleTSAP(value)
	return nil
}

func (n FlexibleTSAP) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", fmt.Sprintf("0x%04X", uint16(n)))), nil
}

type ProtocolConfig struct {
	Location       string       `json:"location"`
	Port           int          `json:"port"`
	Rack           uint8        `json:"rack"`
	Slot           uint8        `json:"slot"`
	PDUSize        FlexibleInt  `json:"pduSize"`
	ConnectionType FlexibleInt  `json:"connectionType"`
	Concurrency    int          `json:"concurrency"`
	LocalTSAP      FlexibleTSAP `json:"localTsap,omitempty"`
	RemoteTSAP     FlexibleTSAP `json:"remoteTsap,omitempty"`
}

func (c *ProtocolConfig) Normalize(model string) {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.PDUSize == 0 {
		c.PDUSize = DefaultPDUSize
	}
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConcurrency
	}
	if model == DriverModelS7200 {
		if c.LocalTSAP == 0 {
			c.LocalTSAP = DefaultLocalTSAP
		}
		if c.RemoteTSAP == 0 {
			c.RemoteTSAP = DefaultRemoteTSAP
		}
		return
	}
	if c.ConnectionType == 0 {
		c.ConnectionType = DefaultConnectionType
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
	Concurrency    int    `json:"concurrency"`
	LocalTSAP      uint16 `json:"localTsap"`
	RemoteTSAP     uint16 `json:"remoteTsap"`

	runtime     *collectorRuntime
	frameTracer collection.FrameTracer
}

func NewCollection(req *CreateReq) (*S7Collection, error) {
	return newCollectionWithDriver(DriverType, DriverModel, req)
}

func NewS7200Collection(req *CreateReq) (*S7Collection, error) {
	return newCollectionWithDriver(DriverType, DriverModelS7200, req)
}

// NewCollectionFor builds an S7 collection under an alternate driver identity.
// It lets protocol drivers that reuse S7 communication (for example PROFINET)
// keep their own driverType/driverModel while sharing the S7 transport.
func NewCollectionFor(driverType, model string, req *CreateReq) (*S7Collection, error) {
	return newCollectionWithDriver(driverType, model, req)
}

func newCollectionWithDriver(driverType, model string, req *CreateReq) (*S7Collection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize(model)
	return &S7Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       driverType,
			DriverModel:      model,
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
		Concurrency:    req.ProtocolConfig.Concurrency,
		LocalTSAP:      uint16(req.ProtocolConfig.LocalTSAP),
		RemoteTSAP:     uint16(req.ProtocolConfig.RemoteTSAP),
	}, nil
}

func (c *S7Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize(c.GetDriverModel())
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Rack = req.ProtocolConfig.Rack
	c.Slot = req.ProtocolConfig.Slot
	c.PDUSize = int(req.ProtocolConfig.PDUSize)
	c.ConnectionType = uint8(req.ProtocolConfig.ConnectionType)
	c.Concurrency = req.ProtocolConfig.Concurrency
	c.LocalTSAP = uint16(req.ProtocolConfig.LocalTSAP)
	c.RemoteTSAP = uint16(req.ProtocolConfig.RemoteTSAP)
}

var _ collection.Collector = (*S7Collection)(nil)
