package updatemgr

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"harns-platform/pkg/logging"
)

const (
	mqttConnectTimeout = 5 * time.Second
	mqttPublishTimeout = 5 * time.Second
	mqttReconnectDelay = 5 * time.Second
	taskTopicPattern   = "harns/gateway/{gatewayId}/ota/task"
	statusTopicPattern = "harns/gateway/{gatewayId}/ota/status"
)

type mqttClient = mqtt.Client

func (s *Service) runMQTT() {
	delay := mqttReconnectDelay
	for {
		settings := s.settingsSnapshot()
		if !settings.Enabled || strings.TrimSpace(settings.BrokerURL) == "" {
			select {
			case <-s.stopCh:
				return
			case <-s.reloadCh:
				continue
			case <-time.After(mqttReconnectDelay):
				continue
			}
		}

		client, err := s.connectMQTT(settings)
		if err != nil {
			logging.ErrorS(err, "Failed to connect gateway OTA MQTT")
			select {
			case <-s.stopCh:
				return
			case <-s.reloadCh:
				continue
			case <-time.After(delay):
			}
			if delay < time.Minute {
				delay *= 2
				if delay > time.Minute {
					delay = time.Minute
				}
			}
			continue
		}
		delay = mqttReconnectDelay
		s.setMQTTClient(client)
		select {
		case <-s.stopCh:
			client.Disconnect(250)
			s.setMQTTClient(nil)
			return
		case <-s.reloadCh:
			client.Disconnect(250)
			s.setMQTTClient(nil)
			continue
		}
	}
}

func (s *Service) connectMQTT(settings Settings) (mqtt.Client, error) {
	options := mqtt.NewClientOptions()
	options.AddBroker(strings.TrimSpace(settings.BrokerURL))
	options.SetClientID(s.mqttClientID())
	options.SetUsername(settings.Username)
	options.SetPassword(settings.Password)
	options.SetKeepAlive(defaultMQTTKeepAlive * time.Second)
	options.SetAutoReconnect(true)
	options.SetCleanSession(false)
	options.SetConnectTimeout(mqttConnectTimeout)
	options.SetOnConnectHandler(func(client mqtt.Client) {
		s.setMQTTConnected(true)
		if err := s.subscribeTasks(client); err != nil {
			logging.ErrorS(err, "Failed to subscribe gateway OTA task topic")
		}
		s.publishLatestStatus()
	})
	options.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		s.setMQTTConnected(false)
		logging.ErrorS(err, "Gateway OTA MQTT connection lost")
	})
	client := mqtt.NewClient(options)
	s.setMQTTClient(client)
	token := client.Connect()
	if !token.WaitTimeout(mqttConnectTimeout) {
		s.setMQTTClient(nil)
		return nil, fmt.Errorf("gateway OTA MQTT connect timeout")
	}
	if err := token.Error(); err != nil {
		s.setMQTTClient(nil)
		return nil, fmt.Errorf("gateway OTA MQTT connect: %w", err)
	}
	return client, nil
}

func (s *Service) subscribeTasks(client mqtt.Client) error {
	topic := s.taskTopic()
	if topic == "" {
		return fmt.Errorf("gateway OTA task topic is empty")
	}
	token := client.Subscribe(topic, byte(defaultMQTTQoS), func(_ mqtt.Client, message mqtt.Message) {
		payload := append([]byte(nil), message.Payload()...)
		go s.handleTaskPayload(payload)
	})
	if !token.WaitTimeout(mqttConnectTimeout) {
		return fmt.Errorf("gateway OTA MQTT subscribe timeout")
	}
	return token.Error()
}

func (s *Service) handleTaskPayload(payload []byte) {
	var message TaskMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		logging.ErrorS(err, "Failed to decode gateway OTA task")
		return
	}
	if message.APIVersion != 1 || message.Type != "gateway.ota.task" {
		logging.Warnf("Ignoring gateway OTA message with type=%q apiVersion=%d", message.Type, message.APIVersion)
		return
	}
	s.acceptTask(message)
}

func (s *Service) publishStatus(task Task) {
	client := s.currentMQTTClient()
	if client == nil || !client.IsConnected() {
		return
	}
	message := StatusMessage{
		APIVersion:      1,
		Type:            "gateway.ota.status",
		TaskID:          task.TaskID,
		GatewayID:       s.opts.GatewayID,
		State:           task.State,
		Progress:        task.Progress,
		ReleaseVersion:  task.ReleaseVersion,
		BackendVersion:  task.BackendVersion,
		FrontendVersion: task.FrontendVersion,
		RestartRequired: task.RestartRequired,
		ErrorCode:       task.ErrorCode,
		Message:         task.Message,
		UpdatedAt:       task.UpdatedAt,
	}
	payload, err := json.Marshal(message)
	if err != nil {
		logging.ErrorS(err, "Failed to encode gateway OTA status")
		return
	}
	token := client.Publish(s.statusTopic(), byte(defaultMQTTQoS), false, payload)
	if !token.WaitTimeout(mqttPublishTimeout) {
		logging.Warnf("Gateway OTA status publish timeout: taskId=%s state=%s", task.TaskID, task.State)
		return
	}
	if err := token.Error(); err != nil {
		logging.ErrorS(err, "Failed to publish gateway OTA status", "taskId", task.TaskID, "state", task.State)
	}
}

func (s *Service) publishIgnored(taskID, message string) {
	if strings.TrimSpace(taskID) == "" {
		return
	}
	s.publishStatus(Task{
		TaskID:    taskID,
		State:     StateIgnored,
		ErrorCode: "ignored",
		Message:   message,
		UpdatedAt: time.Now().UTC(),
	})
}

func (s *Service) publishLatestStatus() {
	state := s.Snapshot()
	if state.Task == nil {
		return
	}
	s.publishStatus(*state.Task)
}

func (s *Service) setMQTTClient(client mqtt.Client) {
	s.mqttMu.Lock()
	s.mqttClient = client
	s.mqttMu.Unlock()
}

func (s *Service) currentMQTTClient() mqtt.Client {
	s.mqttMu.Lock()
	defer s.mqttMu.Unlock()
	return s.mqttClient
}

func (s *Service) setMQTTConnected(connected bool) {
	s.mqttMu.Lock()
	s.mqttConnected = connected
	s.mqttMu.Unlock()
}

func (s *Service) mqttClientID() string {
	return "harns-gateway-ota-" + s.opts.GatewayID
}

func (s *Service) taskTopic() string {
	return replaceGatewayID(taskTopicPattern, s.opts.GatewayID)
}

func (s *Service) statusTopic() string {
	return replaceGatewayID(statusTopicPattern, s.opts.GatewayID)
}

func replaceGatewayID(topic, gatewayID string) string {
	topic = strings.ReplaceAll(topic, "<gatewayId>", gatewayID)
	return strings.ReplaceAll(topic, "{gatewayId}", gatewayID)
}
