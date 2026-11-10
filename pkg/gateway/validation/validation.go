// Package validation provides shared field-level validation errors for Gateway.
package validation

import (
	"fmt"
	"strings"
)

// FieldIssue is one localized field validation failure.
type FieldIssue struct {
	Field string
	MsgID string
	Args  []interface{}
}

// ValidationError carries one or more field-level validation failures.
type ValidationError struct {
	Issues []FieldIssue
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Field, issue.MsgID))
	}
	return strings.Join(parts, "; ")
}

// New returns nil when there are no issues.
func New(issues []FieldIssue) *ValidationError {
	if len(issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: issues}
}
