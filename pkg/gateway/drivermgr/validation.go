package drivermgr

import "fmt"

// FieldIssue is one localized field validation failure.
type FieldIssue struct {
	Field string
	MsgID string
}

// ValidationError carries field-level driver validation failures.
type ValidationError struct {
	Issues []FieldIssue
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "driver validation failed"
	}
	return fmt.Sprintf("driver validation failed: %s", e.Issues[0].Field)
}
