package mc

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func startMC3EServer(t *testing.T, responseData []byte) (int, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				request := make([]byte, 21)
				if _, err := io.ReadFull(conn, request); err != nil {
					return
				}
				response := []byte{0xD0, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, byte(2 + len(responseData)), 0x00, 0x00, 0x00}
				response = append(response, responseData...)
				_, _ = conn.Write(response)
			}(conn)
		}
	}()
	cleanup := func() {
		_ = listener.Close()
		<-done
	}
	return listener.Addr().(*net.TCPAddr).Port, cleanup
}

func TestMC3EPollReadsWord(t *testing.T) {
	port, cleanup := startMC3EServer(t, []byte{0x34, 0x12})
	defer cleanup()
	item, err := NewCollection(Info3E, &CreateReq{
		Name:             "mc3e",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Host: "127.0.0.1", Port: port, TimeoutMs: 500, IntervalMs: 1},
		Variables:        []*collection.Variable{{Name: "d0", Address: "D0", DataType: "INT16", AccessMode: collection.AccessModeRead}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Stop(context.Background()) })
	if ok := item.poll(context.Background()); !ok {
		t.Fatal("poll stopped")
	}
	value, ok := item.GetVariableValue("d0")
	if !ok || value != int16(0x1234) {
		t.Fatalf("value=%#v", value)
	}
}

func TestMC4EUsesSerialNumber(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	seen := make(chan uint16, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		request := make([]byte, 25)
		if _, err := io.ReadFull(conn, request); err != nil {
			return
		}
		serial := uint16(request[2]) | uint16(request[3])<<8
		seen <- serial
		response := []byte{0xD4, 0x00, byte(serial), byte(serial >> 8), 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, 0x04, 0x00, 0x00, 0x00, 0x34, 0x12}
		_, _ = conn.Write(response)
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	item, err := NewCollection(Info4E, &CreateReq{
		Name:             "mc4e",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Host: "127.0.0.1", Port: port, TimeoutMs: 500, IntervalMs: 1},
		Variables:        []*collection.Variable{{Name: "d0", Address: "D0", DataType: "INT16", AccessMode: collection.AccessModeRead}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Stop(context.Background()) })
	if ok := item.poll(context.Background()); !ok {
		t.Fatal("poll stopped")
	}
	select {
	case serial := <-seen:
		if serial != 0 {
			t.Fatalf("first serial=%d, want 0", serial)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not receive request")
	}
}

func TestMCPollMarksUnconnectedOnLinkFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	item, err := NewCollection(Info3E, &CreateReq{
		Name:             "mc3e-fail",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Host: "127.0.0.1", Port: port, TimeoutMs: 100, IntervalMs: 1},
		Variables:        []*collection.Variable{{Name: "d0", Address: "D0", DataType: "INT16", AccessMode: collection.AccessModeRead}},
	})
	if err != nil {
		t.Fatal(err)
	}
	item.Host = "127.0.0.1"
	item.Port = port
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Stop(context.Background()) })
	if ok := item.poll(context.Background()); ok {
		t.Fatal("poll should stop on link failure")
	}
	if status := item.GetCollectStatus(); status != collection.CollectStatusUnconnected {
		t.Fatalf("status=%s", status)
	}
}
