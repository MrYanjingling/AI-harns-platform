package ads

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
		return nil, nil, fmt.Errorf("invalid Beckhoff ADS create request")
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
	issues := make([]validation.FieldIssue, 0, 8)
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
	if r.ProtocolConfig.SrcADSPort < 0 || r.ProtocolConfig.SrcADSPort > 65535 || r.ProtocolConfig.DstADSPort < 0 || r.ProtocolConfig.DstADSPort > 65535 {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.dstAdsPort", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.dstAdsPort"}})
	}
	if r.ProtocolConfig.SrcAMSNetID != "" {
		if _, err := parseAMSNetID(r.ProtocolConfig.SrcAMSNetID); err != nil {
			issues = append(issues, validation.FieldIssue{Field: "protocolConfig.srcAmsNetId", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.srcAmsNetId"}})
		}
	}
	if r.ProtocolConfig.DstAMSNetID != "" {
		if _, err := parseAMSNetID(r.ProtocolConfig.DstAMSNetID); err != nil {
			issues = append(issues, validation.FieldIssue{Field: "protocolConfig.dstAmsNetId", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.dstAmsNetId"}})
		}
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
		address, err := parseAddress(variable.Address, variable.DataType)
		if err != nil {
			issues = append(issues, validation.FieldIssue{Field: fmt.Sprintf("variables[%d].address", index), MsgID: "validate.variable_address_invalid", Args: []interface{}{variable.Address}})
			continue
		}
		if address.ReadOnly && variable.AccessMode != collection.AccessModeRead {
			issues = append(issues, validation.FieldIssue{Field: fmt.Sprintf("variables[%d].accessMode", index), MsgID: "validate.variable_address_read_only", Args: []interface{}{variable.Address}})
		}
		if address.WriteOnly && variable.AccessMode != collection.AccessModeWrite {
			issues = append(issues, validation.FieldIssue{Field: fmt.Sprintf("variables[%d].accessMode", index), MsgID: "validate.variable_address_write_only", Args: []interface{}{variable.Address}})
		}
		if _, err := typeSize(variable.DataType, address); err != nil {
			issues = append(issues, validation.FieldIssue{Field: fmt.Sprintf("variables[%d].dataType", index), MsgID: "validate.variable_data_type_unsupported", Args: []interface{}{variable.DataType}})
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}

var _ collection.CreateHandler = (*CreateHandler)(nil)
