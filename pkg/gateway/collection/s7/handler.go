package s7

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type CreateHandler struct {
	driverType string
	model      string
}

func NewS7200CreateHandler() *CreateHandler {
	return &CreateHandler{driverType: DriverType, model: DriverModelS7200}
}

// NewCreateHandlerFor builds a create handler under an alternate driver identity.
func NewCreateHandlerFor(driverType, model string) *CreateHandler {
	return &CreateHandler{driverType: driverType, model: model}
}

func (h *CreateHandler) DriverType() string {
	if h == nil || h.driverType == "" {
		return DriverType
	}
	return h.driverType
}
func (h *CreateHandler) DriverModel() string {
	if h == nil || h.model == "" {
		return DriverModel
	}
	return h.model
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
		return nil, nil, fmt.Errorf("invalid s7 create request")
	}
	item, err := newCollectionWithDriver(h.DriverType(), h.DriverModel(), req)
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
	if err := collection.NormalizeVariables(r.Variables); err != nil {
		var validationErr *validation.ValidationError
		if errors.As(err, &validationErr) {
			issues = append(issues, validationErr.Issues...)
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}

var _ collection.CreateHandler = (*CreateHandler)(nil)
