package iec61850

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type CollectorFactory struct{}

func (f *CollectorFactory) DriverType() string  { return DriverType }
func (f *CollectorFactory) DriverModel() string { return DriverModel }
func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &Collection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

type CreateHandler struct{}

func (h *CreateHandler) DriverType() string  { return DriverType }
func (h *CreateHandler) DriverModel() string { return DriverModel }
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
		return nil, nil, fmt.Errorf("invalid IEC 61850 create request")
	}
	item, err := NewCollection(req)
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
	if strings.TrimSpace(r.ProtocolConfig.Host) == "" || net.ParseIP(strings.TrimSpace(r.ProtocolConfig.Host)) == nil {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.host", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.host"}})
	}
	if r.ProtocolConfig.Port < 0 || r.ProtocolConfig.Port > 65535 {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.port", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.port"}})
	}
	if strings.TrimSpace(r.ProtocolConfig.DefaultFC) != "" && len(strings.TrimSpace(r.ProtocolConfig.DefaultFC)) != 2 {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.defaultFC", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.defaultFC"}})
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
		if _, err := ParseAddress(variable.Address, r.ProtocolConfig.DefaultFC); err != nil {
			issues = append(issues, validation.FieldIssue{Field: fmt.Sprintf("variables[%d].address", index), MsgID: "validate.variable_address_invalid", Args: []interface{}{variable.Address}})
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}

var _ collection.CreateHandler = (*CreateHandler)(nil)
