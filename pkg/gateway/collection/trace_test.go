package collection

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceHubSeparatesRequestsAndResponses(t *testing.T) {
	hub := NewTraceHub()
	requests, cancelRequests := hub.Subscribe(true)
	defer cancelRequests()
	responses, cancelResponses := hub.Subscribe(false)
	defer cancelResponses()

	traceID := hub.NextTraceID()
	hub.TraceRequest(traceID, "read", []byte{0x01, 0x03})
	hub.TraceResponse(traceID, "read", []byte{0x01, 0x03, 0x02}, 5*time.Millisecond, nil)

	request := <-requests
	assert.Equal(t, traceID, request.TraceID)
	assert.Equal(t, "01 03", request.Hex)
	assert.Equal(t, "read", request.Operation)
	response := <-responses
	assert.Equal(t, "01 03 02", response.Hex)
	assert.Equal(t, int64(5), response.DurationMs)
}

func TestTraceHubSkipsWhenNoSubscribers(t *testing.T) {
	hub := NewTraceHub()
	traceID := hub.NextTraceID()
	hub.TraceRequest(traceID, "read", []byte{0x01})
	hub.TraceResponse(traceID, "read", nil, time.Millisecond, nil)
}

func TestTraceHubClosesSubscribers(t *testing.T) {
	hub := NewTraceHub()
	requests, _ := hub.Subscribe(true)
	hub.Close()
	_, ok := <-requests
	assert.False(t, ok)
}

func TestFormatHex(t *testing.T) {
	assert.Equal(t, "01 03 C4 0B", formatHex([]byte{0x01, 0x03, 0xC4, 0x0B}))
	require.Equal(t, "", formatHex(nil))
}
