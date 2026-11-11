package collection

import (
	"context"
	"errors"
)

// ErrNotImplemented is returned by placeholder collectors.
var ErrNotImplemented = errors.New("collector is not implemented")

// ErrInvalidConfig reports a permanent collector configuration error.
var ErrInvalidConfig = errors.New("invalid collector config")

// Collector is the lifecycle interface implemented by collection drivers.
type Collector interface {
	GetCollectionMeta() *CollectionMeta
	SetCollectionMeta(*CollectionMeta)

	Connect(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	GetCollectStatus() string
	SetCollectStatus(string)
	IndexCollection()
}
