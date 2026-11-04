package things

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Inbound payload decoding errors.
var (
	ErrMissingMessageType        = errors.New("message type is required")
	ErrUnsupportedMessageType    = errors.New("unsupported message type")
	ErrOutboundMessageType       = errors.New("command is an outbound message type")
	ErrEmptyEventIdentify        = errors.New("eventIdentify must not be empty")
	ErrEmptyProperty             = errors.New("property must not be empty")
	ErrEmptyCommandProperty      = errors.New("command property must not be empty")
	ErrEmptyCommandReplyProperty = errors.New("command reply property must not be empty")
	ErrMissingMessageID          = errors.New("messageId is required")
	ErrMissingCommandResult      = errors.New("command reply result is required")
	ErrInvalidCommandTime        = errors.New("command reply time must be a positive millisecond timestamp")
)

// InboundMessage is implemented by every inbound report payload.
type InboundMessage interface {
	GetType() MessageKind
}

// PropertyReport is the batched property report payload.
type PropertyReport struct {
	Type    MessageKind          `json:"type"`
	Payload []PropertyReportItem `json:"payload"`
}

func (r *PropertyReport) GetType() MessageKind { return r.Type }

// PropertyReportItem is a single collected property sample.
type PropertyReportItem struct {
	CollectionID string                     `json:"collectionId"`
	Time         int64                      `json:"time"`
	Property     map[string]json.RawMessage `json:"property"`
}

// EventReport is the batched event report payload.
type EventReport struct {
	Type    MessageKind       `json:"type"`
	Payload []EventReportItem `json:"payload"`
}

func (r *EventReport) GetType() MessageKind { return r.Type }

// EventReportItem is a single event sample. EventIdentify keys are defined by
// the thing model event definition and are therefore not fixed here.
type EventReportItem struct {
	CollectionID  string                     `json:"collectionId"`
	Time          int64                      `json:"time"`
	EventIdentify map[string]json.RawMessage `json:"eventIdentify"`
}

// Command is an outbound command payload sent from the platform to a device.
type Command struct {
	Type    MessageKind   `json:"type"`
	Payload []CommandItem `json:"payload"`
}

func (c *Command) GetType() MessageKind { return c.Type }

// CommandItem is a single command.
type CommandItem struct {
	CollectionID string                     `json:"collectionId"`
	MessageID    string                     `json:"messageId"`
	Property     map[string]json.RawMessage `json:"property"`
}

// CommandReply is the batched command execution result sent by a device.
type CommandReply struct {
	Type    MessageKind        `json:"type"`
	Payload []CommandReplyItem `json:"payload"`
}

func (r *CommandReply) GetType() MessageKind { return r.Type }

// CommandReplyItem is a single command execution result.
type CommandReplyItem struct {
	CollectionID string                     `json:"collectionId"`
	MessageID    string                     `json:"messageId"`
	Property     map[string]json.RawMessage `json:"property"`
	Result       *bool                      `json:"result"`
	Time         int64                      `json:"time"`
}

type rawEnvelope struct {
	Type    MessageKind     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// DecodeInbound decodes an inbound payload by its type field.
func DecodeInbound(data []byte) (InboundMessage, error) {
	var env rawEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	switch env.Type {
	case "":
		return nil, ErrMissingMessageType
	case MessageKindProperty:
		var items []PropertyReportItem
		if err := json.Unmarshal(env.Payload, &items); err != nil {
			return nil, err
		}
		for i := range items {
			if len(items[i].Property) == 0 {
				return nil, fmt.Errorf("%w: index %d", ErrEmptyProperty, i)
			}
		}
		return &PropertyReport{Type: MessageKindProperty, Payload: items}, nil
	case MessageKindEvent:
		var items []EventReportItem
		if err := json.Unmarshal(env.Payload, &items); err != nil {
			return nil, err
		}
		for i := range items {
			if len(items[i].EventIdentify) == 0 {
				return nil, fmt.Errorf("%w: index %d", ErrEmptyEventIdentify, i)
			}
		}
		return &EventReport{Type: MessageKindEvent, Payload: items}, nil
	case MessageKindCommandReply:
		items, err := decodeCommandReplyItems(env.Payload)
		if err != nil {
			return nil, err
		}
		return &CommandReply{Type: MessageKindCommandReply, Payload: items}, nil
	case MessageKindCommand:
		return nil, ErrOutboundMessageType
	default:
		return nil, ErrUnsupportedMessageType
	}
}

// DecodeCommand decodes and validates an outbound command payload.
func DecodeCommand(data []byte) (*Command, error) {
	var env rawEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	if env.Type != MessageKindCommand {
		return nil, ErrUnsupportedMessageType
	}
	var items []CommandItem
	if err := json.Unmarshal(env.Payload, &items); err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].MessageID == "" {
			return nil, fmt.Errorf("%w: index %d", ErrMissingMessageID, i)
		}
		if len(items[i].Property) == 0 {
			return nil, fmt.Errorf("%w: index %d", ErrEmptyCommandProperty, i)
		}
	}
	return &Command{Type: MessageKindCommand, Payload: items}, nil
}

// DecodeCommandReply decodes a command reply payload.
func DecodeCommandReply(data []byte) (*CommandReply, error) {
	msg, err := DecodeInbound(data)
	if err != nil {
		return nil, err
	}
	reply, ok := msg.(*CommandReply)
	if !ok {
		return nil, ErrUnsupportedMessageType
	}
	return reply, nil
}

func decodeCommandReplyItems(payload json.RawMessage) ([]CommandReplyItem, error) {
	var items []CommandReplyItem
	if err := json.Unmarshal(payload, &items); err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].MessageID == "" {
			return nil, fmt.Errorf("%w: index %d", ErrMissingMessageID, i)
		}
		if len(items[i].Property) == 0 {
			return nil, fmt.Errorf("%w: index %d", ErrEmptyCommandReplyProperty, i)
		}
		if items[i].Result == nil {
			return nil, fmt.Errorf("%w: index %d", ErrMissingCommandResult, i)
		}
		if items[i].Time <= 0 {
			return nil, fmt.Errorf("%w: index %d", ErrInvalidCommandTime, i)
		}
	}
	return items, nil
}

// DecodeEventReport decodes an event report payload.
func DecodeEventReport(data []byte) (*EventReport, error) {
	msg, err := DecodeInbound(data)
	if err != nil {
		return nil, err
	}
	report, ok := msg.(*EventReport)
	if !ok {
		return nil, ErrUnsupportedMessageType
	}
	return report, nil
}

// DecodePropertyReport decodes a property report payload.
func DecodePropertyReport(data []byte) (*PropertyReport, error) {
	msg, err := DecodeInbound(data)
	if err != nil {
		return nil, err
	}
	report, ok := msg.(*PropertyReport)
	if !ok {
		return nil, ErrUnsupportedMessageType
	}
	return report, nil
}
