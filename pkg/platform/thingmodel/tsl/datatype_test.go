package tsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(v string) *string { return &v }

func validProperty() Property {
	return Property{
		Identifier: "temperature",
		Name:       "温度",
		RWFlag:     RWRead,
		DataType:   DataTypeDetail{Type: DataTypeDouble, Min: strPtr("0"), Max: strPtr("100")},
	}
}

func TestNormalizeDropsMismatchedFields(t *testing.T) {
	detail := DataTypeDetail{
		Type:     DataTypeInt,
		Unit:     strPtr("℃"),
		EnumSpec: &EnumSpec{Values: []EnumValue{{Value: 1, Desc: "a"}}},
		ArraySpec: &ArraySpec{
			Element: DataTypeDetail{Type: DataTypeStruct, EnumSpec: &EnumSpec{}},
		},
		StructSpec: &StructSpec{Fields: []StructField{{Identifier: "x"}}},
	}
	detail.Normalize()

	assert.Equal(t, "℃", *detail.Unit)
	assert.Nil(t, detail.EnumSpec)
	assert.Nil(t, detail.ArraySpec)
	assert.Nil(t, detail.StructSpec)
}

func TestNormalizeArrayElementRecursively(t *testing.T) {
	detail := DataTypeDetail{
		Type: DataTypeArray,
		ArraySpec: &ArraySpec{
			Element: DataTypeDetail{
				Type:       DataTypeBool,
				EnumSpec:   &EnumSpec{Values: []EnumValue{{Value: 1}}},
				TrueValue:  strPtr("on"),
				FalseValue: strPtr("off"),
			},
		},
	}
	detail.Normalize()

	require.NotNil(t, detail.ArraySpec)
	assert.Equal(t, DataTypeBool, detail.ArraySpec.Element.Type)
	assert.Nil(t, detail.ArraySpec.Element.EnumSpec)
	assert.Equal(t, "on", *detail.ArraySpec.Element.TrueValue)
}

func TestValidateValidProperty(t *testing.T) {
	p := validProperty()
	assert.Empty(t, p.Validate("properties[0]"))
}

func TestValidateIdentifierRules(t *testing.T) {
	cases := []struct {
		name       string
		identifier string
		msgID      string
	}{
		{"empty", "", msgIdentifierRequired},
		{"starts-with-digit", "1temp", msgIdentifierInvalid},
		{"contains-hyphen", "temp-a", msgIdentifierInvalid},
		{"reserved", "time", msgIdentifierReserved},
		{"sql-keyword", "select", msgIdentifierKeyword},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validProperty()
			p.Identifier = tc.identifier
			issues := p.Validate("properties[0]")
			require.NotEmpty(t, issues)
			assert.Equal(t, tc.msgID, issues[0].MsgID)
			assert.Equal(t, "properties[0].identifier", issues[0].Field)
		})
	}
}

func TestValidateNameRules(t *testing.T) {
	p := validProperty()
	p.Name = "1温度"
	issues := p.Validate("properties[0]")
	require.NotEmpty(t, issues)
	assert.Equal(t, msgNameInvalid, issues[0].MsgID)
	assert.Equal(t, "properties[0].name", issues[0].Field)
}

func TestValidateDataTypeRules(t *testing.T) {
	cases := []struct {
		name   string
		detail DataTypeDetail
		field  string
		msgID  string
	}{
		{"unknown", DataTypeDetail{Type: "date"}, "properties[0].dataType.type", msgDataTypeUnknown},
		{"enum-missing", DataTypeDetail{Type: DataTypeEnum}, "properties[0].dataType.enumSpec.values", msgEnumRequired},
		{"array-missing", DataTypeDetail{Type: DataTypeArray}, "properties[0].dataType.arraySpec", msgArrayRequired},
		{"struct-missing", DataTypeDetail{Type: DataTypeStruct}, "properties[0].dataType.structSpec.fields", msgStructRequired},
		{"array-size", DataTypeDetail{Type: DataTypeArray, ArraySpec: &ArraySpec{Size: strPtr("0"), Element: DataTypeDetail{Type: DataTypeInt}}}, "properties[0].dataType.arraySpec.size", msgArraySizeInvalid},
		{"range", DataTypeDetail{Type: DataTypeInt, Min: strPtr("10"), Max: strPtr("1")}, "properties[0].dataType.min", msgNumericRangeInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validProperty()
			p.DataType = tc.detail
			issues := p.Validate("properties[0]")
			require.NotEmpty(t, issues)
			assert.Equal(t, tc.field, issues[0].Field)
			assert.Equal(t, tc.msgID, issues[0].MsgID)
		})
	}
}

func TestValidateNestedFieldPaths(t *testing.T) {
	p := validProperty()
	p.DataType = DataTypeDetail{
		Type: DataTypeStruct,
		StructSpec: &StructSpec{Fields: []StructField{
			{
				Identifier: "high",
				Name:       "上限",
				DataType: DataTypeDetail{
					Type: DataTypeArray,
					ArraySpec: &ArraySpec{
						Element: DataTypeDetail{Type: "date"},
					},
				},
			},
		}},
	}
	issues := p.Validate("properties[0]")
	require.NotEmpty(t, issues)
	assert.Equal(t, "properties[0].dataType.structSpec.fields[0].dataType.arraySpec.element.type", issues[0].Field)
	assert.Equal(t, msgDataTypeUnknown, issues[0].MsgID)
}

func TestValidateDocumentDuplicates(t *testing.T) {
	first := validProperty()
	first.Name = "Temperature"
	second := validProperty()
	second.Name = "temperature"

	doc := ThingModel{Name: "model", Properties: []Property{first, second}}
	verr := doc.Validate()
	require.NotNil(t, verr)
	require.Len(t, verr.Issues, 2)
	assert.Equal(t, "properties[1].identifier", verr.Issues[0].Field)
	assert.Equal(t, msgDuplicateIdentifier, verr.Issues[0].MsgID)
	assert.Equal(t, "properties[1].name", verr.Issues[1].Field)
	assert.Equal(t, msgDuplicateName, verr.Issues[1].MsgID)
}

func TestValidateKey(t *testing.T) {
	assert.Empty(t, ValidateKey("tm_01k8z3m4q7x9v2b6n1c5d0e8fg"))
	assert.NotEmpty(t, ValidateKey("1bad"))
	assert.NotEmpty(t, ValidateKey("ab"))
	assert.NotEmpty(t, ValidateKey("bad key"))
}

func validFeature() Feature {
	return Feature{
		Identifier: "ratedVoltage",
		Name:       "额定电压",
		DataType:   DataTypeDetail{Type: DataTypeDouble, Unit: strPtr("V")},
	}
}

func TestValidateFeature(t *testing.T) {
	feature := validFeature()
	assert.Empty(t, feature.Validate("features[0]"))

	feature.Identifier = "1bad"
	issues := feature.Validate("features[0]")
	require.NotEmpty(t, issues)
	assert.Equal(t, "features[0].identifier", issues[0].Field)
}

func TestValidateFeaturesAllowsPropertyIdentifierCollision(t *testing.T) {
	property := validProperty()
	feature := validFeature()
	feature.Identifier = property.Identifier

	doc := ThingModel{Name: "model", Properties: []Property{property}, Features: []Feature{feature}}
	assert.Nil(t, doc.Validate())
}

func TestValidateDuplicateFeatures(t *testing.T) {
	first := validFeature()
	second := validFeature()
	second.Name = "额定电压2"
	doc := ThingModel{Name: "model", Features: []Feature{first, second}}
	verr := doc.Validate()
	require.NotNil(t, verr)
	require.NotEmpty(t, verr.Issues)
	assert.Equal(t, "features[1].identifier", verr.Issues[0].Field)
	assert.Equal(t, msgDuplicateIdentifier, verr.Issues[0].MsgID)
}

func TestFeatureNormalize(t *testing.T) {
	feature := Feature{
		Identifier: "  ratedVoltage ",
		Name:       " 额定电压 ",
		DataType:   DataTypeDetail{Type: DataTypeInt, EnumSpec: &EnumSpec{Values: []EnumValue{{Value: 1}}}},
	}
	feature.Normalize()
	assert.Equal(t, "ratedVoltage", feature.Identifier)
	assert.Equal(t, "额定电压", feature.Name)
	assert.Nil(t, feature.DataType.EnumSpec)
}
