package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceMetaJSONHasNoTopic(t *testing.T) {
	data, err := json.Marshal(&DeviceMeta{
		Name:        "device-1",
		DeviceCode:  "device-1",
		DeviceType:  "modbus",
		DeviceModel: "modbusTcp",
	})
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"topic"`)
}
