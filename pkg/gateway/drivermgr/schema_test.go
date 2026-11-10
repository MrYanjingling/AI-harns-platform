package drivermgr

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	require.NoError(t, schema.ValidateBody([]byte(`{"host":"127.0.0.1","mode":"a"}`)))
	require.Error(t, schema.ValidateBody([]byte(`{"host":"","mode":"a"}`)))
	require.Error(t, schema.ValidateBody([]byte(`{"host":"127.0.0.1","mode":"c"}`)))
	require.Error(t, schema.ValidateBody([]byte(`{"mode":"a"}`)))

	var fieldErr *validation.ValidationError
	err = schema.ValidateBody([]byte(`{"host":"127.0.0.1","mode":"c"}`))
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "mode", fieldErr.Issues[0].Field)
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
