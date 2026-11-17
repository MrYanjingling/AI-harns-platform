package pushmgr

import (
	"encoding/json"
	"sync"
	"time"
)

type PushPayloadEvent struct {
	GroupID       string          `json:"groupId"`
	ConnectorID   string          `json:"connectorId"`
	ConnectorType string          `json:"connectorType"`
	Topic         string          `json:"topic"`
	Key           string          `json:"key,omitempty"`
	Time          time.Time       `json:"time"`
	DurationMs    int64           `json:"durationMs,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	Error         string          `json:"error,omitempty"`
}

type payloadHub struct {
	mu     sync.Mutex
	subs   map[chan PushPayloadEvent]struct{}
	closed bool
}

func newPayloadHub() *payloadHub {
	return &payloadHub{
		subs: make(map[chan PushPayloadEvent]struct{}),
	}
}

func (h *payloadHub) Publish(event PushPayloadEvent) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	for ch := range h.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *payloadHub) Subscribe() (<-chan PushPayloadEvent, func()) {
	ch := make(chan PushPayloadEvent, 64)
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
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if _, ok := h.subs[ch]; ok {
				delete(h.subs, ch)
				close(ch)
			}
		})
	}
	return ch, cancel
}

func (h *payloadHub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for ch := range h.subs {
		close(ch)
		delete(h.subs, ch)
	}
}
