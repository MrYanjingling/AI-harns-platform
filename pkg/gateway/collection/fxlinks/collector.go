package fxlinks

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/collection/meterlink"
	"harns-platform/pkg/gateway/validation"
)

type linkClient struct {
	link    *meterlink.Client
	timeout time.Duration
	mu      sync.Mutex
}

func (c *linkClient) Available() bool { return c != nil && c.link != nil && c.link.Available() }
func (c *linkClient) Close() error {
	if c == nil || c.link == nil {
		return nil
	}
	return c.link.Close()
}

func (c *linkClient) exchangeRead(req []byte, useSum bool, format int) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if _, err := c.link.Write(req); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	var out []byte
	var one [1]byte
	for len(out) < 512 {
		if _, err := io.ReadFull(c.link, one[:]); err != nil {
			return nil, collection.ConnectionLost(err)
		}
		out = append(out, one[0])
		if out[0] == ctrlNAK && len(out) >= 7 {
			if format == 4 {
				tail := make([]byte, 2)
				_, _ = io.ReadFull(c.link, tail)
				out = append(out, tail...)
			}
			return out, nil
		}
		if one[0] == ctrlETX {
			extra := 0
			if useSum {
				extra += 2
			}
			if format == 4 {
				extra += 2
			}
			if extra > 0 {
				tail := make([]byte, extra)
				if _, err := io.ReadFull(c.link, tail); err != nil {
					return nil, collection.ConnectionLost(err)
				}
				out = append(out, tail...)
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("FX-Links read response exceeded 512 bytes")
}

func (c *linkClient) exchangeWrite(req []byte, format int) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if _, err := c.link.Write(req); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	first := make([]byte, 5)
	if _, err := io.ReadFull(c.link, first); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	out := first
	if first[0] == ctrlNAK {
		errCode := make([]byte, 2)
		if _, err := io.ReadFull(c.link, errCode); err != nil {
			return nil, collection.ConnectionLost(err)
		}
		out = append(out, errCode...)
	}
	if format == 4 {
		crlf := make([]byte, 2)
		if _, err := io.ReadFull(c.link, crlf); err != nil {
			return nil, collection.ConnectionLost(err)
		}
		out = append(out, crlf...)
	}
	return out, nil
}

type collectorRuntime struct {
	mu      sync.Mutex
	batches []*readBatch
	calcs   []*collection.Calculation
	calcErr error
	pool    *connpool.Pool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func (c *Collection) useSumCheck() bool { return c.SumCheck != 0 }

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("FX-Links collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Location) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: location is required", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	batches, err := buildReadBatches(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calcs, calcErr := collection.CompileCalculations(variables)
	rt := &collectorRuntime{batches: batches, calcs: calcs, calcErr: calcErr}
	c.runtime = rt
	if len(batches) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	timeout := time.Duration(c.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = time.Duration(DefaultTimeoutMs) * time.Millisecond
	}
	mode := meterlink.SerialMode(c.BaudRate, c.DataBits, c.Parity, c.StopBits)
	pool, err := connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		item, err := meterlink.Open(ctx, c.Link, c.Location, c.Port, timeout, mode)
		if err != nil {
			return nil, err
		}
		return &linkClient{link: meterlink.NewClient(item, timeout), timeout: timeout}, nil
	})
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	rt.pool = pool
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) Start(ctx context.Context) error {
	rt := c.runtime
	if rt == nil {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	rt.mu.Lock()
	if rt.started {
		rt.mu.Unlock()
		return nil
	}
	if len(rt.batches) == 0 {
		rt.mu.Unlock()
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	if rt.pool == nil {
		rt.mu.Unlock()
		return fmt.Errorf("FX-Links runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	rt.cancel = cancel
	rt.started = true
	rt.wg.Add(1)
	rt.mu.Unlock()
	go c.loop(runCtx)
	return nil
}

func (c *Collection) Stop(ctx context.Context) error {
	rt := c.runtime
	if rt == nil {
		c.SetCollectStatus(collection.CollectStatusStopped)
		return nil
	}
	rt.mu.Lock()
	cancel := rt.cancel
	rt.cancel = nil
	rt.started = false
	pool := rt.pool
	rt.pool = nil
	rt.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	rt.wg.Wait()
	if pool != nil {
		_ = pool.Close()
	}
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) loop(ctx context.Context) {
	rt := c.runtime
	if rt == nil {
		return
	}
	defer rt.wg.Done()
	cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
	if cycle <= 0 {
		cycle = time.Second
	}
	for {
		started := time.Now()
		if !c.poll(ctx) {
			return
		}
		delay := cycle - time.Since(started)
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (c *Collection) withClient(ctx context.Context, fn func(*linkClient) error) error {
	rt := c.runtime
	if rt == nil || rt.pool == nil {
		return collection.ConnectionLost(fmt.Errorf("FX-Links runtime is not connected"))
	}
	raw, err := rt.pool.Get(ctx)
	if err != nil {
		return collection.ConnectionLost(err)
	}
	cli, ok := raw.(*linkClient)
	if !ok {
		rt.pool.Put(raw, true)
		return collection.ConnectionLost(fmt.Errorf("invalid FX-Links client"))
	}
	err = fn(cli)
	rt.pool.Put(raw, err != nil && collection.IsConnectionLost(err))
	return err
}

func (c *Collection) poll(ctx context.Context) bool {
	rt := c.runtime
	if rt == nil || rt.pool == nil {
		return false
	}
	var lastErr error
	useSum := c.useSumCheck()
	for idx, batch := range rt.batches {
		if ctx.Err() != nil {
			return false
		}
		if idx > 0 && c.IntervalMs > 0 {
			time.Sleep(time.Duration(c.IntervalMs) * time.Millisecond)
		}
		req := buildReadRequest(c.Station, c.WaitTime, c.Format, useSum, batch.isBit, batch.device, batch.start, batch.count)
		var rawResp []byte
		err := c.withClient(ctx, func(cli *linkClient) error {
			resp, err := cli.exchangeRead(req, useSum, c.Format)
			if err != nil {
				return err
			}
			rawResp = resp
			return nil
		})
		if err != nil {
			if collection.IsConnectionLost(err) {
				c.GetCollectionMeta().ReportError(err)
				return collection.MarkConnectionLost(c, err)
			}
			lastErr = err
			continue
		}
		data, err := parseReadResponse(rawResp, useSum, batch.isBit, batch.count)
		if err != nil {
			lastErr = err
			continue
		}
		for _, p := range batch.points {
			val, err := decodePoint(p, data, batch.start)
			if err != nil {
				lastErr = err
				continue
			}
			c.GetCollectionMeta().SetVariableValue(p.variable.Name, collection.ApplyTransform(val, p.variable))
		}
	}
	if rt.calcErr != nil {
		lastErr = rt.calcErr
	} else if err := collection.Calculate(c.GetCollectionMeta(), rt.calcs); err != nil {
		lastErr = err
	}
	if lastErr != nil {
		c.GetCollectionMeta().ReportError(lastErr)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.GetCollectionMeta().ReportError(nil)
	c.SetCollectStatus(collection.CollectStatusCollecting)
	return true
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	rt := c.runtime
	if rt == nil || rt.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "actions", MsgID: "validate.action_collection_not_ready"}})
	}
	useSum := c.useSumCheck()
	for _, action := range actions {
		p, err := parseAddress(action.Variable)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.variable_address_invalid", Args: []interface{}{action.Variable.Address}}})
		}
		var req []byte
		if p.isBit {
			b, err := toBool(action.Value)
			if err != nil {
				return collection.DeviceActionError(action, err)
			}
			req = buildBitWriteRequest(c.Station, c.WaitTime, c.Format, useSum, p.device, p.number, []bool{b})
		} else if p.wordBit >= 0 {
			b, err := toBool(action.Value)
			if err != nil {
				return collection.DeviceActionError(action, err)
			}
			readReq := buildReadRequest(c.Station, c.WaitTime, c.Format, useSum, false, p.device, p.number, 1)
			var curResp []byte
			if err := c.withClient(ctx, func(cli *linkClient) error {
				resp, err := cli.exchangeRead(readReq, useSum, c.Format)
				if err != nil {
					return err
				}
				curResp = resp
				return nil
			}); err != nil {
				return collection.DeviceActionError(action, err)
			}
			curData, err := parseReadResponse(curResp, useSum, false, 1)
			if err != nil {
				return collection.DeviceActionError(action, err)
			}
			word := binary.LittleEndian.Uint16(curData[:2])
			if b {
				word |= 1 << uint(p.wordBit)
			} else {
				word &^= 1 << uint(p.wordBit)
			}
			req = buildWordWriteRequest(c.Station, c.WaitTime, c.Format, useSum, p.device, p.number, []uint16{word})
		} else {
			words, err := encodeWordValue(p.dataType, action.Value)
			if err != nil {
				return collection.DeviceActionError(action, err)
			}
			req = buildWordWriteRequest(c.Station, c.WaitTime, c.Format, useSum, p.device, p.number, words)
		}
		var writeResp []byte
		err = c.withClient(ctx, func(cli *linkClient) error {
			resp, err := cli.exchangeWrite(req, c.Format)
			if err != nil {
				return err
			}
			writeResp = resp
			return nil
		})
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		if err := parseWriteResponse(writeResp); err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

var _ collection.Collector = (*Collection)(nil)
