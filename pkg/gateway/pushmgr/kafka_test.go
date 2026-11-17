package pushmgr

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaConfigValidate(t *testing.T) {
	config := KafkaConfig{
		BrokerURL: "localhost:9092, kafka://broker-2:9093",
	}
	require.NoError(t, config.Validate())
	assert.Equal(t, []string{"localhost:9092", "broker-2:9093"}, config.brokers())

	config.BrokerURL = "localhost"
	require.Error(t, config.Validate())
	config.BrokerURL = "localhost:9092"
}

func TestNewKafkaConnector(t *testing.T) {
	connector, err := NewKafkaConnector("connector-1", json.RawMessage(`{
		"brokerUrl":"localhost:9092",
		"clientId":"gateway-push-test"
	}`))
	require.NoError(t, err)
	assert.Equal(t, ConnectorTypeKafka, connector.Type())
	assert.Equal(t, "gateway-push-test", connector.config.ClientID)
}
