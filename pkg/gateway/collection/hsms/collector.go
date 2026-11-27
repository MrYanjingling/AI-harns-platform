package hsms

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

type sfKey struct {
	stream   uint8
	function uint8
}

type hsmsClient struct {
	conn      net.Conn
	deviceID  uint16
	timeout   time.Duration
	onPush    func(msg *Message)
	onClosed  func(err error)
	writeMu   sync.Mutex
	pendingMu sync.Mutex
	pending   map[uint32]chan *Message
	sysSeq    atomic.Uint32
	closeOnce sync.Once
	closedCh  chan struct{}
}

func dialHSMS(ctx context.Context, host string, port int, deviceID uint16, timeout time.Duration, establishComm bool, onPush func(*Message), onClosed func(error)) (*hsmsClient, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	cli := &hsmsClient{
		conn:     conn,
		deviceID: deviceID,
		timeout:  timeout,
		onPush:   onPush,
		onClosed: onClosed,
		pending:  make(map[uint32]chan *Message),
		closedCh: make(chan struct{}),
	}
	go cli.readLoop()

	// Step 1: HSMS Select.req (SType=1) -> Select.rsp (SType=2)
	selSys := cli.nextSystemBytes()
	selRsp, err := cli.request(ctx, buildControlMessage(STypeSelectReq, 0, selSys))
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("HSMS Select.req failed: %w", err)
	}
	if selRsp.Header.SType != STypeSelectRsp {
		_ = cli.Close()
		return nil, fmt.Errorf("HSMS expected Select.rsp (SType=2), got SType=%d", selRsp.Header.SType)
	}
	if selRsp.Header.Function != 0 && selRsp.Header.Function != 1 {
		_ = cli.Close()
		return nil, fmt.Errorf("HSMS Select.rsp rejected with status %d", selRsp.Header.Function)
	}

	// Step 2: Optional GEM S1F13 Establish Communications -> S1F14 (COMMACK == 0)
	if establishComm {
		body, _ := (&Item{Type: ItemList}).Encode()
		s1f14, err := cli.request(ctx, buildDataMessage(deviceID, 1, 13, true, cli.nextSystemBytes(), body))
		if err != nil {
			_ = cli.Close()
			return nil, fmt.Errorf("HSMS S1F13 failed: %w", err)
		}
		if err := validateS1F14(s1f14); err != nil {
			_ = cli.Close()
			return nil, err
		}
	}
	return cli, nil
}

func validateS1F14(msg *Message) error {
	if msg.Header.SType != STypeData || msg.Header.Stream != 1 || msg.Header.Function != 14 {
		return fmt.Errorf("HSMS expected S1F14, got S%dF%d (SType=%d)", msg.Header.Stream, msg.Header.Function, msg.Header.SType)
	}
	if len(msg.Body) == 0 {
		return nil
	}
	root, _, err := DecodeItem(msg.Body)
	if err != nil {
		return fmt.Errorf("HSMS S1F14 decode failed: %w", err)
	}
	if root.Type == ItemList && len(root.Children) >= 1 {
		ack := root.Children[0]
		if (ack.Type == ItemBinary || ack.Type == ItemUint8) && len(ack.Raw) >= 1 && ack.Raw[0] != 0 {
			return fmt.Errorf("HSMS S1F14 COMMACK denied: 0x%02X", ack.Raw[0])
		}
	}
	return nil
}

func (c *hsmsClient) nextSystemBytes() uint32 {
	v := c.sysSeq.Add(1)
	if v == 0 {
		v = c.sysSeq.Add(1)
	}
	return v
}

func (c *hsmsClient) send(msg *Message) error {
	raw := msg.Encode()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	_, err := c.conn.Write(raw)
	return err
}

func (c *hsmsClient) request(ctx context.Context, msg *Message) (*Message, error) {
	sys := msg.Header.SystemBytes
	ch := make(chan *Message, 1)
	c.pendingMu.Lock()
	select {
	case <-c.closedCh:
		c.pendingMu.Unlock()
		return nil, collection.ConnectionLost(fmt.Errorf("HSMS connection closed"))
	default:
		c.pending[sys] = ch
	}
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, sys)
		c.pendingMu.Unlock()
	}()

	if err := c.send(msg); err != nil {
		return nil, collection.ConnectionLost(err)
	}

	timer := time.NewTimer(c.timeout)
	defer timer.Stop()
	select {
	case resp, ok := <-ch:
		if !ok || resp == nil {
			return nil, collection.ConnectionLost(fmt.Errorf("HSMS connection closed while waiting for response"))
		}
		return resp, nil
	case <-c.closedCh:
		return nil, collection.ConnectionLost(fmt.Errorf("HSMS connection lost"))
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, fmt.Errorf("HSMS T3/T6 timeout waiting for reply to S%dF%d (SType=%d)", msg.Header.Stream, msg.Header.Function, msg.Header.SType)
	}
}

func (c *hsmsClient) linktest(ctx context.Context) error {
	rsp, err := c.request(ctx, buildControlMessage(STypeLinktestReq, 0, c.nextSystemBytes()))
	if err != nil {
		return err
	}
	if rsp.Header.SType != STypeLinktestRsp {
		return fmt.Errorf("HSMS expected Linktest.rsp, got SType=%d", rsp.Header.SType)
	}
	return nil
}

func (c *hsmsClient) readLoop() {
	var closeErr error
	defer func() {
		c.closeWithErr(closeErr)
	}()
	for {
		msg, err := ReadMessage(c.conn)
		if err != nil {
			closeErr = err
			return
		}
		switch msg.Header.SType {
		case STypeSeparateReq:
			closeErr = fmt.Errorf("HSMS peer sent Separate.req")
			return
		case STypeLinktestReq:
			_ = c.send(buildControlMessage(STypeLinktestRsp, 0, msg.Header.SystemBytes))
			continue
		case STypeSelectReq:
			_ = c.send(buildControlMessage(STypeSelectRsp, 0, msg.Header.SystemBytes))
			continue
		case STypeDeselectReq:
			_ = c.send(buildControlMessage(STypeDeselectRsp, 0, msg.Header.SystemBytes))
			continue
		}

		// Deliver to waiting caller if SystemBytes matches a pending request.
		c.pendingMu.Lock()
		ch, ok := c.pending[msg.Header.SystemBytes]
		if ok {
			delete(c.pending, msg.Header.SystemBytes)
		}
		c.pendingMu.Unlock()
		if ok {
			ch <- msg
			continue
		}

		// Unsolicited data message (equipment-initiated primary, e.g. S5F1, S6F11, S1F13).
		if msg.Header.SType == STypeData {
			if c.onPush != nil {
				c.onPush(msg)
			}
			if msg.Header.WBit {
				c.sendAutoAck(msg)
			}
		}
	}
}

func (c *hsmsClient) sendAutoAck(req *Message) {
	stream := req.Header.Stream
	fn := req.Header.Function + 1
	var body []byte
	switch {
	case stream == 1 && req.Header.Function == 13:
		item := &Item{
			Type: ItemList,
			Children: []*Item{
				{Type: ItemBinary, Raw: []byte{0x00}},
				{Type: ItemList},
			},
		}
		body, _ = item.Encode()
	case stream == 1 && req.Header.Function == 1:
		body, _ = (&Item{Type: ItemList}).Encode()
	default:
		body, _ = (&Item{Type: ItemBinary, Raw: []byte{0x00}}).Encode()
	}
	_ = c.send(buildDataMessage(c.deviceID, stream, fn, false, req.Header.SystemBytes, body))
}

func (c *hsmsClient) closeWithErr(err error) {
	c.closeOnce.Do(func() {
		close(c.closedCh)
		_ = c.conn.Close()
		c.pendingMu.Lock()
		for k, ch := range c.pending {
			close(ch)
			delete(c.pending, k)
		}
		c.pendingMu.Unlock()
		if c.onClosed != nil && err != nil {
			c.onClosed(err)
		}
	})
}

func (c *hsmsClient) Close() error {
	c.closeOnce.Do(func() {
		// Best-effort Separate.req before closing TCP.
		_ = c.send(buildControlMessage(STypeSeparateReq, 0, c.nextSystemBytes()))
		close(c.closedCh)
		_ = c.conn.Close()
		c.pendingMu.Lock()
		for k, ch := range c.pending {
			close(ch)
			delete(c.pending, k)
		}
		c.pendingMu.Unlock()
	})
	return nil
}

type pollGroup struct {
	key         sfKey
	s1f3Numeric bool
	points      []*point
}

type collectorRuntime struct {
	mu           sync.Mutex
	points       []*point
	pollGroups   []*pollGroup
	pushMap      map[sfKey][]*point
	calcs        []*collection.Calculation
	calcErr      error
	client       *hsmsClient
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	started      bool
	lastLinktest time.Time
}

func buildRuntimePlan(variables []*collection.Variable) ([]*point, []*pollGroup, map[sfKey][]*point, error) {
	points := make([]*point, 0, len(variables))
	pushMap := make(map[sfKey][]*point)
	groupOrder := make([]sfKey, 0)
	groupByKey := make(map[sfKey]*pollGroup)
	var s1f3NumPoints []*point

	for _, v := range variables {
		if v == nil || v.IsCalculated || strings.TrimSpace(v.Address) == "" {
			continue
		}
		p, err := parseAddress(v)
		if err != nil {
			return nil, nil, nil, err
		}
		points = append(points, p)

		// Map both exact SxFy (for unsolicited/reports) and secondary SxF(y+1) (for replies).
		k := sfKey{stream: p.stream, function: p.function}
		pushMap[k] = append(pushMap[k], p)

		// Active polling: odd function, not in report-only list, and variable is readable (not write-only "w").
		if (p.function&1) != 0 && !isReportOnly(p.stream, p.function) && !strings.EqualFold(strings.TrimSpace(string(v.AccessMode)), "w") {
			if p.stream == 1 && p.function == 3 && p.kind == indexSVID && p.svidIsNum {
				s1f3NumPoints = append(s1f3NumPoints, p)
				continue
			}
			g, ok := groupByKey[k]
			if !ok {
				g = &pollGroup{key: k}
				groupByKey[k] = g
				groupOrder = append(groupOrder, k)
			}
			g.points = append(g.points, p)
		}
	}

	groups := make([]*pollGroup, 0, len(groupOrder)+1)
	if len(s1f3NumPoints) > 0 {
		groups = append(groups, &pollGroup{
			key:         sfKey{stream: 1, function: 3},
			s1f3Numeric: true,
			points:      s1f3NumPoints,
		})
	}
	for _, k := range groupOrder {
		groups = append(groups, groupByKey[k])
	}
	return points, groups, pushMap, nil
}

func (c *Collection) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("HSMS collector is nil")
	}
	if c.runtime != nil && collection.ShouldStopBeforeConnect(c.GetCollectStatus()) {
		_ = c.Stop(ctx)
	}
	if strings.TrimSpace(c.Host) == "" {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: host is required", collection.ErrInvalidConfig)
	}
	variables := c.GetCollectionMeta().GetVariables()
	points, groups, pushMap, err := buildRuntimePlan(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calcs, calcErr := collection.CompileCalculations(variables)
	rt := &collectorRuntime{
		points:     points,
		pollGroups: groups,
		pushMap:    pushMap,
		calcs:      calcs,
		calcErr:    calcErr,
	}
	c.runtime = rt
	if len(points) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}

	timeout := time.Duration(c.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = time.Duration(DefaultTimeoutMs) * time.Millisecond
	}
	cli, err := dialHSMS(ctx, c.Host, c.Port, uint16(c.DeviceID), timeout, c.EstablishComm != 0,
		func(msg *Message) { c.handleUnsolicited(rt, msg) },
		func(err error) {
			rt.mu.Lock()
			started := rt.started
			rt.mu.Unlock()
			if started {
				c.GetCollectionMeta().ReportError(err)
				collection.MarkConnectionLost(c, err)
			}
		},
	)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	rt.client = cli
	rt.lastLinktest = time.Now()
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *Collection) handleUnsolicited(rt *collectorRuntime, msg *Message) {
	if msg == nil || len(msg.Body) == 0 {
		return
	}
	root, _, err := DecodeItem(msg.Body)
	if err != nil {
		return
	}
	k := sfKey{stream: msg.Header.Stream, function: msg.Header.Function}
	pts := rt.pushMap[k]
	if len(pts) == 0 {
		return
	}
	for _, p := range pts {
		target, err := extractItem(root, msg.Header.Stream, msg.Header.Function, p)
		if err != nil {
			continue
		}
		val, err := DecodeItemValue(target, p.dataType)
		if err != nil {
			continue
		}
		c.GetCollectionMeta().SetVariableValue(p.variable.Name, collection.ApplyTransform(val, p.variable))
	}
	if rt.calcErr == nil {
		_ = collection.Calculate(c.GetCollectionMeta(), rt.calcs)
	}
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
	if rt.client == nil {
		rt.mu.Unlock()
		return fmt.Errorf("HSMS runtime is not connected")
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
	cli := rt.client
	rt.client = nil
	rt.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if cli != nil {
		_ = cli.Close()
	}
	rt.wg.Wait()
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

func (c *Collection) poll(ctx context.Context) bool {
	rt := c.runtime
	if rt == nil {
		return false
	}
	rt.mu.Lock()
	cli := rt.client
	rt.mu.Unlock()
	if cli == nil {
		return false
	}
	select {
	case <-cli.closedCh:
		return collection.MarkConnectionLost(c, fmt.Errorf("HSMS connection closed"))
	default:
	}

	var lastErr error
	for _, group := range rt.pollGroups {
		if ctx.Err() != nil {
			return false
		}
		if group.s1f3Numeric {
			if err := c.pollS1F3Numeric(ctx, cli, group.points); err != nil {
				if collection.IsConnectionLost(err) {
					c.GetCollectionMeta().ReportError(err)
					return collection.MarkConnectionLost(c, err)
				}
				lastErr = err
			}
			continue
		}
		if err := c.pollGenericGroup(ctx, cli, group); err != nil {
			if collection.IsConnectionLost(err) {
				c.GetCollectionMeta().ReportError(err)
				return collection.MarkConnectionLost(c, err)
			}
			lastErr = err
		}
	}

	// Periodic Linktest keepalive when LinktestIntervalMs > 0 (essential for report-only nodes).
	if c.LinktestIntervalMs > 0 && time.Since(rt.lastLinktest) >= time.Duration(c.LinktestIntervalMs)*time.Millisecond {
		if err := cli.linktest(ctx); err != nil {
			if collection.IsConnectionLost(err) {
				c.GetCollectionMeta().ReportError(err)
				return collection.MarkConnectionLost(c, err)
			}
			lastErr = err
		} else {
			rt.lastLinktest = time.Now()
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

func (c *Collection) pollS1F3Numeric(ctx context.Context, cli *hsmsClient, pts []*point) error {
	children := make([]*Item, 0, len(pts))
	for _, p := range pts {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, p.svidNum)
		children = append(children, &Item{Type: ItemUint32, Raw: buf})
	}
	body, err := (&Item{Type: ItemList, Children: children}).Encode()
	if err != nil {
		return err
	}
	req := buildDataMessage(uint16(c.DeviceID), 1, 3, true, cli.nextSystemBytes(), body)
	rsp, err := cli.request(ctx, req)
	if err != nil {
		return err
	}
	if rsp.Header.SType != STypeData || rsp.Header.Stream != 1 || rsp.Header.Function != 4 {
		return fmt.Errorf("HSMS S1F3 unexpected reply S%dF%d (SType=%d)", rsp.Header.Stream, rsp.Header.Function, rsp.Header.SType)
	}
	root, _, err := DecodeItem(rsp.Body)
	if err != nil {
		return fmt.Errorf("HSMS S1F4 decode failed: %w", err)
	}
	if root.Type != ItemList || len(root.Children) < len(pts) {
		return fmt.Errorf("HSMS S1F4 expected list of %d items", len(pts))
	}
	var lastErr error
	for i, p := range pts {
		val, err := DecodeItemValue(root.Children[i], p.dataType)
		if err != nil {
			lastErr = err
			continue
		}
		c.GetCollectionMeta().SetVariableValue(p.variable.Name, collection.ApplyTransform(val, p.variable))
	}
	return lastErr
}

func (c *Collection) pollGenericGroup(ctx context.Context, cli *hsmsClient, group *pollGroup) error {
	req := buildDataMessage(uint16(c.DeviceID), group.key.stream, group.key.function, true, cli.nextSystemBytes(), nil)
	rsp, err := cli.request(ctx, req)
	if err != nil {
		return err
	}
	if rsp.Header.SType != STypeData {
		return fmt.Errorf("HSMS S%dF%d unexpected SType=%d", group.key.stream, group.key.function, rsp.Header.SType)
	}
	if rsp.Header.Function == 0 {
		return fmt.Errorf("HSMS S%dF%d aborted by equipment (S%dF0)", group.key.stream, group.key.function, rsp.Header.Stream)
	}
	if len(rsp.Body) == 0 {
		return nil
	}
	root, _, err := DecodeItem(rsp.Body)
	if err != nil {
		return fmt.Errorf("HSMS S%dF%d decode failed: %w", rsp.Header.Stream, rsp.Header.Function, err)
	}
	var lastErr error
	for _, p := range group.points {
		target, err := extractItem(root, rsp.Header.Stream, rsp.Header.Function, p)
		if err != nil {
			lastErr = err
			continue
		}
		val, err := DecodeItemValue(target, p.dataType)
		if err != nil {
			lastErr = err
			continue
		}
		c.GetCollectionMeta().SetVariableValue(p.variable.Name, collection.ApplyTransform(val, p.variable))
	}
	return lastErr
}

func (c *Collection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	rt := c.runtime
	if rt == nil {
		return validation.New([]validation.FieldIssue{{Field: "actions", MsgID: "validate.action_collection_not_ready"}})
	}
	rt.mu.Lock()
	cli := rt.client
	rt.mu.Unlock()
	if cli == nil {
		return validation.New([]validation.FieldIssue{{Field: "actions", MsgID: "validate.action_collection_not_ready"}})
	}
	for _, action := range actions {
		p, err := parseAddress(action.Variable)
		if err != nil {
			return validation.New([]validation.FieldIssue{{Field: fmt.Sprintf("actions[%d].name", action.Index), MsgID: "validate.variable_address_invalid", Args: []interface{}{action.Variable.Address}}})
		}
		item, err := EncodeGoValue(p.dataType, action.Value)
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		body, err := item.Encode()
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		req := buildDataMessage(uint16(c.DeviceID), p.stream, p.function, true, cli.nextSystemBytes(), body)
		rsp, err := cli.request(ctx, req)
		if err != nil {
			return collection.DeviceActionError(action, err)
		}
		if rsp.Header.SType != STypeData || rsp.Header.Function == 0 {
			return collection.DeviceActionError(action, fmt.Errorf("HSMS write S%dF%d rejected (S%dF%d SType=%d)", p.stream, p.function, rsp.Header.Stream, rsp.Header.Function, rsp.Header.SType))
		}
		c.GetCollectionMeta().SetVariableValue(action.Variable.Name, collection.ApplyTransform(action.Value, action.Variable))
	}
	return nil
}

var _ collection.Collector = (*Collection)(nil)
