// Package profinet implements PROFINET collection for Siemens CPUs through
// S7 communication (ISO-on-TCP, port 102). PROFINET IO cyclic process data is
// owned by the PLC; the gateway reads the mapped process image via the CPU's
// S7 service, which is the supported access path for a software gateway.
package profinet

import (
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection/s7"
	"harns-platform/pkg/gateway/validation"
)

const (
	DriverType  = "profinet"
	DriverModel = "profinet"

	DefaultPort = s7.DefaultPort
)

type (
	UpdateReq      = s7.UpdateReq
	ProtocolConfig = s7.ProtocolConfig
	Collection     = s7.S7Collection
)

// CreateReq mirrors the S7 create request but validates PROFINET connection
// fields before the S7 transport is built.
type CreateReq s7.CreateReq

func (r *CreateReq) Validate() error {
	if r == nil {
		return validation.New([]validation.FieldIssue{{Field: "protocolConfig.location", MsgID: "validate.collection_field_required", Args: []interface{}{"protocolConfig.location"}}})
	}
	if err := (*s7.CreateReq)(r).Validate(); err != nil {
		return err
	}
	issues := make([]validation.FieldIssue, 0, 2)
	if strings.TrimSpace(r.ProtocolConfig.Location) == "" {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.location", MsgID: "validate.collection_field_required", Args: []interface{}{"protocolConfig.location"}})
	}
	if r.ProtocolConfig.Port != 0 && (r.ProtocolConfig.Port < 1 || r.ProtocolConfig.Port > 65535) {
		issues = append(issues, validation.FieldIssue{Field: "protocolConfig.port", MsgID: "validate.collection_field_invalid", Args: []interface{}{"protocolConfig.port"}})
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}

func NewCollection(req *CreateReq) (*s7.S7Collection, error) {
	if req == nil {
		return nil, fmt.Errorf("profinet create request is nil")
	}
	return s7.NewCollectionFor(DriverType, DriverModel, (*s7.CreateReq)(req))
}
