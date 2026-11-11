package modbusrtu

import (
	"context"
	"io"
	"sync"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection/connpool"
)

type serialPort interface {
	io.ReadWriteCloser
	SetReadTimeout(time.Duration) error
}

var openSerialPort = func(location string, mode *serial.Mode) (serialPort, error) {
	return serial.Open(location, mode)
}

type serialClient struct {
	port serialPort
	mu   sync.Mutex
}

func (c *serialClient) Available() bool { return c != nil && c.port != nil }

func (c *serialClient) Close() error {
	if c == nil || c.port == nil {
		return nil
	}
	return c.port.Close()
}

func newSerialPool(ctx context.Context, configModbus serial.Mode, location string) (*connpool.Pool, error) {
	return connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		port, err := openSerialPort(location, &configModbus)
		if err != nil {
			return nil, err
		}
		return &serialClient{port: port}, nil
	})
}
