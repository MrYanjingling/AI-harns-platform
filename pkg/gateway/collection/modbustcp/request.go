package modbustcp

import (
	"errors"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

// Validate checks the Modbus TCP create request.
func (r *CreateReq) Validate() error {
	issues := make([]validation.FieldIssue, 0, 3)
	if strings.TrimSpace(r.Name) == "" {
		issues = append(issues, validation.FieldIssue{
			Field: "name",
			MsgID: "validate.collection_name_required",
		})
	}
	if r.CollectorCycleMs == 0 {
		issues = append(issues, validation.FieldIssue{
			Field: "collectorCycleMs",
			MsgID: "validate.collection_cycle_invalid",
		})
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
