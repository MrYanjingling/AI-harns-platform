package pushmgr

import (
	"encoding/json"
	"sync"
	"time"
)

// PushCommandEvent is one command message received from a command topic.
type PushCommandEvent struct {
	GroupID       string          `json:"groupId"`
	ConnectorID   string          `json:"connectorId"`
	ConnectorType string          `json:"connectorType"`
	Topic         string          `json:"topic"`
	Time          time.Time       `json:"time"`
	Payload       json.RawMessage `json:"payload"`
}

type commandHub struct {
	mu     sync.Mutex
	subs   map[chan PushCommandEvent]struct{}
	closed bool
}

func newCommandHub() *commandHub {
	return &commandHub{
		subs: make(map[chan PushCommandEvent]struct{}),
	}
}

func (h *commandHub) Publish(event PushCommandEvent) {
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

func (h *commandHub) Subscribe() (<-chan PushCommandEvent, func()) {
	ch := make(chan PushCommandEvent, 64)
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

func (h *commandHub) Close() {
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
