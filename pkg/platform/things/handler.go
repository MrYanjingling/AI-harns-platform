package things

import (
	"context"

	"harns-platform/pkg/logging"
)

// PropertyHandler decodes a property report and delegates it to the business processor.
type PropertyHandler struct {
	processor PropertyProcessor
}

func NewPropertyHandler(processor PropertyProcessor) *PropertyHandler {
	return &PropertyHandler{processor: processor}
}

func (h *PropertyHandler) Handle(ctx context.Context, msg *DeviceMessage) error {
	report, err := DecodePropertyReport([]byte(msg.Payload))
	if err != nil {
		logging.V(2).InfoS("Failed to decode property report",
			"deviceKey", msg.DeviceKey, "messageId", msg.MessageID, "err", err)
		return nil
	}
	if h.processor == nil {
		return nil
	}
	return h.processor.ProcessProperty(ctx, msg, report)
}

// EventHandler decodes an event report and delegates it to the business processor.
type EventHandler struct {
	processor EventProcessor
}

func NewEventHandler(processor EventProcessor) *EventHandler {
	return &EventHandler{processor: processor}
}

func (h *EventHandler) Handle(ctx context.Context, msg *DeviceMessage) error {
	report, err := DecodeEventReport([]byte(msg.Payload))
	if err != nil {
		logging.V(2).InfoS("Failed to decode event report",
			"deviceKey", msg.DeviceKey, "messageId", msg.MessageID, "err", err)
		return nil
	}
	if h.processor == nil {
		return nil
	}
	return h.processor.ProcessEvent(ctx, msg, report)
}

// CommandReplyHandler decodes a command reply and delegates it to the business processor.
type CommandReplyHandler struct {
	processor CommandReplyProcessor
}

func NewCommandReplyHandler(processor CommandReplyProcessor) *CommandReplyHandler {
	return &CommandReplyHandler{processor: processor}
}

func (h *CommandReplyHandler) Handle(ctx context.Context, msg *DeviceMessage) error {
	reply, err := DecodeCommandReply([]byte(msg.Payload))
	if err != nil {
		logging.V(2).InfoS("Failed to decode command reply",
			"deviceKey", msg.DeviceKey, "messageId", msg.MessageID, "err", err)
		return nil
	}
	if h.processor == nil {
		return nil
	}
	return h.processor.ProcessCommandReply(ctx, msg, reply)
}
