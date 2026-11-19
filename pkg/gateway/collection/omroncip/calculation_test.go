package omroncip

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func TestInvalidCalculationKeepsCollectionRunning(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	done := make(chan error, 1)
	go func() {
		done <- serveCIPRead(listener)
	}()

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}

	item, err := NewCollection(&CreateReq{
		Name:             "omron-cip-invalid-calculation",
		CollectorCycleMs: 1000,
		ProtocolConfig:   ProtocolConfig{Location: host, Port: port},
		Variables: []*collection.Variable{
			{Name: "temperature", DataType: "int16", Address: "D100", AccessMode: collection.AccessModeRead},
			{Name: "broken", DataType: "float64", IsCalculated: true, Expression: "missing +", AccessMode: collection.AccessModeRead},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Stop(context.Background()) })
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if item.runtime == nil || item.runtime.calcErr == nil {
		t.Fatal("expected calculation error to be retained")
	}
	if item.GetCollectStatus() != collection.CollectStatusStopped {
		t.Fatalf("unexpected connect status %q", item.GetCollectStatus())
	}
	if !item.poll(context.Background()) {
		t.Fatal("poll stopped unexpectedly")
	}
	value, ok := item.GetVariableValue("temperature")
	if !ok || value != int16(10) {
		t.Fatalf("unexpected temperature value %v", value)
	}
	if item.GetCollectStatus() != collection.CollectStatusCollectingError {
		t.Fatalf("unexpected poll status %q", item.GetCollectStatus())
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("omron cip server did not finish")
	}
}

func serveCIPRead(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()

	header := make([]byte, 24)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	length := int(binary.LittleEndian.Uint16(header[2:]))
	body := make([]byte, length)
	if _, err := io.ReadFull(conn, body); err != nil {
		return err
	}
	if _, err := conn.Write(cipResponse(0x0065, 0x1234, []byte{1, 0, 0, 0})); err != nil {
		return err
	}

	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	length = int(binary.LittleEndian.Uint16(header[2:]))
	body = make([]byte, length)
	if _, err := io.ReadFull(conn, body); err != nil {
		return err
	}
	cip := []byte{0xCC, 0, 0, 0, 0xC3, 0, 1, 0, 0x0A, 0}
	payload := []byte{0, 0, 0, 0, 10, 0, 2, 0, 0, 0, 0, 0, 0xB2, 0, byte(len(cip)), 0}
	payload = append(payload, cip...)
	_, err = conn.Write(cipResponse(0x006F, 0x1234, payload))
	return err
}

func TestConcurrentFramesUseMultipleSessions(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	var active int32
	var maxActive int32
	done := make(chan error, 1)
	go func() {
		var wait sync.WaitGroup
		errs := make(chan error, 2)
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				done <- err
				return
			}
			wait.Add(1)
			go func() {
				defer wait.Done()
				errs <- serveCIPReadConn(conn, &active, &maxActive)
			}()
		}
		wait.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	item, err := NewCollection(&CreateReq{
		Name:             "omron-cip-concurrent",
		CollectorCycleMs: 1000,
		ProtocolConfig:   ProtocolConfig{Location: host, Port: port, Concurrency: 2},
		Variables: []*collection.Variable{
			{Name: "a", DataType: "int16", Address: strings.Repeat("A", 600), AccessMode: collection.AccessModeRead},
			{Name: "b", DataType: "int16", Address: strings.Repeat("B", 600), AccessMode: collection.AccessModeRead},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Stop(context.Background()) })
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !item.poll(context.Background()) {
		t.Fatal("poll stopped unexpectedly")
	}
	if atomic.LoadInt32(&maxActive) != 2 {
		t.Fatalf("expected two concurrent requests, got %d", maxActive)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("omron cip server did not finish")
	}
}

func serveCIPReadConn(conn net.Conn, active, maxActive *int32) error {
	defer conn.Close()
	header := make([]byte, 24)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	length := int(binary.LittleEndian.Uint16(header[2:]))
	body := make([]byte, length)
	if _, err := io.ReadFull(conn, body); err != nil {
		return err
	}
	if _, err := conn.Write(cipResponse(0x0065, 0x1234, []byte{1, 0, 0, 0})); err != nil {
		return err
	}
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	length = int(binary.LittleEndian.Uint16(header[2:]))
	body = make([]byte, length)
	if _, err := io.ReadFull(conn, body); err != nil {
		return err
	}
	current := atomic.AddInt32(active, 1)
	for {
		max := atomic.LoadInt32(maxActive)
		if current <= max || atomic.CompareAndSwapInt32(maxActive, max, current) {
			break
		}
	}
	time.Sleep(100 * time.Millisecond)
	atomic.AddInt32(active, -1)
	cip := []byte{0xCC, 0, 0, 0, 0xC3, 0, 1, 0, 0x0A, 0}
	payload := []byte{0, 0, 0, 0, 10, 0, 2, 0, 0, 0, 0, 0, 0xB2, 0, byte(len(cip)), 0}
	payload = append(payload, cip...)
	_, err := conn.Write(cipResponse(0x006F, 0x1234, payload))
	return err
}
