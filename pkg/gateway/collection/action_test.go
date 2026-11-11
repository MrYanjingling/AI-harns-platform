package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeActionValueWithRateOffset(t *testing.T) {
	variable := &Variable{DataType: "int16", Rate: 0.1, Offset: 1}
	value, err := DecodeActionValue(variable, "20.5")
	require.NoError(t, err)
	assert.Equal(t, int16(195), value)

	variable = &Variable{DataType: "float32", Rate: 0.1}
	value, err = DecodeActionValue(variable, "20.5")
	require.NoError(t, err)
	assert.Equal(t, float32(205), value)
}

func TestDecodeActionValueRejectsInvalidNumbers(t *testing.T) {
	for _, value := range []string{"+1", "1e-3", "1/2", ""} {
		_, err := DecodeActionValue(&Variable{DataType: "int16", Rate: 1}, value)
		require.Error(t, err, value)
	}
	_, err := DecodeActionValue(&Variable{DataType: "int16", Rate: 0.1}, "20.55")
	require.Error(t, err)
}

func TestDecodeActionBool(t *testing.T) {
	variable := &Variable{DataType: "bool"}
	for value, expected := range map[string]bool{"true": true, "FALSE": false, "1": true, "0": false} {
		got, err := DecodeActionValue(variable, value)
		require.NoError(t, err, value)
		assert.Equal(t, expected, got, value)
	}
	_, err := DecodeActionValue(variable, "yes")
	require.Error(t, err)
}

func TestDecodeActionString(t *testing.T) {
	value, err := DecodeActionValue(&Variable{DataType: "STRING"}, "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", value)
}
