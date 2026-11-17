package pushmgr

import (
	"encoding/json"
	"sync"
	"time"
)

const commandBufferSize = 50

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
	buffer []PushCommandEvent
	closed bool
}

func newCommandHub() *commandHub {
	return &commandHub{
		subs:   make(map[chan PushCommandEvent]struct{}),
		buffer: make([]PushCommandEvent, 0, commandBufferSize),
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
	h.buffer = append(h.buffer, event)
	if len(h.buffer) > commandBufferSize {
		h.buffer = append([]PushCommandEvent(nil), h.buffer[len(h.buffer)-commandBufferSize:]...)
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
	for _, event := range h.buffer {
		select {
		case ch <- event:
		default:
		}
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
