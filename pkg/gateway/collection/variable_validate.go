package collection

import (
	"fmt"
	"regexp"
	"strings"

	"harns-platform/pkg/gateway/validation"
)

var variableNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// NormalizeVariables trims variable fields and applies defaults.
func NormalizeVariables(variables []*Variable) error {
	for _, variable := range variables {
		if variable == nil {
			continue
		}
		variable.Name = strings.TrimSpace(variable.Name)
		variable.Type = strings.TrimSpace(variable.Type)
		variable.DataType = strings.TrimSpace(variable.DataType)
		variable.Address = strings.TrimSpace(variable.Address)
		variable.Expression = strings.TrimSpace(variable.Expression)
		if variable.Rate == 0 {
			variable.Rate = 1
		}
	}
	return ValidateVariables(variables)
}

// ValidateVariables validates names and calculated/normal point rules.
func ValidateVariables(variables []*Variable) error {
	seen := make(map[string]struct{}, len(variables))
	issues := make([]validation.FieldIssue, 0)
	for i, variable := range variables {
		if variable == nil {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d]", i),
				MsgID: "validate.variable_null",
			})
			continue
		}
		name := strings.TrimSpace(variable.Name)
		if name == "" {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d].name", i),
				MsgID: "validate.variable_name_required",
			})
		} else {
			if !variableNamePattern.MatchString(name) {
				issues = append(issues, validation.FieldIssue{
					Field: fmt.Sprintf("variables[%d].name", i),
					MsgID: "validate.variable_name_invalid",
					Args:  []interface{}{name},
				})
			}
			if _, ok := seen[name]; ok {
				issues = append(issues, validation.FieldIssue{
					Field: fmt.Sprintf("variables[%d].name", i),
					MsgID: "validate.variable_name_duplicate",
					Args:  []interface{}{name},
				})
			}
			seen[name] = struct{}{}
		}
		if strings.TrimSpace(variable.DataType) == "" {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d].dataType", i),
				MsgID: "validate.variable_data_type_required",
			})
		}
		if !ValidAccessMode(variable.AccessMode) {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d].accessMode", i),
				MsgID: "validate.variable_access_mode_invalid",
			})
		}
		if variable.Precision < 0 {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d].precision", i),
				MsgID: "validate.variable_precision_invalid",
			})
		}
		if variable.IsCalculated {
			if strings.TrimSpace(variable.Address) != "" {
				issues = append(issues, validation.FieldIssue{
					Field: fmt.Sprintf("variables[%d].address", i),
					MsgID: "validate.variable_calculated_address_empty",
				})
			}
			if strings.TrimSpace(variable.Expression) == "" {
				issues = append(issues, validation.FieldIssue{
					Field: fmt.Sprintf("variables[%d].expression", i),
					MsgID: "validate.variable_expression_required",
				})
			}
			if variable.AccessMode != AccessModeRead {
				issues = append(issues, validation.FieldIssue{
					Field: fmt.Sprintf("variables[%d].accessMode", i),
					MsgID: "validate.variable_calculated_read_only",
				})
			}
			continue
		}
		if strings.TrimSpace(variable.Address) == "" {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d].address", i),
				MsgID: "validate.variable_address_required",
			})
		}
		if strings.TrimSpace(variable.Expression) != "" {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("variables[%d].expression", i),
				MsgID: "validate.variable_expression_empty",
			})
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}
