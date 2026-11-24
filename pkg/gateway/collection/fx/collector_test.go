package fx

import (
	"context"
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

func (p *fakePort) Write([]byte) (int, error)          { return 1, nil }
func (p *fakePort) Close() error                       { return nil }
func (p *fakePort) SetReadTimeout(time.Duration) error { return nil }

func withFakePort(t *testing.T, port *fakePort) {
	t.Helper()
	original := openSerialPort
	openSerialPort = func(string, *serial.Mode) (serialPort, error) { return port, nil }
	t.Cleanup(func() { openSerialPort = original })
}

func fxResponse(data []byte) []byte {
	frame := []byte{stx}
	frame = append(frame, bin2hex(data)...)
	frame = append(frame, etx)
	frame = append(frame, checksum(frame[1:])...)
	return frame
}

func TestPollReadsWordAndBit(t *testing.T) {
	port := &fakePort{responses: [][]byte{fxResponse([]byte{0x34, 0x12}), fxResponse([]byte{0x01})}}
	withFakePort(t, port)
	item, err := NewCollection(&CreateReq{
		Name:             "fx-test",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Device: "/dev/fake", TimeoutMs: 100, IntervalMs: 1},
		Variables: []*collection.Variable{
			{Name: "d0", Address: "D0", DataType: "INT16", AccessMode: collection.AccessModeRead},
			{Name: "m0", Address: "M0", DataType: "BIT", AccessMode: collection.AccessModeRead},
		},
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
	value, _ := item.GetVariableValue("d0")
	if value != int16(0x1234) {
		t.Fatalf("d0=%#v", value)
	}
	bit, _ := item.GetVariableValue("m0")
	if bit != true {
		t.Fatalf("m0=%#v", bit)
	}
}

func TestPollMarksUnconnectedOnLinkFailure(t *testing.T) {
	port := &fakePort{readErr: io.ErrUnexpectedEOF}
	withFakePort(t, port)
	item, err := NewCollection(&CreateReq{
		Name:             "fx-fail",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Device: "/dev/fake", TimeoutMs: 100, IntervalMs: 1},
		Variables:        []*collection.Variable{{Name: "d0", Address: "D0", DataType: "INT16", AccessMode: collection.AccessModeRead}},
	})
	if err != nil {
		t.Fatal(err)
	}
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
