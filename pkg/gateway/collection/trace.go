package collection

import (
	"encoding/hex"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// FrameEvent is one live protocol frame pushed over SSE.
type FrameEvent struct {
	TraceID    uint64    `json:"traceId"`
	Time       time.Time `json:"time"`
	Operation  string    `json:"operation"`
	Hex        string    `json:"hex"`
	Length     int       `json:"length"`
	DurationMs int64     `json:"durationMs,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// FrameTracer records raw request and response frames when subscribers exist.
type FrameTracer interface {
	NextTraceID() uint64
	TraceRequest(traceID uint64, operation string, data []byte)
	TraceResponse(traceID uint64, operation string, data []byte, duration time.Duration, err error)
}

// FrameTracerAware is implemented by collectors that support raw frame tracing.
type FrameTracerAware interface {
	SetFrameTracer(FrameTracer)
}

// TraceHub broadcasts request and response frames to separate SSE subscribers.
type TraceHub struct {
	mu sync.Mutex

	requestSubs  map[chan FrameEvent]struct{}
	responseSubs map[chan FrameEvent]struct{}
	closed       bool
	traceID      atomic.Uint64
}

// NewTraceHub creates a trace hub.
func NewTraceHub() *TraceHub {
	return &TraceHub{
		requestSubs:  make(map[chan FrameEvent]struct{}),
		responseSubs: make(map[chan FrameEvent]struct{}),
	}
}

// NextTraceID returns a monotonically increasing trace id.
func (h *TraceHub) NextTraceID() uint64 {
	if h == nil {
		return 0
	}
	return h.traceID.Add(1)
}

// TraceRequest publishes one request frame when request subscribers exist.
func (h *TraceHub) TraceRequest(traceID uint64, operation string, data []byte) {
	if h == nil || len(data) == 0 {
		return
	}
	h.mu.Lock()
	active := len(h.requestSubs) > 0 && !h.closed
	h.mu.Unlock()
	if !active {
		return
	}
	h.publish(true, FrameEvent{
		TraceID:   traceID,
		Time:      time.Now().UTC(),
		Operation: operation,
		Hex:       formatHex(data),
		Length:    len(data),
	})
}

// TraceResponse publishes one response frame when response subscribers exist.
func (h *TraceHub) TraceResponse(traceID uint64, operation string, data []byte, duration time.Duration, err error) {
	if h == nil {
		return
	}
	h.mu.Lock()
	active := len(h.responseSubs) > 0 && !h.closed
	h.mu.Unlock()
	if !active {
		return
	}
	event := FrameEvent{
		TraceID:    traceID,
		Time:       time.Now().UTC(),
		Operation:  operation,
		Hex:        formatHex(data),
		Length:     len(data),
		DurationMs: duration.Milliseconds(),
	}
	if err != nil {
		event.Error = err.Error()
	}
	h.publish(false, event)
}

// Subscribe subscribes to request or response events.
func (h *TraceHub) Subscribe(requests bool) (<-chan FrameEvent, func()) {
	ch := make(chan FrameEvent, 32)
	if h == nil {
		close(ch)
		return ch, func() {}
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		close(ch)
		return ch, func() {}
	}
	if requests {
		h.requestSubs[ch] = struct{}{}
	} else {
		h.responseSubs[ch] = struct{}{}
	}
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if requests {
				if _, ok := h.requestSubs[ch]; ok {
					delete(h.requestSubs, ch)
					close(ch)
				}
			} else {
				if _, ok := h.responseSubs[ch]; ok {
					delete(h.responseSubs, ch)
					close(ch)
				}
			}
		})
	}
	return ch, cancel
}

// Close closes all subscribers and rejects new subscriptions.
func (h *TraceHub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for ch := range h.requestSubs {
		close(ch)
		delete(h.requestSubs, ch)
	}
	for ch := range h.responseSubs {
		close(ch)
		delete(h.responseSubs, ch)
	}
}

func (h *TraceHub) publish(requests bool, event FrameEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	subs := h.responseSubs
	if requests {
		subs = h.requestSubs
	}
	for ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func formatHex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	encoded := strings.ToUpper(hex.EncodeToString(data))
	var builder strings.Builder
	builder.Grow(len(encoded) + len(data) - 1)
	for i := 0; i < len(encoded); i += 2 {
		if i > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(encoded[i : i+2])
	}
	return builder.String()
}
