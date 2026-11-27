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

func TestMC1EPollAndWrite(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	wordVal := []byte{0x34, 0x12}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				hdr := make([]byte, 12)
				for {
					if _, err := io.ReadFull(conn, hdr); err != nil {
						return
					}
					switch hdr[0] {
					case a1eCmdWordRead:
						_, _ = conn.Write(append([]byte{0x81, 0x00}, wordVal...))
					case a1eCmdWordWrite:
						buf := make([]byte, int(hdr[10])*2)
						if _, err := io.ReadFull(conn, buf); err != nil {
							return
						}
						wordVal = buf
						_, _ = conn.Write([]byte{0x83, 0x00})
					}
				}
			}(conn)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	item, err := NewCollection(InfoA1E, &CreateReq{
		Name:             "mc1e",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Host: "127.0.0.1", Port: port, TimeoutMs: 500, IntervalMs: 1},
		Variables:        []*collection.Variable{{Name: "d0", Address: "D0", DataType: "INT16", AccessMode: collection.AccessModeReadWrite}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := item.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Stop(ctx) })
	if ok := item.poll(ctx); !ok {
		t.Fatal("1E poll failed")
	}
	if value, ok := item.GetVariableValue("d0"); !ok || value != int16(0x1234) {
		t.Fatalf("read value=%#v", value)
	}
	if err := item.DeliverAction(ctx, []collection.ActionValue{{Variable: item.Variables[0], Value: int16(0x5678)}}); err != nil {
		t.Fatalf("1E write failed: %v", err)
	}
	if ok := item.poll(ctx); !ok {
		t.Fatal("1E poll after write failed")
	}
	if value, ok := item.GetVariableValue("d0"); !ok || value != int16(0x5678) {
		t.Fatalf("readback value=%#v, want 0x5678", value)
	}
}
