package things

import (
	"context"
	"errors"
	"strings"
	"time"

	"harns-platform/pkg/logging"
	"harns-platform/pkg/timeseries"
)

// PropertyHandler decodes a property report and delegates it to the business processor.
type PropertyHandler struct {
	processor PropertyProcessor
	logWriter PropertyLogWriter
}

func NewPropertyHandler(processor PropertyProcessor) *PropertyHandler {
	return &PropertyHandler{processor: processor}
}

func NewPropertyHandlerWithLogWriter(processor PropertyProcessor, writer PropertyLogWriter) *PropertyHandler {
	return &PropertyHandler{processor: processor, logWriter: writer}
}

func (h *PropertyHandler) Handle(ctx context.Context, msg *DeviceMessage) error {
	report, err := DecodePropertyReport([]byte(msg.Payload))
	if err != nil {
		logging.V(2).InfoS("Failed to decode property report",
			"deviceKey", msg.DeviceKey, "messageId", msg.MessageID, "err", err)
		h.writeFailureLog(ctx, msg, err)
		return nil
	}
	if h.processor == nil {
		return nil
	}
	return h.processor.ProcessProperty(ctx, msg, report)
}

func (h *PropertyHandler) writeFailureLog(ctx context.Context, msg *DeviceMessage, cause error) {
	if msg == nil || h.logWriter == nil {
		return
	}
	modelKey := strings.TrimSpace(msg.ThingModelKey)
	if modelKey == "" {
		logging.V(2).InfoS("Cannot write property failure log without thing model key",
			"deviceKey", msg.DeviceKey, "messageId", msg.MessageID)
		return
	}
	log := timeseries.PropertyLog{
		ModelID:   msg.ThingModelID,
		DeviceID:  msg.ThingID,
		DeviceKey: msg.DeviceKey,
		Time:      time.Now(),
		Success:   false,
		Payload:   msg.Payload,
		ErrorMsg:  cause.Error(),
	}
	if err := h.logWriter.WritePropertyLog(ctx, modelKey, log); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
		logging.ErrorS(err, "Failed to write property failure log",
			"deviceKey", msg.DeviceKey, "thingModelKey", modelKey)
	}
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
