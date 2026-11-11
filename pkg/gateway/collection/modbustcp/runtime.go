package modbustcp

import (
	"context"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
)

type collectorRuntime struct {
	mu sync.Mutex

	pool   *connpool.Pool
	frames []*wireFrame
	calcs  []*collection.Calculation

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	tx      uint32

	failureStreak int
	backupSince   time.Time
	probeSuccess  int
	lastProbe     time.Time
}
