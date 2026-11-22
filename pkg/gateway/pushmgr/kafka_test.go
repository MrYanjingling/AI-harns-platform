package pushmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"syscall"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
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

func TestKafkaConnectorErrorClassification(t *testing.T) {
	assert.False(t, isKafkaConnectorError(nil))
	assert.False(t, isKafkaConnectorError(context.Canceled))
	assert.False(t, isKafkaConnectorError(kafka.MessageTooLargeError{Message: kafka.Message{}}))
	assert.False(t, isKafkaConnectorError(kafka.TopicAuthorizationFailed))

	assert.True(t, isKafkaConnectorError(kafka.BrokerNotAvailable))
	assert.True(t, isKafkaConnectorError(kafka.NetworkException))
	assert.True(t, isKafkaConnectorError(io.EOF))
	assert.True(t, isKafkaConnectorError(&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}))
	assert.True(t, isKafkaConnectorError(kafka.WriteErrors{net.ErrClosed}))
}

func TestKafkaReconnectRecovers(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	connector, err := NewKafkaConnector("connector-1", json.RawMessage(fmt.Sprintf(`{
		"brokerUrl":%q,
		"clientId":"gateway-push-test",
		"groupId":"gateway-push-group",
		"saslMechanism":"none"
	}`, addr)))
	require.NoError(t, err)

	connected := make(chan struct{}, 1)
	connector.SetConnectedHandler(func() {
		select {
		case connected <- struct{}{}:
		default:
		}
	})
	require.NoError(t, connector.Connect(context.Background()))

	require.NoError(t, listener.Close())
	connector.startReconnect()
	time.Sleep(200 * time.Millisecond)
	listener, err = net.Listen("tcp", addr)
	require.NoError(t, err)
	defer listener.Close()

	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		t.Fatal("kafka connector did not report reconnection")
	}
	require.NoError(t, connector.Close(context.Background()))
}

func TestKafkaReconnectStopsOnClose(t *testing.T) {
	connector, err := NewKafkaConnector("connector-1", json.RawMessage(`{
		"brokerUrl":"127.0.0.1:1",
		"clientId":"gateway-push-test",
		"groupId":"gateway-push-group",
		"saslMechanism":"none"
	}`))
	require.NoError(t, err)
	connector.startReconnect()

	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, connector.Close(context.Background()))
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("kafka reconnect loop did not stop on close")
	}
}
