package ethercat

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
		return nil, nil, fmt.Errorf("invalid ethercat create request")
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
	issues := make([]validation.FieldIssue, 0, 4)
	if strings.TrimSpace(r.Name) == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.collection_name_required"})
	}
	if r.CollectorCycleMs == 0 {
		issues = append(issues, validation.FieldIssue{Field: "collectorCycleMs", MsgID: "validate.collection_cycle_invalid"})
	}
	if strings.TrimSpace(r.ProtocolConfig.Location) == "" {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.location", MsgID: "validate.collection_field_required", Args: []interface{}{"protocolConfig.location"}})
	}
	if r.ProtocolConfig.Port != 0 && (r.ProtocolConfig.Port < 1 || r.ProtocolConfig.Port > 65535) {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.port", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.port"}})
	}
	if r.ProtocolConfig.AMSPort != 0 && (r.ProtocolConfig.AMSPort < 1 || r.ProtocolConfig.AMSPort > 65535) {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.amsPort", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.amsPort"}})
	}
	if r.ProtocolConfig.LocalAMSPort != 0 && (r.ProtocolConfig.LocalAMSPort < 1 || r.ProtocolConfig.LocalAMSPort > 65535) {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.localAmsPort", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.localAmsPort"}})
	}
	if r.ProtocolConfig.AMSNetID != "" {
		if _, err := parseAMSNetID(r.ProtocolConfig.AMSNetID); err != nil {
			issues = append(issues, validation.FieldIssue{Field: "protocolConfig.amsNetId", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.amsNetId"}})
		}
	}
	if r.ProtocolConfig.LocalAMSNetID != "" {
		if _, err := parseAMSNetID(r.ProtocolConfig.LocalAMSNetID); err != nil {
			issues = append(issues, validation.FieldIssue{Field: "protocolConfig.localAmsNetId", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.localAmsNetId"}})
		}
	}
	if r.ProtocolConfig.StringLength < 0 || r.ProtocolConfig.StringLength > 65000 {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.stringLength", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.stringLength"}})
	}
	if r.ProtocolConfig.TimeoutMs < 0 {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.timeoutMs", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.timeoutMs"}})
	}
	if err := collection.NormalizeVariables(r.Variables); err != nil {
		var validationErr *validation.ValidationError
		if errors.As(err, &validationErr) {
			issues = append(issues, validationErr.Issues...)
		}
	}
	stringLength := r.ProtocolConfig.StringLength
	if stringLength <= 0 {
		stringLength = DefaultStringLength
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
			continue
		}
		if _, err := typeSize(variable.DataType, stringLength); err != nil {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d].dataType", index),
				MsgID: "validate.variable_data_type_invalid",
				Args:  []interface{}{variable.DataType},
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
