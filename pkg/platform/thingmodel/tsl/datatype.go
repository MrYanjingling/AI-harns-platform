// Package tsl defines the thing specification language value objects used by
// the thing model module. It has no dependency on the transport or storage
// layers, so both the API DTOs and the domain model can share it.
package tsl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DataType is the discriminator of a thing model data type.
type DataType string

const (
	DataTypeInt    DataType = "int"
	DataTypeLong   DataType = "long"
	DataTypeFloat  DataType = "float"
	DataTypeDouble DataType = "double"
	DataTypeBool   DataType = "bool"
	DataTypeEnum   DataType = "enum"
	DataTypeText   DataType = "text"
	DataTypeTime   DataType = "time"
	DataTypeArray  DataType = "array"
	DataTypeStruct DataType = "struct"
)

// Read/write flags of a property.
const (
	RWRead      = "r"
	RWWrite     = "w"
	RWReadWrite = "rw"
)

var dataTypes = map[DataType]struct{}{
	DataTypeInt:    {},
	DataTypeLong:   {},
	DataTypeFloat:  {},
	DataTypeDouble: {},
	DataTypeBool:   {},
	DataTypeEnum:   {},
	DataTypeText:   {},
	DataTypeTime:   {},
	DataTypeArray:  {},
	DataTypeStruct: {},
}

// Valid reports whether the data type is supported.
func (t DataType) Valid() bool {
	_, ok := dataTypes[t]
	return ok
}

// IsNumeric reports whether the data type is numeric.
func (t DataType) IsNumeric() bool {
	switch t {
	case DataTypeInt, DataTypeLong, DataTypeFloat, DataTypeDouble:
		return true
	}
	return false
}

// DataTypeDetail describes a property data type. Type is the discriminator and
// only the spec matching Type is meaningful.
type DataTypeDetail struct {
	Type DataType `json:"type"`

	Unit     *string `json:"unit,omitempty"`
	UnitName *string `json:"unitName,omitempty"`
	Min      *string `json:"min,omitempty"`
	Max      *string `json:"max,omitempty"`

	TrueValue  *string `json:"trueValue,omitempty"`
	FalseValue *string `json:"falseValue,omitempty"`
	Length     *string `json:"length,omitempty"`

	EnumSpec   *EnumSpec   `json:"enumSpec,omitempty"`
	ArraySpec  *ArraySpec  `json:"arraySpec,omitempty"`
	StructSpec *StructSpec `json:"structSpec,omitempty"`
}

// EnumSpec is the specification of an enum data type.
type EnumSpec struct {
	Values []EnumValue `json:"values"`
}

// EnumValue is a single enum member.
type EnumValue struct {
	Value int    `json:"value"`
	Desc  string `json:"desc"`
}

// ArraySpec is the specification of an array data type.
type ArraySpec struct {
	Size    *string        `json:"size,omitempty"`
	Element DataTypeDetail `json:"element"`
}

// StructSpec is the specification of a struct data type.
type StructSpec struct {
	Fields []StructField `json:"fields"`
}

// StructField is a single struct member.
type StructField struct {
	Identifier string         `json:"identifier"`
	Name       string         `json:"name"`
	DataType   DataTypeDetail `json:"dataType"`
}

// Property is a thing model property value object, shared by the domain model,
// API DTOs and import/export formats.
type Property struct {
	Identifier  string                 `json:"identifier"`
	Name        string                 `json:"name"`
	DataType    DataTypeDetail         `json:"dataType"`
	RWFlag      string                 `json:"rwFlag"`
	Required    bool                   `json:"required"`
	Description string                 `json:"description,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
	Sort        int                    `json:"sort"`
}

// Feature is a static thing model feature definition.
type Feature struct {
	Identifier  string                 `json:"identifier"`
	Name        string                 `json:"name"`
	DataType    DataTypeDetail         `json:"dataType"`
	Description string                 `json:"description,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
	Sort        int                    `json:"sort"`
}

// ThingModel is the import/export document of a thing model.
type ThingModel struct {
	ThingModelKey string     `json:"thingModelKey,omitempty"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	Properties    []Property `json:"properties"`
	Features      []Feature  `json:"features,omitempty"`
}

// FieldIssue is a single field-level validation failure. Field is a JSON path
// such as properties[0].dataType.arraySpec.element.type.
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

// NewValidationError returns nil when there are no issues.
func NewValidationError(issues []FieldIssue) *ValidationError {
	if len(issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: issues}
}

const (
	msgKeyInvalid           = "validate.thing_model_key_invalid"
	msgNameRequired         = "validate.thing_model_name_required"
	msgNameInvalid          = "validate.thing_model_name_invalid"
	msgIdentifierRequired   = "validate.thing_model_identifier_required"
	msgIdentifierInvalid    = "validate.thing_model_identifier_invalid"
	msgIdentifierReserved   = "validate.thing_model_identifier_reserved"
	msgIdentifierKeyword    = "validate.thing_model_identifier_keyword"
	msgDataTypeRequired     = "validate.thing_model_data_type_required"
	msgDataTypeUnknown      = "validate.thing_model_data_type_unknown"
	msgEnumRequired         = "validate.thing_model_enum_required"
	msgEnumValueDuplicate   = "validate.thing_model_enum_value_duplicate"
	msgArrayRequired        = "validate.thing_model_array_required"
	msgArraySizeInvalid     = "validate.thing_model_array_size_invalid"
	msgStructRequired       = "validate.thing_model_struct_required"
	msgStructFieldRequired  = "validate.thing_model_struct_field_required"
	msgStructFieldDuplicate = "validate.thing_model_struct_field_duplicate"
	msgNumericRangeInvalid  = "validate.thing_model_numeric_range_invalid"
	msgTextLengthInvalid    = "validate.thing_model_text_length_invalid"
	msgRWFlagInvalid        = "validate.thing_model_rw_flag_invalid"
	msgDuplicateIdentifier  = "validate.thing_model_duplicate_identifier"
	msgDuplicateName        = "validate.thing_model_duplicate_name"
)

var (
	identifierPattern    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	thingModelKeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{3,63}$`)
)

var reservedIdentifiers = map[string]struct{}{
	"time":      {},
	"device_id": {},
}

var sqlKeywords = map[string]struct{}{
	"add": {}, "all": {}, "alter": {}, "and": {}, "any": {}, "as": {}, "asc": {},
	"attribute": {}, "authorization": {}, "between": {}, "by": {}, "case": {}, "cast": {},
	"check": {}, "column": {}, "constraint": {}, "create": {}, "cross": {},
	"current_date": {}, "current_time": {}, "current_timestamp": {}, "database": {},
	"default": {}, "delete": {}, "desc": {}, "distinct": {}, "drop": {}, "else": {},
	"end": {}, "engine": {}, "escape": {}, "except": {}, "exists": {}, "false": {},
	"fetch": {}, "field": {}, "for": {}, "foreign": {}, "from": {}, "full": {},
	"function": {}, "grant": {}, "group": {}, "hash": {}, "having": {}, "in": {},
	"index": {}, "inner": {}, "insert": {}, "intersect": {}, "into": {}, "is": {},
	"join": {}, "key": {}, "left": {}, "like": {}, "limit": {}, "local": {},
	"measurement": {}, "not": {}, "null": {}, "offset": {}, "on": {}, "or": {},
	"order": {}, "outer": {}, "overwrite": {}, "partition": {}, "partitions": {},
	"primary": {}, "procedure": {}, "references": {}, "right": {}, "schema": {},
	"select": {}, "sequence": {}, "set": {}, "storage_type": {}, "table": {},
	"tag": {}, "then": {}, "timestamp": {}, "to": {}, "true": {}, "truncate": {},
	"ttl": {}, "union": {}, "unique": {}, "update": {}, "update_mode": {},
	"user": {}, "using": {}, "value": {}, "values": {}, "view": {}, "when": {},
	"where": {}, "with": {},
}

// ValidThingModelKey reports whether a user supplied key is well formed.
func ValidThingModelKey(key string) bool {
	return thingModelKeyPattern.MatchString(key)
}

// Normalize drops fields that do not belong to the current data type and
// applies defaults. It is intentionally lenient: mismatched fields are
// discarded instead of rejected.
func (d *DataTypeDetail) Normalize() {
	if d == nil {
		return
	}
	switch d.Type {
	case DataTypeInt, DataTypeLong, DataTypeFloat, DataTypeDouble:
		d.TrueValue, d.FalseValue, d.Length = nil, nil, nil
		d.EnumSpec, d.ArraySpec, d.StructSpec = nil, nil, nil
	case DataTypeBool:
		d.Unit, d.UnitName, d.Min, d.Max = nil, nil, nil, nil
		d.Length = nil
		d.EnumSpec, d.ArraySpec, d.StructSpec = nil, nil, nil
	case DataTypeEnum:
		d.Unit, d.UnitName, d.Min, d.Max = nil, nil, nil, nil
		d.TrueValue, d.FalseValue, d.Length = nil, nil, nil
		d.ArraySpec, d.StructSpec = nil, nil
	case DataTypeText:
		d.Unit, d.UnitName, d.Min, d.Max = nil, nil, nil, nil
		d.TrueValue, d.FalseValue = nil, nil
		d.EnumSpec, d.ArraySpec, d.StructSpec = nil, nil, nil
	case DataTypeTime:
		d.Unit, d.UnitName, d.Min, d.Max = nil, nil, nil, nil
		d.TrueValue, d.FalseValue, d.Length = nil, nil, nil
		d.EnumSpec, d.ArraySpec, d.StructSpec = nil, nil, nil
	case DataTypeArray:
		d.Unit, d.UnitName, d.Min, d.Max = nil, nil, nil, nil
		d.TrueValue, d.FalseValue, d.Length = nil, nil, nil
		d.EnumSpec, d.StructSpec = nil, nil
		if d.ArraySpec != nil {
			d.ArraySpec.Element.Normalize()
		}
	case DataTypeStruct:
		d.Unit, d.UnitName, d.Min, d.Max = nil, nil, nil, nil
		d.TrueValue, d.FalseValue, d.Length = nil, nil, nil
		d.EnumSpec, d.ArraySpec = nil, nil
		if d.StructSpec != nil {
			for i := range d.StructSpec.Fields {
				d.StructSpec.Fields[i].DataType.Normalize()
			}
		}
	}
}

// Normalize applies defaults to a property.
func (p *Property) Normalize() {
	if p == nil {
		return
	}
	p.Identifier = strings.TrimSpace(p.Identifier)
	p.Name = strings.TrimSpace(p.Name)
	if p.RWFlag == "" {
		p.RWFlag = RWRead
	}
	p.DataType.Normalize()
}

// Normalize normalizes a feature definition.
func (f *Feature) Normalize() {
	if f == nil {
		return
	}
	f.Identifier = strings.TrimSpace(f.Identifier)
	f.Name = strings.TrimSpace(f.Name)
	f.DataType.Normalize()
}

// Normalize normalizes every property and feature of the document.
func (m *ThingModel) Normalize() {
	if m == nil {
		return
	}
	m.ThingModelKey = strings.TrimSpace(m.ThingModelKey)
	m.Name = strings.TrimSpace(m.Name)
	for i := range m.Properties {
		m.Properties[i].Normalize()
	}
	for i := range m.Features {
		m.Features[i].Normalize()
	}
}

// Validate checks a property and returns field-level issues.
func (p *Property) Validate(path string) []FieldIssue {
	issues := make([]FieldIssue, 0, 4)
	if p == nil {
		return append(issues, FieldIssue{Field: path, MsgID: msgIdentifierRequired})
	}

	id := p.Identifier
	switch {
	case id == "":
		issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgIdentifierRequired})
	case len(id) > 64 || !identifierPattern.MatchString(id):
		issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgIdentifierInvalid, Args: []interface{}{id}})
	default:
		lower := strings.ToLower(id)
		if _, ok := reservedIdentifiers[lower]; ok {
			issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgIdentifierReserved, Args: []interface{}{id}})
		} else if _, ok := sqlKeywords[lower]; ok {
			issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgIdentifierKeyword, Args: []interface{}{id}})
		}
	}

	switch {
	case p.Name == "":
		issues = append(issues, FieldIssue{Field: path + ".name", MsgID: msgNameRequired})
	case utf8.RuneCountInString(p.Name) > 64:
		issues = append(issues, FieldIssue{Field: path + ".name", MsgID: msgNameInvalid, Args: []interface{}{p.Name}})
	default:
		first, _ := utf8.DecodeRuneInString(p.Name)
		if !unicode.IsLetter(first) {
			issues = append(issues, FieldIssue{Field: path + ".name", MsgID: msgNameInvalid, Args: []interface{}{p.Name}})
		}
	}

	switch p.RWFlag {
	case "", RWRead, RWWrite, RWReadWrite:
	default:
		issues = append(issues, FieldIssue{Field: path + ".rwFlag", MsgID: msgRWFlagInvalid, Args: []interface{}{p.RWFlag}})
	}

	issues = append(issues, p.DataType.Validate(path+".dataType")...)
	return issues
}

// Validate checks a feature definition and returns field-level issues.
func (f *Feature) Validate(path string) []FieldIssue {
	issues := make([]FieldIssue, 0, 3)
	if f == nil {
		return append(issues, FieldIssue{Field: path, MsgID: msgIdentifierRequired})
	}

	id := f.Identifier
	switch {
	case id == "":
		issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgIdentifierRequired})
	case len(id) > 64 || !identifierPattern.MatchString(id):
		issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgIdentifierInvalid, Args: []interface{}{id}})
	default:
		lower := strings.ToLower(id)
		if _, ok := reservedIdentifiers[lower]; ok {
			issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgIdentifierReserved, Args: []interface{}{id}})
		} else if _, ok := sqlKeywords[lower]; ok {
			issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgIdentifierKeyword, Args: []interface{}{id}})
		}
	}

	switch {
	case f.Name == "":
		issues = append(issues, FieldIssue{Field: path + ".name", MsgID: msgNameRequired})
	case utf8.RuneCountInString(f.Name) > 64:
		issues = append(issues, FieldIssue{Field: path + ".name", MsgID: msgNameInvalid, Args: []interface{}{f.Name}})
	default:
		first, _ := utf8.DecodeRuneInString(f.Name)
		if !unicode.IsLetter(first) {
			issues = append(issues, FieldIssue{Field: path + ".name", MsgID: msgNameInvalid, Args: []interface{}{f.Name}})
		}
	}

	issues = append(issues, f.DataType.Validate(path+".dataType")...)
	return issues
}

// Validate checks a data type and returns field-level issues.
func (d *DataTypeDetail) Validate(path string) []FieldIssue {
	issues := make([]FieldIssue, 0, 4)
	if d.Type == "" {
		return append(issues, FieldIssue{Field: path + ".type", MsgID: msgDataTypeRequired})
	}
	if !d.Type.Valid() {
		return append(issues, FieldIssue{Field: path + ".type", MsgID: msgDataTypeUnknown, Args: []interface{}{string(d.Type)}})
	}

	if d.Type.IsNumeric() {
		issues = append(issues, validateNumericRange(path, d)...)
	}
	switch d.Type {
	case DataTypeEnum:
		if d.EnumSpec == nil || len(d.EnumSpec.Values) == 0 {
			issues = append(issues, FieldIssue{Field: path + ".enumSpec.values", MsgID: msgEnumRequired})
			break
		}
		seen := make(map[int]struct{}, len(d.EnumSpec.Values))
		for _, v := range d.EnumSpec.Values {
			if _, ok := seen[v.Value]; ok {
				issues = append(issues, FieldIssue{Field: path + ".enumSpec.values", MsgID: msgEnumValueDuplicate, Args: []interface{}{v.Value}})
			}
			seen[v.Value] = struct{}{}
		}
	case DataTypeText:
		if d.Length != nil && *d.Length != "" {
			n, err := strconv.Atoi(strings.TrimSpace(*d.Length))
			if err != nil || n <= 0 {
				issues = append(issues, FieldIssue{Field: path + ".length", MsgID: msgTextLengthInvalid, Args: []interface{}{*d.Length}})
			}
		}
	case DataTypeArray:
		if d.ArraySpec == nil {
			issues = append(issues, FieldIssue{Field: path + ".arraySpec", MsgID: msgArrayRequired})
			break
		}
		if d.ArraySpec.Size != nil && strings.TrimSpace(*d.ArraySpec.Size) != "" {
			n, err := strconv.Atoi(strings.TrimSpace(*d.ArraySpec.Size))
			if err != nil || n < 1 || n > 512 {
				issues = append(issues, FieldIssue{Field: path + ".arraySpec.size", MsgID: msgArraySizeInvalid, Args: []interface{}{*d.ArraySpec.Size}})
			}
		}
		issues = append(issues, d.ArraySpec.Element.Validate(path+".arraySpec.element")...)
	case DataTypeStruct:
		if d.StructSpec == nil || len(d.StructSpec.Fields) == 0 {
			issues = append(issues, FieldIssue{Field: path + ".structSpec.fields", MsgID: msgStructRequired})
			break
		}
		seen := make(map[string]struct{}, len(d.StructSpec.Fields))
		for i, f := range d.StructSpec.Fields {
			fieldPath := fmt.Sprintf("%s.structSpec.fields[%d]", path, i)
			identifier := strings.TrimSpace(f.Identifier)
			if identifier == "" {
				issues = append(issues, FieldIssue{Field: fieldPath + ".identifier", MsgID: msgStructFieldRequired})
			} else if !identifierPattern.MatchString(identifier) || len(identifier) > 64 {
				issues = append(issues, FieldIssue{Field: fieldPath + ".identifier", MsgID: msgIdentifierInvalid, Args: []interface{}{identifier}})
			} else if _, ok := seen[identifier]; ok {
				issues = append(issues, FieldIssue{Field: fieldPath + ".identifier", MsgID: msgStructFieldDuplicate, Args: []interface{}{identifier}})
			}
			seen[identifier] = struct{}{}
			issues = append(issues, f.DataType.Validate(fieldPath+".dataType")...)
		}
	}
	return issues
}

func validateNumericRange(path string, d *DataTypeDetail) []FieldIssue {
	if d.Min == nil || d.Max == nil || *d.Min == "" || *d.Max == "" {
		return nil
	}
	minVal, err := strconv.ParseFloat(strings.TrimSpace(*d.Min), 64)
	if err != nil {
		return nil
	}
	maxVal, err := strconv.ParseFloat(strings.TrimSpace(*d.Max), 64)
	if err != nil {
		return nil
	}
	if minVal > maxVal {
		return []FieldIssue{{
			Field: path + ".min",
			MsgID: msgNumericRangeInvalid,
			Args:  []interface{}{*d.Min, *d.Max},
		}}
	}
	return nil
}

// Validate checks the whole document, including document-level uniqueness.
func (m *ThingModel) Validate() *ValidationError {
	if m == nil {
		return NewValidationError([]FieldIssue{{Field: "thingModel", MsgID: msgNameRequired}})
	}
	issues := make([]FieldIssue, 0, len(m.Properties))

	switch {
	case m.Name == "":
		issues = append(issues, FieldIssue{Field: "name", MsgID: msgNameRequired})
	case utf8.RuneCountInString(m.Name) > 64:
		issues = append(issues, FieldIssue{Field: "name", MsgID: msgNameInvalid, Args: []interface{}{m.Name}})
	}
	if m.ThingModelKey != "" && !ValidThingModelKey(m.ThingModelKey) {
		issues = append(issues, FieldIssue{Field: "thingModelKey", MsgID: msgKeyInvalid, Args: []interface{}{m.ThingModelKey}})
	}

	identifierSeen := make(map[string]struct{}, len(m.Properties))
	nameSeen := make(map[string]struct{}, len(m.Properties))
	for i := range m.Properties {
		path := fmt.Sprintf("properties[%d]", i)
		p := &m.Properties[i]
		issues = append(issues, p.Validate(path)...)

		if id := p.Identifier; id != "" {
			key := strings.ToLower(id)
			if _, ok := identifierSeen[key]; ok {
				issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgDuplicateIdentifier, Args: []interface{}{id}})
			} else {
				identifierSeen[key] = struct{}{}
			}
		}
		if name := p.Name; name != "" {
			key := strings.ToLower(name)
			if _, ok := nameSeen[key]; ok {
				issues = append(issues, FieldIssue{Field: path + ".name", MsgID: msgDuplicateName, Args: []interface{}{name}})
			} else {
				nameSeen[key] = struct{}{}
			}
		}
	}

	featureIdentifierSeen := make(map[string]struct{}, len(m.Features))
	featureNameSeen := make(map[string]struct{}, len(m.Features))
	for i := range m.Features {
		path := fmt.Sprintf("features[%d]", i)
		f := &m.Features[i]
		issues = append(issues, f.Validate(path)...)

		if id := f.Identifier; id != "" {
			key := strings.ToLower(id)
			if _, ok := featureIdentifierSeen[key]; ok {
				issues = append(issues, FieldIssue{Field: path + ".identifier", MsgID: msgDuplicateIdentifier, Args: []interface{}{id}})
			} else {
				featureIdentifierSeen[key] = struct{}{}
			}
		}
		if name := f.Name; name != "" {
			key := strings.ToLower(name)
			if _, ok := featureNameSeen[key]; ok {
				issues = append(issues, FieldIssue{Field: path + ".name", MsgID: msgDuplicateName, Args: []interface{}{name}})
			} else {
				featureNameSeen[key] = struct{}{}
			}
		}
	}
	return NewValidationError(issues)
}

// ValidateName checks a thing model name.
func ValidateName(name string) []FieldIssue {
	switch {
	case name == "":
		return []FieldIssue{{Field: "name", MsgID: msgNameRequired}}
	case utf8.RuneCountInString(name) > 64:
		return []FieldIssue{{Field: "name", MsgID: msgNameInvalid, Args: []interface{}{name}}}
	}
	return nil
}

// ValidateKey checks a thing model key.
func ValidateKey(key string) []FieldIssue {
	if !ValidThingModelKey(key) {
		return []FieldIssue{{Field: "thingModelKey", MsgID: msgKeyInvalid, Args: []interface{}{key}}}
	}
	return nil
}
