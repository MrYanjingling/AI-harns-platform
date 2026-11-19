package fins

import (
	"context"
	"fmt"
	"net"
	"strconv"
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
		done <- serveFINSRead(listener)
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
		Name:             "fins-invalid-calculation",
		CollectorCycleMs: 1000,
		ProtocolConfig: ProtocolConfig{
			Location:  host,
			Port:      port,
			Transport: TransportTCP,
			Expires:   1000,
			MaxSize:   DefaultMaxSize,
		},
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
		t.Fatal("fins server did not finish")
	}
}

func serveFINSRead(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, _, _, err := readTCPPacket(conn); err != nil {
		return err
	}
	if _, err := conn.Write(wrapTCP(1, []byte{0, 0, 0, 1, 0, 0, 0, 2})); err != nil {
		return err
	}
	command, _, request, err := readTCPPacket(conn)
	if err != nil {
		return err
	}
	if command != 2 {
		return fmt.Errorf("unexpected fins command %d", command)
	}
	response := []byte{
		0xC0, 0x00, 0x02, 0x00,
		0x02, 0x00, 0x00, 0x01,
		0x00, request[9],
		0x01, 0x01, 0x00, 0x00,
		0x00, 0x0A,
	}
	_, err = conn.Write(wrapTCP(2, response))
	return err
}

func TestDecodeFrameKeepsSuccessfulValues(t *testing.T) {
	frames, err := buildReadFrames([]*collection.Variable{
		{Name: "a", DataType: "int16", Address: "D100", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "int16", Address: "D101", AccessMode: collection.AccessModeRead},
	}, DefaultMaxSize)
	if err != nil {
		t.Fatal(err)
	}
	values, err := decodeFrame(frames[0], []byte{0x00, 0x0A})
	if err == nil {
		t.Fatal("expected decode error")
	}
	if len(values) != 1 || values[0].Variable.Name != "a" || values[0].Value != int16(10) {
		t.Fatalf("unexpected decoded values %+v", values)
	}
}

func TestConcurrentFramesUseMultipleConnections(t *testing.T) {
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
				errs <- serveFINSReadConn(conn, &active, &maxActive)
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
		Name:             "fins-concurrent",
		CollectorCycleMs: 1000,
		ProtocolConfig: ProtocolConfig{
			Location:    host,
			Port:        port,
			Transport:   TransportTCP,
			Concurrency: 2,
			Expires:     1000,
			MaxSize:     DefaultMaxSize,
		},
		Variables: []*collection.Variable{
			{Name: "a", DataType: "int16", Address: "D100", AccessMode: collection.AccessModeRead},
			{Name: "b", DataType: "int16", Address: "D200", AccessMode: collection.AccessModeRead},
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
		t.Fatalf("expected two concurrent requests, got %d", atomic.LoadInt32(&maxActive))
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fins server did not finish")
	}
}

func serveFINSReadConn(conn net.Conn, active, maxActive *int32) error {
	defer conn.Close()
	if _, _, _, err := readTCPPacket(conn); err != nil {
		return err
	}
	if _, err := conn.Write(wrapTCP(1, []byte{0, 0, 0, 1, 0, 0, 0, 2})); err != nil {
		return err
	}
	command, _, request, err := readTCPPacket(conn)
	if err != nil {
		return err
	}
	if command != 2 {
		return fmt.Errorf("unexpected fins command %d", command)
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
	response := []byte{
		0xC0, 0x00, 0x02, 0x00,
		0x02, 0x00, 0x00, 0x01,
		0x00, request[9],
		0x01, 0x01, 0x00, 0x00,
		0x00, 0x0A,
	}
	_, err = conn.Write(wrapTCP(2, response))
	return err
}
