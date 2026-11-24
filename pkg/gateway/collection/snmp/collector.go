package snmp

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
)

type tagRef struct {
	variable *collection.Variable
	address  Address
}
type collectorRuntime struct {
	mu        sync.Mutex
	tags      []*tagRef
	calcs     []*collection.Calculation
	calcErr   error
	pool      *connpool.Pool
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	started   bool
	last      time.Time
	host      string
	port      int
	localPort int
	timeout   time.Duration
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("SNMP collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Host) == "" || c.Port < 1 || c.Port > 65535 || c.LocalPort < 0 || c.LocalPort > 65535 {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: SNMP endpoint is invalid", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	tags := make([]*tagRef, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode != collection.AccessModeRead {
			continue
		}
		address, err := ParseAddress(variable.Address)
		if err != nil {
			c.SetCollectStatus(collection.CollectStatusError)
			return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
		}
		tags = append(tags, &tagRef{variable: variable, address: address})
	}
	calculations, calcErr := collection.CompileCalculations(variables)
	runtime := &collectorRuntime{
		tags: tags, calcs: calculations, calcErr: calcErr,
		host: c.Host, port: c.Port, localPort: c.LocalPort, timeout: c.timeout(),
	}
	c.runtime = runtime
	if len(tags) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		return openClient(ctx, runtime.host, runtime.port, runtime.localPort, runtime.timeout)
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
	if runtime.pool == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("SNMP runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
	runtime.last = time.Time{}
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
			if !runtime.startedAt(now, cycle) {
				continue
			}
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
			if runtime.calcErr != nil {
				c.GetCollectionMeta().ReportError(runtime.calcErr)
				c.SetCollectStatus(collection.CollectStatusCollectingError)
				continue
			}
			if err := collection.Calculate(c.GetCollectionMeta(), runtime.calcs); err != nil {
				c.GetCollectionMeta().ReportError(err)
				c.SetCollectStatus(collection.CollectStatusCollectingError)
				continue
			}
			c.GetCollectionMeta().ReportError(nil)
		}
	}
}

func (r *collectorRuntime) startedAt(now time.Time, cycle time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.last.IsZero() && now.Sub(r.last) < cycle {
		return false
	}
	r.last = now
	return true
}

func (c *Collection) poll(ctx context.Context, runtime *collectorRuntime) error {
	if runtime.pool == nil {
		return collection.ConnectionLost(fmt.Errorf("SNMP runtime is not connected"))
	}
	succeeded := 0
	connectionFailures := 0
	protocolFailure := false
	var cycleErr error
	for index, tag := range runtime.tags {
		if index > 0 && c.IntervalMs > 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(c.IntervalMs) * time.Millisecond):
			}
		}
		poolClient, err := runtime.pool.Get(ctx)
		if err != nil {
			return collection.ConnectionLost(err)
		}
		conn, ok := poolClient.(*client)
		if !ok {
			runtime.pool.Put(poolClient, true)
			return collection.ConnectionLost(fmt.Errorf("invalid SNMP client type"))
		}
		value, err := conn.Read(tag.address)
		runtime.pool.Put(poolClient, err != nil && isTransportError(err))
		if err != nil {
			if cycleErr == nil {
				cycleErr = err
			}
			if isTransportError(err) {
				connectionFailures++
			} else {
				protocolFailure = true
			}
			continue
		}
		succeeded++
		c.GetCollectionMeta().SetVariableValue(tag.variable.Name, collection.ApplyTransform(value, tag.variable))
	}
	if ctx.Err() != nil {
		return nil
	}
	if cycleErr != nil {
		c.GetCollectionMeta().ReportError(cycleErr)
		if !collection.ApplyCycleFailure(c, cycleErr, succeeded, connectionFailures, protocolFailure) {
			return collection.ConnectionLost(cycleErr)
		}
		return cycleErr
	}
	return nil
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
	return strings.Contains(message, "timeout") || strings.Contains(message, "connection") || strings.Contains(message, "closed") || strings.Contains(message, "refused") || strings.Contains(message, "unreachable") || strings.Contains(message, "eof")
}

var _ collection.Collector = (*Collection)(nil)
