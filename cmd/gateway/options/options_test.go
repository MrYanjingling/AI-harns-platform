package options

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestGatewayFlagsMatchPlatform(t *testing.T) {
	opts := NewDefaultOptions()
	fs := pflag.NewFlagSet("harns-gateway", pflag.ContinueOnError)
	opts.AddFlags(fs)

	for _, name := range []string{"port", "graceful-timeout", "cert-file", "key-file"} {
		assert.NotNil(t, fs.Lookup(name), "missing flag %s", name)
	}
	for _, name := range []string{"mqtt-broker-urls", "mqtt-username", "mqtt-password"} {
		assert.Nil(t, fs.Lookup(name), "unexpected flag %s", name)
	}
}

func TestMQTTConfigLoadsFromYAML(t *testing.T) {
	opts := NewDefaultOptions()
	data := []byte(`
mqtt-broker-urls:
  - tcp://mqtt.example.com:1883
mqtt-username: gateway-user
mqtt-password: gateway-pass
`)
	require.NoError(t, yaml.Unmarshal(data, opts))
	assert.Equal(t, []string{"tcp://mqtt.example.com:1883"}, opts.MqttBrokerUrls)
	assert.Equal(t, "gateway-user", opts.MqttUsername)
	assert.Equal(t, "gateway-pass", opts.MqttPassword)
}
