package comli

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection"
)

type fakePort struct {
	mu        sync.Mutex
	responses [][]byte
	readErr   error
	writes    [][]byte
}

func (p *fakePort) Read(buf []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.readErr != nil {
		return 0, p.readErr
	}
	if len(p.responses) == 0 {
		return 0, nil
	}
	response := p.responses[0]
	p.responses = p.responses[1:]
	return copy(buf, response), nil
}

func (p *fakePort) Write(buf []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.writes = append(p.writes, append([]byte(nil), buf...))
	return len(buf), nil
}

func (p *fakePort) Close() error                       { return nil }
func (p *fakePort) SetReadTimeout(time.Duration) error { return nil }

func withFakePort(t *testing.T, port *fakePort) {
	t.Helper()
	original := openSerialPort
	openSerialPort = func(string, *serial.Mode) (serialPort, error) { return port, nil }
	t.Cleanup(func() { openSerialPort = original })
}

func newTestCollection(t *testing.T) *Collection {
	t.Helper()
	item, err := NewCollection(&CreateReq{
		Name:             "comli-test",
		CollectorCycleMs: 1000,
		ProtocolConfig: ProtocolConfig{
			Device:     "/dev/fake",
			BaudRate:   9600,
			DataBits:   8,
			Parity:     0,
			StopBits:   1,
			TimeoutMs:  100,
			IntervalMs: 1,
		},
		Variables: []*collection.Variable{
			{Name: "temperature", DataType: "INT16", Address: "1!1.0", AccessMode: collection.AccessModeRead},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestPollReadsRegister(t *testing.T) {
	data := []byte{0x34, 0x12}
	for i := range data {
		data[i] = mirrorByte(data[i])
	}
	response := []byte{stx, '0', '0', '1', '2', '4', '0', '0', '0', '0', '2'}
	response = append(response, data...)
	response = append(response, etx)
	response = append(response, checksum(response))

	port := &fakePort{responses: [][]byte{response}}
	withFakePort(t, port)
	item := newTestCollection(t)
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Stop(context.Background()) })

	if ok := item.poll(context.Background()); !ok {
		t.Fatal("poll stopped unexpectedly")
	}
	if status := item.GetCollectStatus(); status != collection.CollectStatusCollecting {
		t.Fatalf("status=%s, want collecting", status)
	}
	value, ok := item.GetVariableValue("temperature")
	if !ok || value != int16(0x1234) {
		t.Fatalf("value=%#v, want int16(0x1234)", value)
	}
}

func TestPollMarksUnconnectedOnReadFailure(t *testing.T) {
	port := &fakePort{readErr: io.ErrUnexpectedEOF}
	withFakePort(t, port)
	item := newTestCollection(t)
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Stop(context.Background()) })

	if ok := item.poll(context.Background()); ok {
		t.Fatal("poll should stop when the serial link fails")
	}
	if status := item.GetCollectStatus(); status != collection.CollectStatusUnconnected {
		t.Fatalf("status=%s, want unconnected", status)
	}
}

func TestConnectRequiresDevice(t *testing.T) {
	item := &Collection{}
	item.SetCollectionMeta(&collection.CollectionMeta{})
	err := item.Connect(context.Background())
	if !errors.Is(err, collection.ErrInvalidConfig) {
		t.Fatalf("err=%v, want ErrInvalidConfig", err)
	}
}
