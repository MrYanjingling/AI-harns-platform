package things

import (
	"context"

	"harns-platform/pkg/logging"
)

// EventProcessor handles decoded event reports.
type EventProcessor interface {
	ProcessEvent(ctx context.Context, msg *DeviceMessage, report *EventReport) error
}

// LoggingEventProcessor is a placeholder that only logs the report.
type LoggingEventProcessor struct{}

func (p *LoggingEventProcessor) ProcessEvent(ctx context.Context, msg *DeviceMessage, report *EventReport) error {
	logging.V(2).InfoS("Received thing event report",
		"deviceKey", msg.DeviceKey, "topic", msg.Topic, "messageId", msg.MessageID,
		"source", msg.Source, "items", len(report.Payload))
	return nil
}
