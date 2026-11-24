package focas

import (
	"context"
	"fmt"
	"sync"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type tagRef struct {
	variable *collection.Variable
	address  Address
}

type collectorRuntime struct {
	mu      sync.Mutex
	tags    []*tagRef
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func (c *Collection) Connect(ctx context.Context) error {
	_ = ctx
	if c == nil {
		return fmt.Errorf("FOCAS collector is nil")
	}
	if c.Host == "" || c.Port < 1 || c.Port > 65535 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: FOCAS host/port is invalid", collection.ErrInvalidConfig)
	}
	for index, variable := range c.GetCollectionMeta().GetVariables() {
		if variable == nil || variable.IsCalculated {
			continue
		}
		if _, err := ParseAddress(variable.Address, variable.DataType); err != nil {
			c.SetCollectStatus(collection.CollectStatusError)
			return fmt.Errorf("%w: variables[%d]: %v", collection.ErrInvalidConfig, index, err)
		}
	}
	library, err := findLibrary()
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return fmt.Errorf("%w: put libfocas32.so.1 in plugins/dependencies/focas/", err)
	}
	c.SetCollectStatus(collection.CollectStatusUnconnected)
	return fmt.Errorf("%w: library %s", ErrLibraryNotWired, library)
}

func (c *Collection) Start(ctx context.Context) error {
	runtime := c.runtime
	if runtime == nil {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	runtime.mu.Lock()
	if runtime.started {
		runtime.mu.Unlock()
		return nil
	}
	runtime.started = true
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.wg.Add(1)
	runtime.mu.Unlock()
	go c.loop(runCtx)
	return nil
}

func (c *Collection) Stop(ctx context.Context) error {
	runtime := c.runtime
	if runtime == nil {
		c.SetCollectStatus(collection.CollectStatusStopped)
		return nil
	}
	runtime.mu.Lock()
	cancel := runtime.cancel
	runtime.cancel = nil
	runtime.started = false
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) loop(ctx context.Context) {
	runtime := c.runtime
	if runtime == nil {
		return
	}
	defer runtime.wg.Done()
	<-ctx.Done()
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
}

var _ collection.Collector = (*Collection)(nil)
