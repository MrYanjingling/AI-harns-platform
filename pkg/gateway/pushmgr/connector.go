package pushmgr

import (
	"context"
	"encoding/json"
)

// CommandHandler handles one raw command message from a connector.
type CommandHandler func(ctx context.Context, payload []byte)

// Connector is one remote message-center connection.
type Connector interface {
	Type() string
	Connect(ctx context.Context) error
	Close(ctx context.Context) error
	Publish(ctx context.Context, payload []byte) error
	SubscribeCommands(ctx context.Context, handler CommandHandler) error
}

// ConnectorFactory creates a connector from persisted push config.
type ConnectorFactory interface {
	Type() string
	New(id string, config json.RawMessage) (Connector, error)
}
