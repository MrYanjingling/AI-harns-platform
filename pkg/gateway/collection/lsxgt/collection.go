// Package lsxgt implements LS Electric XGT FEnet (TCP binary) and XGT Cnet (ASCII) protocols.
package lsxgt

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

const (
	DriverType = "LS Electric"
	ModelFEnet = "XGT FEnet"
	ModelCnet  = "XGT Cnet"

	DefaultFEnetPort = 2004
	DefaultCnetPort  = 2005
	DefaultTimeoutMs = 3000
)

type ProtocolConfig struct {
	Location  string `json:"location"`
	Station   int    `json:"station,omitempty"`
	TimeoutMs int    `json:"timeoutMs,omitempty"`
}

func (c *ProtocolConfig) Normalize() {
	c.Location = strings.TrimSpace(c.Location)
	if c.Station < 0 || c.Station > 31 {
		c.Station = 0
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
}

func (c *ProtocolConfig) endpoint(defaultPort int) string {
	loc := strings.TrimSpace(c.Location)
	if loc == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(loc); err == nil {
		return loc
	}
	return net.JoinHostPort(loc, strconv.Itoa(defaultPort))
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
	Station   int    `json:"station,omitempty"`
	TimeoutMs int    `json:"timeoutMs"`

	cnet        bool
	runtime     *collectorRuntime
	frameTracer collection.FrameTracer
}

func NewCollection(model string, req *CreateReq) (*Collection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	if model == "" {
		model = ModelFEnet
	}
	req.ProtocolConfig.Normalize()
	return &Collection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       DriverType,
			DriverModel:      model,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location:  req.ProtocolConfig.Location,
		Station:   req.ProtocolConfig.Station,
		TimeoutMs: req.ProtocolConfig.TimeoutMs,
		cnet:      model == ModelCnet,
	}, nil
}

func (c *Collection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Station = req.ProtocolConfig.Station
	c.TimeoutMs = req.ProtocolConfig.TimeoutMs
}

func (c *Collection) ConfigJSON() ([]byte, error) {
	cfg := ProtocolConfig{Location: c.Location, Station: c.Station, TimeoutMs: c.TimeoutMs}
	cfg.Normalize()
	return json.Marshal(cfg)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var cfg ProtocolConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	cfg.Normalize()
	c.Location = cfg.Location
	c.Station = cfg.Station
	c.TimeoutMs = cfg.TimeoutMs
	return nil
}

func (c *Collection) SetFrameTracer(tracer collection.FrameTracer) {
	c.frameTracer = tracer
}

type CollectorFactory struct {
	model string
}

func NewCollectorFactory(model string) *CollectorFactory {
	return &CollectorFactory{model: model}
}

func (f *CollectorFactory) DriverType() string  { return DriverType }
func (f *CollectorFactory) DriverModel() string { return f.model }

func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &Collection{cnet: f.model == ModelCnet}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

type CreateHandler struct {
	model string
}

func NewCreateHandler(model string) *CreateHandler {
	return &CreateHandler{model: model}
}

func (h *CreateHandler) DriverType() string  { return DriverType }
func (h *CreateHandler) DriverModel() string { return h.model }

func (h *CreateHandler) DecodeCreate(body []byte) (collection.CreateRequest, error) {
	var req CreateReq
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (h *CreateHandler) BuildCreate(raw collection.CreateRequest) (collection.Collector, json.RawMessage, error) {
	req, ok := raw.(*CreateReq)
	if !ok {
		return nil, nil, fmt.Errorf("invalid %s create request", h.model)
	}
	item, err := NewCollection(h.model, req)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := item.ConfigJSON()
	if err != nil {
		return nil, nil, err
	}
	return item, json.RawMessage(cfg), nil
}

func (r *CreateReq) Validate() error {
	issues := make([]validation.FieldIssue, 0, 3)
	if strings.TrimSpace(r.Name) == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.collection_name_required"})
	}
	if r.CollectorCycleMs == 0 {
		issues = append(issues, validation.FieldIssue{Field: "collectorCycleMs", MsgID: "validate.collection_cycle_invalid"})
	}
	if strings.TrimSpace(r.ProtocolConfig.Location) == "" {
		issues = append(issues, validation.FieldIssue{
			Field: "protocolConfig.location",
			MsgID: "validate.collection_field_required",
			Args:  []interface{}{"protocolConfig.location"},
		})
	}
	if err := collection.NormalizeVariables(r.Variables); err != nil {
		var validationErr *validation.ValidationError
		if errors.As(err, &validationErr) {
			issues = append(issues, validationErr.Issues...)
		}
	}
	for index, variable := range r.Variables {
		if variable == nil || variable.IsCalculated {
			continue
		}
		if _, err := parseAddress(variable.Address, variable.DataType); err != nil {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d].address", index),
				MsgID: "validate.variable_address_invalid",
				Args:  []interface{}{variable.Address},
			})
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}

var (
	_ collection.Collector        = (*Collection)(nil)
	_ collection.ActionHandler    = (*Collection)(nil)
	_ collection.CollectorFactory = (*CollectorFactory)(nil)
	_ collection.CreateHandler    = (*CreateHandler)(nil)
)
