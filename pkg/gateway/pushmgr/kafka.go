package pushmgr

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

const (
	kafkaCloseTimeout   = 5 * time.Second
	kafkaConnectTimeout = 5 * time.Second
	kafkaIDPrefix       = "harns-"
	kafkaIDLength       = 6
	kafkaIDLetters      = "abcdefghijklmnopqrstuvwxyz"
)

const (
	KafkaSASLNone        = "none"
	KafkaSASLPlain       = "plain"
	KafkaSASLScramSHA256 = "scram-sha-256"
	KafkaSASLScramSHA512 = "scram-sha-512"
)

type KafkaConfig struct {
	BrokerURL     string `json:"brokerUrl"`
	ClientID      string `json:"clientId,omitempty"`
	GroupID       string `json:"groupId,omitempty"`
	SASLMechanism string `json:"saslMechanism,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
}

func (c *KafkaConfig) Normalize() {
	c.SASLMechanism = strings.ToLower(strings.TrimSpace(c.SASLMechanism))
	if c.SASLMechanism == "" {
		if strings.TrimSpace(c.Username) != "" || strings.TrimSpace(c.Password) != "" {
			c.SASLMechanism = KafkaSASLPlain
		} else {
			c.SASLMechanism = KafkaSASLNone
		}
	}
	if strings.TrimSpace(c.ClientID) == "" {
		c.ClientID = randomKafkaID()
	}
	if strings.TrimSpace(c.GroupID) == "" {
		c.GroupID = randomKafkaID()
	}
}

func randomKafkaID() string {
	buffer := make([]byte, kafkaIDLength)
	if _, err := rand.Read(buffer); err != nil {
		return kafkaIDPrefix + "aaaaaa"
	}
	out := make([]byte, kafkaIDLength)
	for index, value := range buffer {
		out[index] = kafkaIDLetters[int(value)%len(kafkaIDLetters)]
	}
	return kafkaIDPrefix + string(out)
}

func (c *KafkaConfig) brokers() []string {
	raw := strings.ReplaceAll(c.BrokerURL, ";", ",")
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		broker := strings.TrimSpace(part)
		broker = strings.TrimPrefix(broker, "kafka://")
		broker = strings.TrimPrefix(broker, "tcp://")
		if broker != "" {
			out = append(out, broker)
		}
	}
	return out
}

func (c *KafkaConfig) Validate() error {
	brokers := c.brokers()
	if len(brokers) == 0 {
		return fmt.Errorf("brokerUrl is required")
	}
	for _, broker := range brokers {
		if _, _, err := net.SplitHostPort(broker); err != nil {
			return fmt.Errorf("broker %q must use host:port format", broker)
		}
	}
	switch c.SASLMechanism {
	case KafkaSASLNone, KafkaSASLPlain, KafkaSASLScramSHA256, KafkaSASLScramSHA512:
	default:
		return fmt.Errorf("saslMechanism must be none, plain, scram-sha-256 or scram-sha-512")
	}
	if strings.TrimSpace(c.ClientID) == "" {
		return fmt.Errorf("clientId is required")
	}
	if strings.TrimSpace(c.GroupID) == "" {
		return fmt.Errorf("groupId is required")
	}
	return nil
}

func (c *KafkaConfig) saslMechanism() (sasl.Mechanism, error) {
	switch c.SASLMechanism {
	case KafkaSASLNone:
		return nil, nil
	case KafkaSASLPlain:
		return plain.Mechanism{Username: c.Username, Password: c.Password}, nil
	case KafkaSASLScramSHA256:
		return scram.Mechanism(scram.SHA256, c.Username, c.Password)
	case KafkaSASLScramSHA512:
		return scram.Mechanism(scram.SHA512, c.Username, c.Password)
	default:
		return nil, fmt.Errorf("unsupported sasl mechanism %q", c.SASLMechanism)
	}
}

func (c *KafkaConfig) dialer(mechanism sasl.Mechanism) *kafka.Dialer {
	return &kafka.Dialer{
		ClientID:      c.ClientID,
		Timeout:       kafkaConnectTimeout,
		DualStack:     true,
		SASLMechanism: mechanism,
	}
}

type kafkaReader struct {
	reader *kafka.Reader
	cancel context.CancelFunc
	done   chan struct{}
}

var _ Connector = (*KafkaConnector)(nil)

type KafkaConnector struct {
	id      string
	config  KafkaConfig
	writer  *kafka.Writer
	readers map[string]*kafkaReader
	mu      sync.Mutex
}

func NewKafkaConnector(id string, raw json.RawMessage) (*KafkaConnector, error) {
	var config KafkaConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("invalid kafka pushConfig: %w", err)
	}
	config.Normalize()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &KafkaConnector{id: id, config: config, readers: make(map[string]*kafkaReader)}, nil
}

func (c *KafkaConnector) Type() string { return ConnectorTypeKafka }

func (c *KafkaConnector) Connect(ctx context.Context) error {
	mechanism, err := c.config.saslMechanism()
	if err != nil {
		return err
	}
	dialer := c.config.dialer(mechanism)
	var lastErr error
	for _, broker := range c.config.brokers() {
		conn, err := dialer.DialContext(ctx, "tcp", broker)
		if err != nil {
			lastErr = err
			continue
		}
		_ = conn.Close()
		lastErr = nil
		break
	}
	if lastErr != nil {
		return fmt.Errorf("kafka broker is unreachable: %w", lastErr)
	}
	transport := &kafka.Transport{
		ClientID: c.config.ClientID,
		SASL:     mechanism,
	}
	c.mu.Lock()
	c.writer = &kafka.Writer{
		Addr:         kafka.TCP(c.config.brokers()...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Transport:    transport,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	c.mu.Unlock()
	return nil
}

func (c *KafkaConnector) Close(ctx context.Context) error {
	c.mu.Lock()
	readers := c.readers
	c.readers = make(map[string]*kafkaReader)
	writer := c.writer
	c.writer = nil
	c.mu.Unlock()

	for _, reader := range readers {
		closeKafkaReader(reader)
	}
	if writer != nil {
		_ = writer.Close()
	}
	return nil
}

func (c *KafkaConnector) Publish(ctx context.Context, message Message) error {
	c.mu.Lock()
	writer := c.writer
	c.mu.Unlock()
	if writer == nil {
		return fmt.Errorf("kafka connector is not connected")
	}
	if strings.TrimSpace(message.Topic) == "" {
		return fmt.Errorf("kafka topic is required")
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Topic: message.Topic,
		Key:   []byte(message.Key),
		Value: message.Payload,
	})
}

func (c *KafkaConnector) Subscribe(ctx context.Context, topic string, handler CommandHandler) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil
	}
	c.mu.Lock()
	if _, exists := c.readers[topic]; exists {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	mechanism, err := c.config.saslMechanism()
	if err != nil {
		return err
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.config.brokers(),
		GroupID:  c.config.GroupID,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  time.Second,
		Dialer:   c.config.dialer(mechanism),
	})
	loopCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	c.mu.Lock()
	if _, exists := c.readers[topic]; exists {
		c.mu.Unlock()
		cancel()
		_ = reader.Close()
		return nil
	}
	c.readers[topic] = &kafkaReader{reader: reader, cancel: cancel, done: done}
	c.mu.Unlock()
	go func() {
		defer close(done)
		for {
			message, err := reader.ReadMessage(loopCtx)
			if err != nil {
				if loopCtx.Err() != nil {
					return
				}
				time.Sleep(250 * time.Millisecond)
				continue
			}
			if handler != nil {
				handler(context.Background(), append([]byte(nil), message.Value...))
			}
		}
	}()
	return nil
}

func (c *KafkaConnector) Unsubscribe(ctx context.Context, topic string) error {
	topic = strings.TrimSpace(topic)
	c.mu.Lock()
	reader := c.readers[topic]
	delete(c.readers, topic)
	c.mu.Unlock()
	if reader != nil {
		closeKafkaReader(reader)
	}
	return nil
}

func closeKafkaReader(reader *kafkaReader) {
	if reader == nil {
		return
	}
	reader.cancel()
	select {
	case <-reader.done:
	case <-time.After(kafkaCloseTimeout):
	}
	_ = reader.reader.Close()
}

type KafkaFactory struct{}

var _ ConnectorFactory = (*KafkaFactory)(nil)
var _ ConnectorConfigNormalizer = (*KafkaFactory)(nil)

func NewKafkaFactory() *KafkaFactory { return &KafkaFactory{} }

func (f *KafkaFactory) Type() string { return ConnectorTypeKafka }

func (f *KafkaFactory) Normalize(_ string, raw json.RawMessage) (json.RawMessage, error) {
	var config KafkaConfig
	if len(raw) == 0 || json.Unmarshal(raw, &config) != nil {
		return nil, fmt.Errorf("invalid kafka pushConfig")
	}
	config.Normalize()
	return json.Marshal(config)
}

func (f *KafkaFactory) Validate(raw json.RawMessage) error {
	var config KafkaConfig
	if len(raw) == 0 || json.Unmarshal(raw, &config) != nil {
		return fmt.Errorf("invalid kafka pushConfig")
	}
	config.Normalize()
	return config.Validate()
}

func (f *KafkaFactory) New(id string, config json.RawMessage) (Connector, error) {
	return NewKafkaConnector(id, config)
}
