package hostlinkcmode

import (
	"context"
	"io"
	"testing"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection"
)

func TestInvalidCalculationKeepsCollectionRunning(t *testing.T) {
	port := &fakeSerialPort{response: finishFrame("@00RD00000A00")}
	originalOpen := openSerialPort
	openSerialPort = func(string, *serial.Mode) (serialPort, error) {
		return port, nil
	}
	t.Cleanup(func() { openSerialPort = originalOpen })

	item, err := NewCollection(&CreateReq{
		Name:             "hostlink-invalid-calculation",
		CollectorCycleMs: 1000,
		ProtocolConfig: ProtocolConfig{
			Location:     "COM1",
			IntervalMs:   1,
			MaxReadWords: DefaultMaxReadWords,
		},
		Variables: []*collection.Variable{
			{Name: "temperature", DataType: "int16", Address: "00!D0000", AccessMode: collection.AccessModeRead},
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
	if port.closed {
		t.Fatal("serial port must stay open on calculation errors")
	}
}

type fakeSerialPort struct {
	response []byte
	offset   int
	closed   bool
}

func (p *fakeSerialPort) Read(buffer []byte) (int, error) {
	if p.offset >= len(p.response) {
		return 0, io.EOF
	}
	read := copy(buffer, p.response[p.offset:])
	p.offset += read
	return read, nil
}

func (p *fakeSerialPort) Write(buffer []byte) (int, error) {
	p.offset = 0
	return len(buffer), nil
}

func (p *fakeSerialPort) Close() error {
	p.closed = true
	return nil
}

func (p *fakeSerialPort) SetReadTimeout(time.Duration) error {
	return nil
}

func TestDecodeFrameKeepsSuccessfulValues(t *testing.T) {
	frame := &hostlinkFrame{
		Address: parsedAddress{length: 3},
		Variables: []*hostlinkFrameVariable{
			{Variable: &collection.Variable{Name: "a", DataType: "int16"}, Address: parsedAddress{length: 1}},
			{Variable: &collection.Variable{Name: "b", DataType: "int64"}, Address: parsedAddress{length: 4}},
		},
	}
	values, err := decodeFrame(frame, response{data: "000A00140020"})
	if err == nil {
		t.Fatal("expected decode error")
	}
	if len(values) != 1 || values[0].Variable.Name != "a" || values[0].Value != int16(10) {
		t.Fatalf("unexpected decoded values %+v", values)
	}
}
