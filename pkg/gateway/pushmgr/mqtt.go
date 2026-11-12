package pushmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	if strings.TrimSpace(c.DataTopic) == "" {
		return fmt.Errorf("dataTopic is required")
	}
	if c.QoS > 2 {
		return fmt.Errorf("qos must be 0, 1 or 2")
	}
	return nil
}

type MQTTConnector struct {
	id      string
	config  MQTTConfig
	client  mqtt.Client
	handler CommandHandler
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
	return &MQTTConnector{id: id, config: config}, nil
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
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}
	return nil
}

func (c *MQTTConnector) Publish(ctx context.Context, payload []byte) error {
	if c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("mqtt connector is not connected")
	}
	token := c.client.Publish(c.config.DataTopic, c.config.QoS, c.config.Retain, payload)
	if !token.WaitTimeout(mqttConnectTimeout) {
		return fmt.Errorf("mqtt publish timeout")
	}
	return token.Error()
}

func (c *MQTTConnector) SubscribeCommands(ctx context.Context, handler CommandHandler) error {
	if strings.TrimSpace(c.config.CommandTopic) == "" {
		return nil
	}
	if c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("mqtt connector is not connected")
	}
	c.handler = handler
	token := c.client.Subscribe(c.config.CommandTopic, c.config.QoS, func(_ mqtt.Client, message mqtt.Message) {
		if c.handler == nil {
			return
		}
		go c.handler(context.Background(), append([]byte(nil), message.Payload()...))
	})
	if !token.WaitTimeout(mqttConnectTimeout) {
		return fmt.Errorf("mqtt subscribe timeout")
	}
	return token.Error()
}

type MQTTFactory struct{}

func NewMQTTFactory() *MQTTFactory { return &MQTTFactory{} }

func (f *MQTTFactory) Type() string { return ConnectorTypeMQTT }

func (f *MQTTFactory) New(id string, config json.RawMessage) (Connector, error) {
	return NewMQTTConnector(id, config)
}
