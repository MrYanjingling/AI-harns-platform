package drivermgr

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

func TestParseSchemaAndValidateBody(t *testing.T) {
	raw := json.RawMessage(`{
		"fields":[
			{"name":"host","labelZh":"地址","labelEn":"Host","regex":"^\\S+$","sort":2,"type":"input","required":true},
			{"name":"mode","labelZh":"模式","labelEn":"Mode","sort":1,"type":"select","required":true,"options":[
				{"value":"a","labelZh":"A","labelEn":"A"},
				{"value":"b","labelZh":"B","labelEn":"B"}
			]}
		]
	}`)
	schema, err := ParseSchema(raw)
	require.NoError(t, err)
	require.Len(t, schema.Fields, 2)

	require.NoError(t, schema.ValidateBody([]byte(`{"protocolConfig":{"host":"127.0.0.1","mode":"a"}}`)))
	require.Error(t, schema.ValidateBody([]byte(`{"protocolConfig":{"host":"","mode":"a"}}`)))
	require.Error(t, schema.ValidateBody([]byte(`{"protocolConfig":{"host":"127.0.0.1","mode":"c"}}`)))
	require.Error(t, schema.ValidateBody([]byte(`{"protocolConfig":{"mode":"a"}}`)))

	var fieldErr *validation.ValidationError
	err = schema.ValidateBody([]byte(`{"protocolConfig":{"host":"127.0.0.1","mode":"c"}}`))
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "protocolConfig.mode", fieldErr.Issues[0].Field)
}

func TestSchemaVariableAddress(t *testing.T) {
	raw := json.RawMessage(`{
		"fields":[],
		"variableAddress":{
			"labelZh":"变量地址","labelEn":"Variable Address",
			"regex":"^[0-9]+![1-4][0-9]+$"
		}
	}`)
	schema, err := ParseSchema(raw)
	require.NoError(t, err)
	require.NoError(t, schema.ValidateVariableAddresses([]*collection.Variable{{
		Name: "temperature", DataType: "float32", Address: "1!40001", AccessMode: collection.AccessModeRead,
	}}))
	err = schema.ValidateVariableAddresses([]*collection.Variable{{
		Name: "temperature", DataType: "float32", Address: "40001", AccessMode: collection.AccessModeRead,
	}})
	require.Error(t, err)
	var validationErr *validation.ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "variables[0].address", validationErr.Issues[0].Field)
}

func TestParseSchemaVariableAddressInvalidRegex(t *testing.T) {
	_, err := ParseSchema(json.RawMessage(`{
		"fields":[],
		"variableAddress":{"labelZh":"地址","labelEn":"Address","regex":"["}
	}`))
	require.Error(t, err)
}

func TestParseSchemaRejectsInvalidDefinitions(t *testing.T) {
	cases := []string{
		`{"fields":[{"name":"a","labelZh":"A","labelEn":"A","type":"input"},{"name":"a","labelZh":"A","labelEn":"A","type":"input"}]}`,
		`{"fields":[{"name":"a","labelZh":"A","labelEn":"A","type":"bad"}]}`,
		`{"fields":[{"name":"a","labelZh":"A","labelEn":"A","type":"input","regex":"["}]}`,
		`{"fields":[{"name":"a","labelZh":"A","labelEn":"A","type":"select"}]}`,
		`{"fields":[{"name":"a","labelZh":"A","labelEn":"A","type":"select","options":[{"value":"x","labelZh":"X","labelEn":"X"},{"value":"x","labelZh":"X","labelEn":"X"}]}]}`,
	}
	for _, raw := range cases {
		_, err := ParseSchema(json.RawMessage(raw))
		assert.Error(t, err, raw)
	}
}

func TestSchemaValidateDataTypes(t *testing.T) {
	raw := json.RawMessage(`{
		"fields":[],
		"variableAddress":{"labelZh":"地址","labelEn":"Address","regex":"^.*$"},
		"variableDataTypes":[
			{"value":"BOOL","labelZh":"布尔","labelEn":"Boolean"},
			{"value":"INT16","labelZh":"整数","labelEn":"Int16"}
		]
	}`)
	schema, err := ParseSchema(raw)
	require.NoError(t, err)
	require.Len(t, schema.VariableDataTypes, 2)

	require.NoError(t, schema.ValidateDataTypes([]*collection.Variable{
		{Name: "a", DataType: "BOOL"},
		{Name: "b", DataType: "int16"},
	}))
	err = schema.ValidateDataTypes([]*collection.Variable{{Name: "c", DataType: "NUMBER"}})
	require.Error(t, err)
	var validationErr *validation.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "variables[0].dataType", validationErr.Issues[0].Field)

	body := []byte(`{"variables":[{"name":"c","dataType":"FLOAT32"}]}`)
	err = schema.ValidateBody(body)
	require.Error(t, err)
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "variables[0].dataType", validationErr.Issues[0].Field)
}

func TestS7SchemaSupportsArrayAddresses(t *testing.T) {
	schema, err := ParseSchema(json.RawMessage(s71500Schema))
	require.NoError(t, err)
	for _, address := range []string{"DB1.DBB0.10", "DB1.DBW0.10", "DB1.DBD0.10", "DB1.DBX0.0", "I0", "IW0", "I0.0", "Q0", "QW0", "M0"} {
		require.NoError(t, schema.ValidateVariableAddresses([]*collection.Variable{{
			Name: "v", DataType: "STRING", Address: address, AccessMode: collection.AccessModeRead,
		}}), address)
	}
}
