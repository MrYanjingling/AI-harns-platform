package bacnet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
)

type tagRef struct {
	variable *collection.Variable
	point    Point
}

type collectorRuntime struct {
	mu      sync.Mutex
	tags    []*tagRef
	pool    *connpool.Pool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	last    time.Time
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("BACnet collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Host) == "" || c.Port < 1 || c.Port > 65535 || c.SrcPort < 0 || c.SrcPort > 65535 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: BACnet endpoint is invalid", collection.ErrInvalidConfig)
	}
	tags := make([]*tagRef, 0)
	for _, variable := range c.GetCollectionMeta().GetVariables() {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		point, err := ParseAddress(variable.Address)
		if err != nil {
			c.SetCollectStatus(collection.CollectStatusError)
			return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
		}
		tags = append(tags, &tagRef{variable: variable, point: point})
	}
	var writable bool
	for _, variable := range c.GetCollectionMeta().GetVariables() {
		if variable != nil && !variable.IsCalculated && variable.AccessMode == collection.AccessModeWrite {
			writable = true
			break
		}
	}
	runtime := &collectorRuntime{tags: tags}
	c.runtime = runtime
	if len(tags) == 0 && !writable {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		item, err := openClient(ctx, c.Host, c.Port, c.SrcPort, c.BBMD == 1, c.timeout())
		if err != nil {
			return nil, collection.ConnectionLost(err)
		}
		return item, nil
	})
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.pool = pool
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
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
	runtime.last = time.Time{}
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
	pool := runtime.pool
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if pool != nil {
		_ = pool.Close()
	}
	runtime.mu.Lock()
	if runtime.pool == pool {
		runtime.pool = nil
	}
	runtime.mu.Unlock()
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) loop(ctx context.Context) {
	runtime := c.runtime
	if runtime == nil {
		return
	}
	defer runtime.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
			if cycle <= 0 {
				cycle = time.Second
			}
			if !runtime.last.IsZero() && now.Sub(runtime.last) < cycle {
				continue
			}
			runtime.last = now
			if err := c.poll(ctx, runtime); err != nil {
				if collection.IsConnectionLost(err) {
					c.GetCollectionMeta().ReportError(err)
					collection.MarkConnectionLost(c, err)
					return
				}
				c.GetCollectionMeta().ReportError(err)
				c.SetCollectStatus(collection.CollectStatusCollectingError)
				continue
			}
			c.SetCollectStatus(collection.CollectStatusCollecting)
			c.GetCollectionMeta().ReportError(nil)
		}
	}
}

func (c *Collection) poll(ctx context.Context, runtime *collectorRuntime) error {
	var firstErr error
	for index, tag := range runtime.tags {
		if index > 0 && c.IntervalMs > 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(c.IntervalMs) * time.Millisecond):
			}
		}
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.ConnectionLost(err)
		}
		conn, ok := client.(*bacClient)
		if !ok {
			runtime.pool.Put(client, true)
			return collection.ConnectionLost(fmt.Errorf("invalid BACnet client type"))
		}
		value, err := conn.Read(tag.point)
		runtime.pool.Put(client, err != nil && isTransportError(err))
		if err != nil {
			transportErr := collection.ConnectionLost(err)
			if collection.IsConnectionLost(transportErr) && isTransportError(err) {
				return transportErr
			}
			c.GetCollectionMeta().ReportError(err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		c.GetCollectionMeta().SetVariableValue(tag.variable.Name, collection.ApplyTransform(coerceValue(value, tag.variable.DataType), tag.variable))
	}
	return firstErr
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		if action.Variable == nil {
			continue
		}
		point, err := ParseAddress(action.Variable.Address)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: "address", MsgID: "validate.variable_address_invalid", Args: []interface{}{action.Variable.Address}}})
		}
		if point.Property != propertyPresentValue {
			return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_variable_not_writable", Args: []interface{}{action.Variable.Name}}})
		}
		client, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.DeviceActionError(action, collection.ConnectionLost(err))
		}
		conn, ok := client.(*bacClient)
		if !ok {
			runtime.pool.Put(client, true)
			return collection.DeviceActionError(action, collection.ConnectionLost(fmt.Errorf("invalid BACnet client type")))
		}
		err = conn.Write(point, action.Value, action.Variable.DataType)
		runtime.pool.Put(client, err != nil && isTransportError(err))
		if err != nil {
			if isTransportError(err) {
				return collection.DeviceActionError(action, collection.ConnectionLost(err))
			}
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func coerceValue(value any, dataType string) any {
	switch strings.ToLower(dataType) {
	case "bool":
		switch typed := value.(type) {
		case uint64:
			return typed != 0
		case uint32:
			return typed != 0
		case uint16:
			return typed != 0
		case uint8:
			return typed != 0
		case int64:
			return typed != 0
		case int32:
			return typed != 0
		case int16:
			return typed != 0
		case int8:
			return typed != 0
		case int:
			return typed != 0
		}
	case "bit":
		switch typed := value.(type) {
		case bool:
			if typed {
				return uint8(1)
			}
			return uint8(0)
		case uint64:
			return uint8(typed)
		case uint32:
			return uint8(typed)
		case uint16:
			return uint8(typed)
		case uint8:
			return typed
		case int64:
			return uint8(typed)
		case int32:
			return uint8(typed)
		case int16:
			return uint8(typed)
		case int8:
			return uint8(typed)
		case int:
			return uint8(typed)
		}
	}
	return value
}

func (c *Collection) timeout() time.Duration {
	if c.TimeoutMs <= 0 {
		return time.Duration(DefaultTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.TimeoutMs) * time.Millisecond
}

func isTransportError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "timeout") || strings.Contains(message, "connection") || strings.Contains(message, "closed") || strings.Contains(message, "refused") || strings.Contains(message, "unreachable")
}

var _ collection.Collector = (*Collection)(nil)
