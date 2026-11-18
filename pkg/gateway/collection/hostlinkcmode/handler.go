package hostlinkcmode

import (
	"encoding/json"
	"errors"
	"fmt"
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
		return nil, nil, fmt.Errorf("invalid hostlink create request")
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
	issues := make([]validation.FieldIssue, 0, 3)
	if strings.TrimSpace(r.Name) == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.collection_name_required"})
	}
	if r.CollectorCycleMs == 0 {
		issues = append(issues, validation.FieldIssue{Field: "collectorCycleMs", MsgID: "validate.collection_cycle_invalid"})
	}
	if strings.TrimSpace(r.ProtocolConfig.Location) == "" {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.location", MsgID: "validate.collection_field_required", Args: []interface{}{"protocolConfig.location"}})
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
	_ collection.CollectorFactory = (*CollectorFactory)(nil)
	_ collection.CreateHandler    = (*CreateHandler)(nil)
)
