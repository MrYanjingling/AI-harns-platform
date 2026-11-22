package pushmgr

import (
	"context"
	"time"
)

const (
	DefaultReadySendMaxMessages = 10000
	DefaultReadySendMaxBytes    = 64 << 20
	DefaultReadySendMaxAge      = 24 * time.Hour
)

// ReadySendStore is the persistence contract used by the push runtime.
type ReadySendStore interface {
	SaveReadySend(ctx context.Context, batch *ReadySendBatch) error
	ListReadySend(ctx context.Context, groupID string, limit int) ([]*ReadySendBatch, error)
	DeleteReadySend(ctx context.Context, groupID string, seq int64) error
	ReadySendStats(ctx context.Context, groupID string) (ReadySendStats, error)
}

// ReadySendBatch is one merged payload persisted for later delivery.
type ReadySendBatch struct {
	Seq         int64
	GroupID     string
	ConnectorID string
	Topic       string
	Payload     []byte
	PayloadSize int
	CreatedTime time.Time
}

// ReadySendStats is the lightweight aggregate used to seed runtime counters.
type ReadySendStats struct {
	Messages int64
	Batches  int64
	Oldest   time.Time
}

// SaveReadySend stores one merged batch and trims old/oversized rows.
