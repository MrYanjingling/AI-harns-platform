package pushmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const mqttConnectTimeout = 5 * time.Second

type MQTTConfig struct {
	BrokerURL    string `json:"brokerUrl"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DataTopic    string `json:"dataTopic"`
	CommandTopic string `json:"commandTopic"`
	ClientID     string `json:"clientId,omitempty"`
	QoS          byte   `json:"qos,omitempty"`
	Retain       bool   `json:"retain,omitempty"`
	KeepAlive    int    `json:"keepAlive,omitempty"`
}

func (c *MQTTConfig) Normalize(connectorID string) {
	if c.ClientID == "" {
		c.ClientID = "gateway-push-" + connectorID
	}
	if c.QoS == 0 {
		c.QoS = DefaultMQTTQoS
	}
	if c.KeepAlive <= 0 {
		c.KeepAlive = DefaultMQTTKeepAlive
	}
}

func (c *MQTTConfig) Validate() error {
	if strings.TrimSpace(c.BrokerURL) == "" {
		return fmt.Errorf("brokerUrl is required")
	}
	if c.QoS > 2 {
		return fmt.Errorf("qos must be 0, 1 or 2")
	}
	return nil
}

var _ Connector = (*MQTTConnector)(nil)
var _ ConnectorErrorHandler = (*MQTTConnector)(nil)

type MQTTConnector struct {
	id         string
	config     MQTTConfig
	client     mqtt.Client
	mu         sync.RWMutex
	errHandler func(error)
	subs       map[string]struct{}
}

func (c *MQTTConnector) SetErrorHandler(handler func(error)) {
	c.mu.Lock()
	c.errHandler = handler
	c.mu.Unlock()
}

func (c *MQTTConnector) notifyError(err error) {
	c.mu.RLock()
	handler := c.errHandler
	c.mu.RUnlock()
	if handler != nil && err != nil {
		handler(err)
	}
}

func NewMQTTConnector(id string, raw json.RawMessage) (*MQTTConnector, error) {
	var config MQTTConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("invalid mqtt pushConfig: %w", err)
	}
	config.Normalize(id)
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &MQTTConnector{id: id, config: config, subs: make(map[string]struct{})}, nil
}

func (c *MQTTConnector) Type() string { return ConnectorTypeMQTT }

func (c *MQTTConnector) Connect(ctx context.Context) error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(c.config.BrokerURL)
	opts.SetClientID(c.config.ClientID)
	opts.SetUsername(c.config.Username)
	opts.SetPassword(c.config.Password)
	opts.SetKeepAlive(time.Duration(c.config.KeepAlive) * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectTimeout(mqttConnectTimeout)
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		c.notifyError(fmt.Errorf("mqtt connection lost: %w", err))
	})
	c.client = mqtt.NewClient(opts)
	token := c.client.Connect()
	if !token.WaitTimeout(mqttConnectTimeout) {
		return fmt.Errorf("mqtt connect timeout")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}
	return nil
}

func (c *MQTTConnector) Close(ctx context.Context) error {
	c.mu.Lock()
	c.subs = make(map[string]struct{})
	c.mu.Unlock()
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}
	return nil
}

func (c *MQTTConnector) Publish(ctx context.Context, message Message) error {
	if c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("mqtt connector is not connected")
	}
	if strings.TrimSpace(message.Topic) == "" {
		return fmt.Errorf("mqtt topic is required")
	}
	token := c.client.Publish(message.Topic, c.config.QoS, c.config.Retain, message.Payload)
	if !token.WaitTimeout(mqttConnectTimeout) {
		return fmt.Errorf("mqtt publish timeout")
	}
	return token.Error()
}

func (c *MQTTConnector) Subscribe(ctx context.Context, topic string, handler CommandHandler) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil
	}
	if c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("mqtt connector is not connected")
	}
	token := c.client.Subscribe(topic, c.config.QoS, func(_ mqtt.Client, message mqtt.Message) {
		if handler != nil {
			go handler(context.Background(), append([]byte(nil), message.Payload()...))
		}
	})
	if !token.WaitTimeout(mqttConnectTimeout) {
		return fmt.Errorf("mqtt subscribe timeout")
	}
	if err := token.Error(); err != nil {
		return err
	}
	c.mu.Lock()
	c.subs[topic] = struct{}{}
	c.mu.Unlock()
	return nil
}

func (c *MQTTConnector) Unsubscribe(ctx context.Context, topic string) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil
	}
	if c.client == nil || !c.client.IsConnected() {
		return nil
	}
	token := c.client.Unsubscribe(topic)
	if !token.WaitTimeout(mqttConnectTimeout) {
		return fmt.Errorf("mqtt unsubscribe timeout")
	}
	if err := token.Error(); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.subs, topic)
	c.mu.Unlock()
	return nil
}

type MQTTFactory struct{}

var _ ConnectorFactory = (*MQTTFactory)(nil)

func NewMQTTFactory() *MQTTFactory { return &MQTTFactory{} }

func (f *MQTTFactory) Type() string { return ConnectorTypeMQTT }

func (f *MQTTFactory) Validate(raw json.RawMessage) error {
	var config MQTTConfig
	if len(raw) == 0 || json.Unmarshal(raw, &config) != nil {
		return fmt.Errorf("invalid mqtt pushConfig")
	}
	config.Normalize("")
	return config.Validate()
}

func (f *MQTTFactory) New(id string, config json.RawMessage) (Connector, error) {
	return NewMQTTConnector(id, config)
}
