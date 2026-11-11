package modbustcp

import (
	"context"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
)

type collectorRuntime struct {
	mu sync.Mutex

	pool   *tcpPool
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
