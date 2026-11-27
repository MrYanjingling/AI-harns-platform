package cclinkie

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type collectorRuntime struct {
	mu       sync.Mutex
	conn     net.Conn
	points   []*point
	calcs    []*collection.Calculation
	calcErr  error
	masterID uint32
	slaveID  uint32
	occupied int
	groupNo  byte
	timeout  time.Duration
	seq      uint16
	rww      []byte
	ry       []byte
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	started  bool
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("CC-Link IE collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	cfg := ProtocolConfig{
		Location:         c.Location,
		OccupiedStations: c.OccupiedStations,
		GroupNo:          c.GroupNo,
		TimeoutMs:        c.TimeoutMs,
	}
	cfg.Normalize()
	host, endpoint := cfg.endpoint()
	if endpoint == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: location is required", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	pts := make([]*point, 0, len(variables))
	for _, v := range variables {
		if v == nil || v.IsCalculated || strings.TrimSpace(v.Address) == "" {
			continue
		}
		p, err := parseAddress(v)
		if err != nil {
			c.SetCollectStatus(collection.CollectStatusError)
			return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
		}
		pts = append(pts, p)
	}
	calcs, calcErr := collection.CompileCalculations(variables)
	rt := &collectorRuntime{
		points:   pts,
		calcs:    calcs,
		calcErr:  calcErr,
		slaveID:  ipv4ToUint32(host),
		occupied: cfg.OccupiedStations,
		groupNo:  byte(cfg.GroupNo),
		timeout:  time.Duration(cfg.TimeoutMs) * time.Millisecond,
		rww:      make([]byte, cfg.OccupiedStations*bytesWordArea),
		ry:       make([]byte, cfg.OccupiedStations*bytesBitArea),
	}
	c.runtime = rt
	if len(pts) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	dialer := net.Dialer{Timeout: rt.timeout}
	conn, err := dialer.DialContext(ctx, "udp4", endpoint)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok && localAddr.IP != nil {
		rt.masterID = ipv4ToUint32(localAddr.IP.String())
	} else {
		rt.masterID = 0x0100007F
	}
	rt.conn = conn
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
	if len(rt.points) == 0 {
		rt.mu.Unlock()
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	if rt.conn == nil {
		rt.mu.Unlock()
		return fmt.Errorf("CC-Link IE runtime is not connected")
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
	conn := rt.conn
	rt.conn = nil
	rt.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	rt.wg.Wait()
	if conn != nil {
		_ = conn.Close()
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

func (rt *collectorRuntime) exchangeLocked() ([]byte, []byte, error) {
	if rt.conn == nil {
		return nil, nil, collection.ConnectionLost(fmt.Errorf("CC-Link IE UDP socket is closed"))
	}
	rt.seq++
	req := buildCyclicRequest(rt.masterID, rt.slaveID, rt.groupNo, rt.seq, rt.occupied, rt.rww, rt.ry)
	if err := rt.conn.SetDeadline(time.Now().Add(rt.timeout)); err != nil {
		return nil, nil, collection.ConnectionLost(err)
	}
	if _, err := rt.conn.Write(req); err != nil {
		return nil, nil, collection.ConnectionLost(err)
	}
	buf := make([]byte, 2048)
	n, err := rt.conn.Read(buf)
	if err != nil {
		return nil, nil, collection.ConnectionLost(err)
	}
	return parseCyclicResponse(buf[:n])
}

func (c *Collection) poll(ctx context.Context) bool {
	rt := c.runtime
	if rt == nil {
		return false
	}
	rt.mu.Lock()
	rwr, rx, err := rt.exchangeLocked()
	rwwSnap := append([]byte(nil), rt.rww...)
	rySnap := append([]byte(nil), rt.ry...)
	rt.mu.Unlock()
	if err != nil {
		if collection.IsConnectionLost(err) {
			c.GetCollectionMeta().ReportError(err)
			return collection.MarkConnectionLost(c, err)
		}
		c.GetCollectionMeta().ReportError(err)
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	var lastErr error
	for _, p := range rt.points {
		var val any
		switch p.area {
		case areaRX:
			val, err = decodeBit(rx, p.index, p.dataType)
		case areaRY:
			val, err = decodeBit(rySnap, p.index, p.dataType)
		case areaRWr:
			val, err = decodeWordPoint(p, rwr)
		case areaRWw:
			val, err = decodeWordPoint(p, rwwSnap)
		}
		if err != nil {
			lastErr = err
			continue
		}
		c.GetCollectionMeta().SetVariableValue(p.variable.Name, collection.ApplyTransform(val, p.variable))
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
	if rt == nil {
		return validation.New([]validation.FieldIssue{{Field: "actions", MsgID: "validate.action_collection_not_ready"}})
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.conn == nil {
		return validation.New([]validation.FieldIssue{{Field: "actions", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		p, err := parseAddress(action.Variable)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.variable_address_invalid", Args: []interface{}{action.Variable.Address}}})
		}
		switch p.area {
		case areaRY:
			b, err := toBool(action.Value)
			if err != nil {
				return collection.DeviceActionError(action, err)
			}
			if err := setBit(rt.ry, p.index, b); err != nil {
				return collection.DeviceActionError(action, err)
			}
		case areaRWw:
			if err := encodeWordPoint(p, rt.rww, action.Value); err != nil {
				return collection.DeviceActionError(action, err)
			}
		default:
			return collection.DeviceActionError(action, fmt.Errorf("CC-Link IE input area %s is read-only", action.Variable.Address))
		}
	}
	if _, _, err := rt.exchangeLocked(); err != nil {
		return collection.DeviceActionError(actions[0], err)
	}
	for _, action := range actions {
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

var _ collection.Collector = (*Collection)(nil)
