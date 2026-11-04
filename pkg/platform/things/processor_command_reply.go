package things

import (
	"context"

	"harns-platform/pkg/logging"
)

// CommandReplyProcessor handles decoded command replies.
type CommandReplyProcessor interface {
	ProcessCommandReply(ctx context.Context, msg *DeviceMessage, reply *CommandReply) error
}

// LoggingCommandReplyProcessor is a placeholder that only logs the reply.
type LoggingCommandReplyProcessor struct{}

func (p *LoggingCommandReplyProcessor) ProcessCommandReply(ctx context.Context, msg *DeviceMessage, reply *CommandReply) error {
	logging.V(2).InfoS("Received thing command reply",
		"deviceKey", msg.DeviceKey, "topic", msg.Topic, "messageId", msg.MessageID,
		"source", msg.Source, "items", len(reply.Payload))
	return nil
}
