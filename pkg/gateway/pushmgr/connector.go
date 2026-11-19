package pushmgr

import (
	"context"
	"encoding/json"
	"errors"
)

var (
	ErrConnectorNotFound        = errors.New("push connector not found")
	ErrConnectorInUse           = errors.New("push connector is in use")
	ErrGroupNotFound            = errors.New("push group not found")
	ErrConnectorAlreadyRunning  = errors.New("push connector is already running")
	ErrConnectorAlreadyStopped  = errors.New("push connector is already stopped")
	ErrConnectorStartFailed     = errors.New("push connector start failed")
	ErrConnectorStopFailed      = errors.New("push connector stop failed")
	ErrGroupAlreadyRunning      = errors.New("push group is already running")
	ErrGroupAlreadyStopped      = errors.New("push group is already stopped")
	ErrGroupDisabled            = errors.New("push group is disabled")
	ErrGroupConnectorNotRunning = errors.New("push connector is not running")
)

type ConnectorStartError struct {
	Reason string
}

func (e *ConnectorStartError) Error() string { return "push connector start failed: " + e.Reason }
func (e *ConnectorStartError) Unwrap() error { return ErrConnectorStartFailed }

type ConnectorStopError struct {
	Reason string
}

func (e *ConnectorStopError) Error() string { return "push connector stop failed: " + e.Reason }
func (e *ConnectorStopError) Unwrap() error { return ErrConnectorStopFailed }

// CommandHandler handles one raw command message from a connector.
type CommandHandler func(ctx context.Context, payload []byte)

type Message struct {
	Topic   string
	Key     string
	Payload []byte
}

// Connector is one remote message-center connection.
type Connector interface {
	Type() string
	Connect(ctx context.Context) error
	Close(ctx context.Context) error
	Publish(ctx context.Context, message Message) error
	Subscribe(ctx context.Context, topic string, handler CommandHandler) error
	Unsubscribe(ctx context.Context, topic string) error
}

// ConnectorFactory creates a connector from persisted push config.
type ConnectorFactory interface {
	Type() string
	Validate(config json.RawMessage) error
	New(id string, config json.RawMessage) (Connector, error)
}

// ConnectorConfigNormalizer fills connector defaults before persistence.
type ConnectorConfigNormalizer interface {
	Normalize(name string, config json.RawMessage) (json.RawMessage, error)
}

type ConnectorErrorHandler interface {
	SetErrorHandler(handler func(error))
}
