package things

import (
	"context"

	"harns-platform/pkg/logging"
)

// PropertyProcessor handles decoded property reports.
type PropertyProcessor interface {
	ProcessProperty(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error
}

// LoggingPropertyProcessor is a placeholder that only logs the report.
type LoggingPropertyProcessor struct{}

func (p *LoggingPropertyProcessor) ProcessProperty(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error {
	logging.V(2).InfoS("Received thing property report",
		"deviceKey", msg.DeviceKey, "topic", msg.Topic, "messageId", msg.MessageID,
		"source", msg.Source, "items", len(report.Payload))
	return nil
}
