package drivermgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"harns-platform/pkg/gateway/validation"
)

// FieldType is a supported form field type.
type FieldType string

const (
	FieldTypeInput  FieldType = "input"
	FieldTypeSelect FieldType = "select"
)

// Schema describes the configurable fields of a driver.
type Schema struct {
	Fields []*Field `json:"fields"`
}

// Field describes one driver config field.
type Field struct {
	Name     string    `json:"name"`
	LabelZh  string    `json:"labelZh"`
	LabelEn  string    `json:"labelEn"`
	Regex    string    `json:"regex"`
	Sort     int       `json:"sort"`
	Type     FieldType `json:"type"`
	Required bool      `json:"required,omitempty"`
	Options  []*Option `json:"options,omitempty"`

	compiled *regexp.Regexp
}

// Option is one select option.
type Option struct {
	Value   string `json:"value"`
	LabelZh string `json:"labelZh"`
	LabelEn string `json:"labelEn"`
}

// ParseSchema parses and validates a driver schema.
func ParseSchema(raw json.RawMessage) (*Schema, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return &Schema{Fields: []*Field{}}, nil
	}
	var schema Schema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	if err := schema.Validate(); err != nil {
		return nil, err
	}
	return &schema, nil
}

// Validate validates schema definitions.
func (s *Schema) Validate() error {
	if s == nil {
		return fmt.Errorf("schema is nil")
	}
	names := make(map[string]struct{}, len(s.Fields))
	for i, field := range s.Fields {
		if field == nil {
			return fmt.Errorf("fields[%d] is null", i)
		}
		field.Name = strings.TrimSpace(field.Name)
		field.LabelZh = strings.TrimSpace(field.LabelZh)
		field.LabelEn = strings.TrimSpace(field.LabelEn)
		field.Regex = strings.TrimSpace(field.Regex)
		if field.Name == "" {
			return fmt.Errorf("fields[%d].name is required", i)
		}
		if _, exists := names[field.Name]; exists {
			return fmt.Errorf("duplicate field name %s", field.Name)
		}
		names[field.Name] = struct{}{}
		if field.LabelZh == "" || field.LabelEn == "" {
			return fmt.Errorf("field %s labels are required", field.Name)
		}
		switch field.Type {
		case FieldTypeInput:
		case FieldTypeSelect:
			if len(field.Options) == 0 {
				return fmt.Errorf("field %s select options are required", field.Name)
			}
		default:
			return fmt.Errorf("field %s type must be input or select", field.Name)
		}
		if field.Regex != "" {
			re, err := regexp.Compile(field.Regex)
			if err != nil {
				return fmt.Errorf("field %s regex is invalid: %w", field.Name, err)
			}
			field.compiled = re
		}
		if field.Type == FieldTypeSelect {
			values := make(map[string]struct{}, len(field.Options))
			for j, option := range field.Options {
				if option == nil || option.Value == "" || option.LabelZh == "" || option.LabelEn == "" {
					return fmt.Errorf("field %s options[%d] is invalid", field.Name, j)
				}
				if _, exists := values[option.Value]; exists {
					return fmt.Errorf("field %s has duplicate option value %s", field.Name, option.Value)
				}
				values[option.Value] = struct{}{}
			}
		}
	}
	return nil
}

// ValidateBody validates protocol fields in a request body.
func (s *Schema) ValidateBody(body []byte) error {
	if s == nil || len(s.Fields) == 0 {
		return nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(body, &values); err != nil {
		return err
	}
	issues := make([]validation.FieldIssue, 0)
	for _, field := range s.sortedFields() {
		raw, exists := values[field.Name]
		if !exists || len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
			if field.Required {
				issues = append(issues, validation.FieldIssue{
					Field: field.Name,
					MsgID: "validate.collection_field_required",
					Args:  []interface{}{field.Name},
				})
			}
			continue
		}
		value, err := rawString(raw)
		if err != nil {
			issues = append(issues, validation.FieldIssue{
				Field: field.Name,
				MsgID: "validate.collection_field_invalid",
				Args:  []interface{}{field.Name},
			})
			continue
		}
		if field.Type == FieldTypeSelect && !field.hasOption(value) {
			issues = append(issues, validation.FieldIssue{
				Field: field.Name,
				MsgID: "validate.collection_select_invalid",
				Args:  []interface{}{field.Name},
			})
			continue
		}
		if field.compiled != nil && !field.compiled.MatchString(value) {
			issues = append(issues, validation.FieldIssue{
				Field: field.Name,
				MsgID: "validate.collection_field_invalid",
				Args:  []interface{}{field.Name},
			})
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}

func (s *Schema) sortedFields() []*Field {
	fields := append([]*Field(nil), s.Fields...)
	for i := 1; i < len(fields); i++ {
		for j := i; j > 0 && fields[j].Sort < fields[j-1].Sort; j-- {
			fields[j], fields[j-1] = fields[j-1], fields[j]
		}
	}
	return fields
}

func (f *Field) hasOption(value string) bool {
	for _, option := range f.Options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func rawString(raw json.RawMessage) (string, error) {
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	switch v := value.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return fmt.Sprintf("%v", v), nil
	default:
		return fmt.Sprint(v), nil
	}
}
