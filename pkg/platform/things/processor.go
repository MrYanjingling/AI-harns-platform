package things

import (
	"context"

	"harns-platform/pkg/logging"
)

// MessageHandler processes one kind of platform message.
type MessageHandler interface {
	Handle(ctx context.Context, msg *DeviceMessage) error
}

// MessageRouter looks up the handler by message kind.
type MessageRouter struct {
	handlers map[MessageKind]MessageHandler
}

// NewMessageRouter builds a router from a kind-to-handler map.
func NewMessageRouter(handlers map[MessageKind]MessageHandler) *MessageRouter {
	copied := make(map[MessageKind]MessageHandler, len(handlers))
	for kind, handler := range handlers {
		if handler != nil {
			copied[kind] = handler
		}
	}
	return &MessageRouter{handlers: copied}
}

// Handle routes a message to the matching handler. Unknown kinds are logged
// and ignored.
func (r *MessageRouter) Handle(ctx context.Context, msg *DeviceMessage) error {
	if r == nil || msg == nil {
		return nil
	}
	handler, ok := r.handlers[msg.Kind]
	if !ok {
		logging.V(2).InfoS("No handler for thing message",
			"kind", msg.Kind, "deviceKey", msg.DeviceKey, "messageId", msg.MessageID)
		return nil
	}
	return handler.Handle(ctx, msg)
}

// NewDefaultMessageRouter wires the logging processors. It is used by tests
// and as a safe default when no business processors are configured.
func NewDefaultMessageRouter() *MessageRouter {
	return NewMessageRouter(map[MessageKind]MessageHandler{
		MessageKindProperty:     NewPropertyHandler(&LoggingPropertyProcessor{}),
		MessageKindEvent:        NewEventHandler(&LoggingEventProcessor{}),
		MessageKindCommandReply: NewCommandReplyHandler(&LoggingCommandReplyProcessor{}),
	})
}
