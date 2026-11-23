package cjt188

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type CollectorFactory struct{ model string }

func NewCollectorFactory(model string) *CollectorFactory { return &CollectorFactory{model: model} }
func (f *CollectorFactory) DriverType() string           { return DriverType }
func (f *CollectorFactory) DriverModel() string          { return f.model }
func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &Collection{Version: f.model}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

type CreateHandler struct{ model string }

func NewCreateHandler(model string) *CreateHandler { return &CreateHandler{model: model} }
func (h *CreateHandler) DriverType() string        { return DriverType }
func (h *CreateHandler) DriverModel() string       { return h.model }
func (h *CreateHandler) DecodeCreate(body []byte) (collection.CreateRequest, error) {
	var req CreateReq
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	req.model = h.model
	return &req, nil
}
func (h *CreateHandler) BuildCreate(raw collection.CreateRequest) (collection.Collector, json.RawMessage, error) {
	req, ok := raw.(*CreateReq)
	if !ok {
		return nil, nil, fmt.Errorf("invalid cjt188 create request")
	}
	item, err := NewCollection(h.model, req)
	if err != nil {
		return nil, nil, err
	}
	config, err := item.ConfigJSON()
	if err != nil {
		return nil, nil, err
	}
	return item, json.RawMessage(config), nil
}

func (r *CreateReq) Validate() error {
	issues := make([]validation.FieldIssue, 0, 6)
	if strings.TrimSpace(r.Name) == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.collection_name_required"})
	}
	if r.CollectorCycleMs == 0 {
		issues = append(issues, validation.FieldIssue{Field: "collectorCycleMs", MsgID: "validate.collection_cycle_invalid"})
	}
	if r.ProtocolConfig.Link != 0 && r.ProtocolConfig.Link != 1 {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.link", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.link"}})
	}
	if strings.TrimSpace(r.ProtocolConfig.Location) == "" {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.location", MsgID: "validate.collection_field_required", Args: []interface{}{"protocolConfig.location"}})
	}
	if r.ProtocolConfig.Port < 0 || r.ProtocolConfig.Port > 65535 || (r.ProtocolConfig.Link == 1 && r.ProtocolConfig.Port < 1) {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.port", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.port"}})
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
		if _, err := parseAddress(variable.Address); err != nil {
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

var _ collection.CreateHandler = (*CreateHandler)(nil)
