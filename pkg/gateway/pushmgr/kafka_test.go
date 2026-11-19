package pushmgr

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaConfigValidate(t *testing.T) {
	config := KafkaConfig{
		BrokerURL: "localhost:9092, kafka://broker-2:9093",
	}
	config.Normalize()
	require.NoError(t, config.Validate())
	assert.Equal(t, []string{"localhost:9092", "broker-2:9093"}, config.brokers())
	assert.Regexp(t, regexp.MustCompile(`^harns-[a-z]{6}$`), config.ClientID)
	assert.Regexp(t, regexp.MustCompile(`^harns-[a-z]{6}$`), config.GroupID)
	assert.Equal(t, KafkaSASLNone, config.SASLMechanism)

	config.BrokerURL = "localhost"
	require.Error(t, config.Validate())
}

func TestKafkaConfigNormalizesSASL(t *testing.T) {
	config := KafkaConfig{
		BrokerURL: "localhost:9092",
		Username:  "user",
		Password:  "pass",
	}
	config.Normalize()
	assert.Equal(t, KafkaSASLPlain, config.SASLMechanism)

	config = KafkaConfig{
		BrokerURL:     "localhost:9092",
		ClientID:      "client",
		GroupID:       "group",
		SASLMechanism: "SCRAM-SHA-256",
		Username:      "user",
		Password:      "pass",
	}
	config.Normalize()
	assert.Equal(t, KafkaSASLScramSHA256, config.SASLMechanism)
	_, err := config.saslMechanism()
	require.NoError(t, err)

	config.SASLMechanism = KafkaSASLScramSHA512
	config.Normalize()
	_, err = config.saslMechanism()
	require.NoError(t, err)

	config.SASLMechanism = KafkaSASLNone
	config.Normalize()
	mechanism, err := config.saslMechanism()
	require.NoError(t, err)
	assert.Nil(t, mechanism)

	config.SASLMechanism = "invalid"
	config.Normalize()
	assert.Error(t, config.Validate())
}

func TestKafkaFactoryNormalizePersistsIDs(t *testing.T) {
	raw, err := NewKafkaFactory().Normalize("test", json.RawMessage(`{
		"brokerUrl":"localhost:9092",
		"username":"user",
		"password":"pass"
	}`))
	require.NoError(t, err)

	var config KafkaConfig
	require.NoError(t, json.Unmarshal(raw, &config))
	assert.Regexp(t, regexp.MustCompile(`^harns-[a-z]{6}$`), config.ClientID)
	assert.Regexp(t, regexp.MustCompile(`^harns-[a-z]{6}$`), config.GroupID)
	assert.Equal(t, KafkaSASLPlain, config.SASLMechanism)
}

func TestNewKafkaConnector(t *testing.T) {
	connector, err := NewKafkaConnector("connector-1", json.RawMessage(`{
		"brokerUrl":"localhost:9092",
		"clientId":"gateway-push-test",
		"groupId":"gateway-push-group",
		"saslMechanism":"none"
	}`))
	require.NoError(t, err)
	assert.Equal(t, ConnectorTypeKafka, connector.Type())
	assert.Equal(t, "gateway-push-test", connector.config.ClientID)
	assert.Equal(t, "gateway-push-group", connector.config.GroupID)
	assert.Equal(t, KafkaSASLNone, connector.config.SASLMechanism)
}
