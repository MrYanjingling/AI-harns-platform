package fins

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type CollectorFactory struct {
	model string
}

func NewCollectorFactoryForModel(model string) *CollectorFactory {
	return &CollectorFactory{model: model}
}

func (f *CollectorFactory) DriverType() string { return DriverType }
func (f *CollectorFactory) DriverModel() string {
	if f != nil && f.model != "" {
		return f.model
	}
	return ModelFins
}
func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &Collection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	if f != nil && f.model == ModelHostlinkFins && item.Transport != TransportHostlink {
		item.Transport = TransportHostlink
	}
	return item, nil
}

type CreateHandler struct {
	model string
}

func NewCreateHandlerForModel(model string) *CreateHandler {
	return &CreateHandler{model: model}
}

func (h *CreateHandler) DriverType() string { return DriverType }
func (h *CreateHandler) DriverModel() string {
	if h != nil && h.model != "" {
		return h.model
	}
	return ModelFins
}
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
		return nil, nil, fmt.Errorf("invalid fins create request")
	}
	item, err := NewCollectionForModel(h.DriverModel(), req)
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
	if r.ProtocolConfig.Transport != "" && r.ProtocolConfig.Transport != TransportTCP && r.ProtocolConfig.Transport != TransportUDP && r.ProtocolConfig.Transport != TransportHostlink {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.transport", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.transport"}})
	}
	if r.ProtocolConfig.MaxSize != 0 && (r.ProtocolConfig.MaxSize < MinMaxSize || r.ProtocolConfig.MaxSize > MaxMaxSize) {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.maxSize", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.maxSize"}})
	}
	if r.ProtocolConfig.Concurrency != 0 && (r.ProtocolConfig.Concurrency < 1 || r.ProtocolConfig.Concurrency > MaxConcurrency) {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.concurrency", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.concurrency"}})
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
		if _, err := parseAddress(variable.Address, variable.DataType, variable.BitsValue()); err != nil {
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
