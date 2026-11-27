package lsxgt

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
)

type tcpClient struct {
	conn    net.Conn
	reader  *bufio.Reader
	cnet    bool
	timeout time.Duration
	mu      sync.Mutex
}

func (c *tcpClient) Available() bool { return c != nil && c.conn != nil }

func (c *tcpClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *tcpClient) exchange(request []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("XGT connection is closed"))
	}
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if _, err := c.conn.Write(request); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if c.cnet {
		// Read until ETX (0x03), then drain optional 2-byte ASCII BCC if buffered.
		frame, err := c.reader.ReadBytes(0x03)
		if err != nil {
			return nil, collection.ConnectionLost(err)
		}
		if c.reader.Buffered() >= 2 {
			bcc := make([]byte, 2)
			if _, err := io.ReadFull(c.reader, bcc); err == nil {
				frame = append(frame, bcc...)
			}
		}
		return frame, nil
	}
	hdr := make([]byte, fenetHeaderSize)
	if _, err := io.ReadFull(c.reader, hdr); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	bodyLen := int(binary.LittleEndian.Uint16(hdr[16:18]))
	if bodyLen < 0 || bodyLen > 65535-fenetHeaderSize {
		return nil, fmt.Errorf("invalid FEnet response length %d", bodyLen)
	}
	resp := make([]byte, fenetHeaderSize+bodyLen)
	copy(resp, hdr)
	if bodyLen > 0 {
		if _, err := io.ReadFull(c.reader, resp[fenetHeaderSize:]); err != nil {
			return nil, collection.ConnectionLost(err)
		}
	}
	return resp, nil
}

type collectorRuntime struct {
	mu       sync.Mutex
	batches  []*readBatch
	calcs    []*collection.Calculation
	calcErr  error
	pool     *connpool.Pool
	invokeID uint16
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	started  bool
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("XGT collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	cfg := ProtocolConfig{Location: c.Location, Station: c.Station, TimeoutMs: c.TimeoutMs}
	cfg.Normalize()
	defaultPort := DefaultFEnetPort
	maxBytes := maxFEnetBatchBytes
	if c.cnet {
		defaultPort = DefaultCnetPort
		maxBytes = maxCnetBatchBytes
	}
	endpoint := cfg.endpoint(defaultPort)
	if endpoint == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: location is required", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	batches, err := buildReadBatches(variables, maxBytes)
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
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	pool, err := connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		dialer := net.Dialer{Timeout: timeout}
		conn, err := dialer.DialContext(ctx, "tcp", endpoint)
		if err != nil {
			return nil, err
		}
		return &tcpClient{conn: conn, reader: bufio.NewReader(conn), cnet: c.cnet, timeout: timeout}, nil
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
		return fmt.Errorf("XGT runtime is not connected")
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

func (c *Collection) nextInvokeID() uint16 {
	rt := c.runtime
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.invokeID++
	return rt.invokeID
}

func (c *Collection) poll(ctx context.Context) bool {
	rt := c.runtime
	if rt == nil || rt.pool == nil {
		return false
	}
	failed := false
	var cycleErr error
	for _, batch := range rt.batches {
		var req []byte
		if c.cnet {
			req = buildCnetReadRequest(c.Station, batch.area, batch.byteOffset, batch.byteCount)
		} else {
			req = buildFEnetReadRequest(c.nextInvokeID(), batch.area, batch.byteOffset, batch.byteCount)
		}
		resp, err := c.exchange(ctx, "read", req)
		if err != nil {
			if collection.IsConnectionLost(err) {
				return collection.MarkConnectionLost(c, err)
			}
			failed = true
			if cycleErr == nil {
				cycleErr = err
			}
			continue
		}
		var payload []byte
		if c.cnet {
			payload, err = parseCnetReadResponse(resp)
		} else {
			payload, err = parseFEnetReadResponse(resp)
		}
		if err != nil {
			failed = true
			if cycleErr == nil {
				cycleErr = err
			}
			continue
		}
		for _, item := range batch.items {
			val, err := decodeValue(item.addr, payload[item.offset:])
			if err != nil {
				failed = true
				if cycleErr == nil {
					cycleErr = err
				}
				continue
			}
			c.GetCollectionMeta().SetVariableValue(item.variable.Name, collection.ApplyTransform(val, item.variable))
		}
	}
	if ctx.Err() != nil {
		return false
	}
	if failed {
		c.GetCollectionMeta().ReportError(cycleErr)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.SetCollectStatus(collection.CollectStatusCollecting)
	if rt.calcErr != nil {
		c.GetCollectionMeta().ReportError(rt.calcErr)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	if err := collection.Calculate(c.GetCollectionMeta(), rt.calcs); err != nil {
		c.GetCollectionMeta().ReportError(err)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.GetCollectionMeta().ReportError(nil)
	return true
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	rt := c.runtime
	if rt == nil || rt.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		addr, err := parseAddress(action.Variable.Address, action.Variable.DataType)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].name", action.Index),
				MsgID: "validate.action_variable_not_writable",
				Args:  []interface{}{action.Variable.Name},
			}})
		}
		data, err := encodeValue(addr, action.Value)
		if err != nil {
			return validation.New([]validation.FieldIssue{{
				Field: fmt.Sprintf("actions[%d].value", action.Index),
				MsgID: "validate.action_value_invalid",
				Args:  []interface{}{action.Variable.Name, err.Error()},
			}})
		}
		var req []byte
		if c.cnet {
			req = buildCnetWriteRequest(c.Station, addr, data)
		} else {
			req = buildFEnetWriteRequest(c.nextInvokeID(), addr, data)
		}
		resp, err := c.exchange(ctx, "write", req)
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		if c.cnet {
			err = parseCnetWriteResponse(resp)
		} else {
			err = parseFEnetWriteResponse(resp)
		}
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

func (c *Collection) exchange(ctx context.Context, op string, req []byte) ([]byte, error) {
	rt := c.runtime
	if rt == nil || rt.pool == nil {
		return nil, collection.ConnectionLost(fmt.Errorf("XGT runtime is not connected"))
	}
	var traceID uint64
	if c.frameTracer != nil {
		traceID = c.frameTracer.NextTraceID()
		c.frameTracer.TraceRequest(traceID, op, req)
	}
	started := time.Now()
	var resp []byte
	err := rt.pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
		tc, ok := client.(*tcpClient)
		if !ok {
			return false, fmt.Errorf("invalid XGT client type")
		}
		frame, exErr := tc.exchange(req)
		if exErr != nil {
			return collection.IsConnectionLost(exErr), exErr
		}
		resp = frame
		return false, nil
	})
	if c.frameTracer != nil && traceID != 0 {
		var respBytes []byte
		if err == nil {
			respBytes = resp
		}
		c.frameTracer.TraceResponse(traceID, op, respBytes, time.Since(started), err)
	}
	return resp, err
}
