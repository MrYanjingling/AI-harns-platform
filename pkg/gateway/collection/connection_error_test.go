package collection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"
)

type statusCollector struct {
	meta *CollectionMeta
}

func (c *statusCollector) GetCollectionMeta() *CollectionMeta { return c.meta }
func (c *statusCollector) SetCollectionMeta(meta *CollectionMeta) {
	c.meta = meta
}
func (c *statusCollector) Connect(context.Context) error { return nil }
func (c *statusCollector) Start(context.Context) error   { return nil }
func (c *statusCollector) Stop(context.Context) error    { return nil }
func (c *statusCollector) GetCollectStatus() string      { return c.meta.GetCollectStatus() }
func (c *statusCollector) SetCollectStatus(status string) {
	c.meta.SetCollectStatus(status)
}
func (c *statusCollector) IndexCollection() { c.meta.IndexCollection() }

func TestIsConnectionLost(t *testing.T) {
	cases := map[string]struct {
		err      error
		expected bool
	}{
		"nil":               {nil, false},
		"eof":               {io.EOF, true},
		"unexpected eof":    {io.ErrUnexpectedEOF, true},
		"closed":            {net.ErrClosed, true},
		"deadline":          {context.DeadlineExceeded, true},
		"reset":             {fmt.Errorf("read tcp: %w", syscall.ECONNRESET), true},
		"refused text":      {errors.New("dial tcp: connection refused"), true},
		"not connected":     {errors.New("runtime is not connected"), true},
		"protocol error":    {errors.New("invalid modbus crc"), false},
		"device exception":  {errors.New("modbus function error 2"), false},
		"wrapped transport": {ConnectionLost(errors.New("port closed")), true},
	}
	for name, tc := range cases {
		if actual := IsConnectionLost(tc.err); actual != tc.expected {
			t.Fatalf("%s: expected %v, got %v", name, tc.expected, actual)
		}
	}
}

func TestApplyCycleFailure(t *testing.T) {
	collector := &statusCollector{meta: &CollectionMeta{}}

	if cont := ApplyCycleFailure(collector, io.EOF, 0, 1, false); cont {
		t.Fatal("expected polling to stop after a connection loss")
	}
	if status := collector.GetCollectStatus(); status != CollectStatusUnconnected {
		t.Fatalf("expected unconnected, got %s", status)
	}

	if cont := ApplyCycleFailure(collector, errors.New("read timeout"), 1, 1, false); !cont {
		t.Fatal("expected polling to continue after a partial failure")
	}
	if status := collector.GetCollectStatus(); status != CollectStatusCollectingError {
		t.Fatalf("expected collectingError, got %s", status)
	}

	if cont := ApplyCycleFailure(collector, errors.New("invalid crc"), 0, 1, true); !cont {
		t.Fatal("expected polling to continue after a protocol failure")
	}
	if status := collector.GetCollectStatus(); status != CollectStatusCollectingError {
		t.Fatalf("expected collectingError, got %s", status)
	}
}
