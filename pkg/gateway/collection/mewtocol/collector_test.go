package mewtocol

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func TestCollectorReadsAndMarksConnectionLost(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		request := make([]byte, 0, 128)
		buf := make([]byte, 1)
		for {
			if _, err := conn.Read(buf); err != nil {
				return
			}
			request = append(request, buf[0])
			if buf[0] == '\r' {
				break
			}
		}
		response := appendResponse([]byte("%01RD000A"))
		_, _ = conn.Write(response)
		time.Sleep(100 * time.Millisecond)
	}()

	item, err := NewCollection(&CreateReq{
		Name: "mewtocol-test", CollectorCycleMs: 100,
		ProtocolConfig: ProtocolConfig{Location: "127.0.0.1", Port: port, Unit: 1, TimeoutMs: 200},
		Variables: []*collection.Variable{{
			Name: "value", DataType: "INT16", Address: "DT0", AccessMode: collection.AccessModeRead,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := item.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := item.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer item.Stop(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if value, ok := item.GetCollectionMeta().GetVariableValue("value"); ok && value == int16(10) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	value, ok := item.GetCollectionMeta().GetVariableValue("value")
	if !ok || value != int16(10) {
		t.Fatalf("unexpected value %v", value)
	}

	for time.Now().Before(deadline) {
		if item.GetCollectStatus() == collection.CollectStatusUnconnected {
			<-done
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected unconnected status, got %s", item.GetCollectStatus())
}

func appendResponse(content []byte) []byte {
	return []byte(string(content) + hexByte(checksum(content)) + "\r")
}

func hexByte(value byte) string {
	return fmt.Sprintf("%02X", value)
}
