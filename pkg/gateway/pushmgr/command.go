package pushmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"time"

	"harns-platform/pkg/logging"
)

func commandValue(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), true
	case json.Number:
		return v.String(), true
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), true
	default:
		return "", false
	}
}

type commandMessage struct {
	Type    string        `json:"type"`
	Payload []commandItem `json:"payload"`
}

type commandItem struct {
	CollectionID string         `json:"collectionId"`
	MessageID    string         `json:"messageId"`
	Property     map[string]any `json:"property"`
}

func (m *RuntimeManager) groupCommandHandler(runtime *groupRuntime, connectorID, topic string) CommandHandler {
	return func(ctx context.Context, payload []byte) {
		connectorType := ""
		if connector := m.connectorRuntime(connectorID); connector != nil {
			connectorType = connector.connector.Type()
		}
		runtime.commands.Publish(PushCommandEvent{
			GroupID:       runtime.group.ID,
			ConnectorID:   connectorID,
			ConnectorType: connectorType,
			Topic:         topic,
			Time:          time.Now().UTC(),
			Payload:       normalizeCommandPayload(payload),
		})
		m.handleCommand(ctx, payload)
	}
}

func normalizeCommandPayload(payload []byte) json.RawMessage {
	if json.Valid(payload) {
		return append(json.RawMessage(nil), payload...)
	}
	encoded, err := json.Marshal(string(payload))
	if err != nil {
		return json.RawMessage(`null`)
	}
	return encoded
}

func (m *RuntimeManager) handleCommand(ctx context.Context, payload []byte) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var message commandMessage
	if err := decoder.Decode(&message); err != nil {
		logging.ErrorS(err, "Failed to decode push command")
		return
	}
	if message.Type != "command" {
		return
	}
	for _, item := range message.Payload {
		actions := make([]Action, 0, len(item.Property))
		for name, value := range item.Property {
			converted, ok := commandValue(value)
			if !ok {
				continue
			}
			actions = append(actions, Action{Name: name, Value: converted})
		}
		if len(actions) == 0 {
			continue
		}
		logging.InfoS("Push command received", "collectionId", item.CollectionID, "messageId", item.MessageID, "actions", len(actions))
		if err := m.source.DeliverActions(ctx, item.CollectionID, actions); err != nil {
			logging.ErrorS(err, "Failed to deliver push command", "collectionId", item.CollectionID, "messageId", item.MessageID)
		}
	}
}
