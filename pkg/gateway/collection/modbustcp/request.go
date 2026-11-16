package modbustcp

import (
	"errors"
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbus"
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
	for index, err := range modbus.ValidateVariablePositions(r.Variables, r.ProtocolConfig.PositionAddress) {
		if err == nil {
			continue
		}
		issues = append(issues, validation.FieldIssue{
			Field: fmt.Sprintf("variables[%d].address", index),
			MsgID: "validate.variable_address_invalid",
			Args:  []interface{}{r.Variables[index].Address},
		})
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}
