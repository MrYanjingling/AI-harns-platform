package mpi

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection/meterlink"
)

const maxFrameSize = 512

type mpiClient struct {
	link    *meterlink.Client
	timeout time.Duration
	station byte
	seq     byte
}

func openMPIClient(ctx context.Context, location string, station byte, timeout time.Duration, mode *serial.Mode) (*mpiClient, error) {
	link, err := meterlink.Open(ctx, 0, location, 0, timeout, mode)
	if err != nil {
		return nil, err
	}
	client := &mpiClient{link: meterlink.NewClient(link, timeout), timeout: timeout, station: station}
	if err := client.connect(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func (c *mpiClient) Available() bool { return c != nil && c.link != nil && c.link.Available() }
func (c *mpiClient) Close() error {
	if c == nil || c.link == nil {
		return nil
	}
	return c.link.Close()
}

func (c *mpiClient) connect() error {
	if err := c.initAdapter(); err != nil {
		return err
	}
	if err := c.connectPLC(); err != nil {
		return err
	}
	return c.setupPDU()
}

func (c *mpiClient) initAdapter() error {
	if err := c.sendByte(0x02); err != nil {
		return err
	}
	if err := c.recvByte(0x10); err != nil {
		return err
	}
	if err := c.write(withLRC([]byte{
		0x01, 0x03, 0x02, 0x17, 0x00, 0x9f, 0x01, 0x3c, 0x00, 0x90, 0x01, 0x14, 0x00,
		0x00, 0x14, 0x02, 0x00, 0x0f, 0x05, 0x01, 0x01, 0x03, 0x80,
	})); err != nil {
		return err
	}
	if err := c.recvByte(0x10); err != nil {
		return err
	}
	if err := c.recvByte(0x02); err != nil {
		return err
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	if _, err := c.recvFrame(); err != nil {
		return err
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	if err := c.sendByte(0x02); err != nil {
		return err
	}
	return c.recvByte(0x10)
}

func (c *mpiClient) connectPLC() error {
	connect := func() []byte {
		frame := []byte{0x00, 0x0d, 0x00, 0x03, 0xe0, 0x04, 0x00, 0x80, 0x00, 0x02,
			0x01, 0x06, 0x01, 0x00, 0x00, 0x01, 0x02, c.station, 0x01, 0x00}
		return withLRC(frame)
	}()
	if err := c.write(connect); err != nil {
		return err
	}
	if err := c.recvByte(0x10); err != nil {
		return err
	}
	if err := c.recvByte(0x02); err != nil {
		return err
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	response, err := c.recvFrame()
	if err != nil {
		return err
	}
	if len(response) <= 0x0f || response[0x0f] != c.station {
		return fmt.Errorf("MPI station mismatch")
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	if err := c.sendByte(0x02); err != nil {
		return err
	}
	if err := c.recvByte(0x10); err != nil {
		return err
	}
	if err := c.write(withLRC([]byte{0x00, 0x0c, 0x03, 0x03, 0x05, 0x01})); err != nil {
		return err
	}
	if err := c.recvByte(0x10); err != nil {
		return err
	}
	if err := c.recvByte(0x02); err != nil {
		return err
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	ack, err := c.recvFrame()
	if err != nil {
		return err
	}
	if len(ack) < 9 || string(ack[:9]) != string([]byte{0x00, 0x0c, 0x03, 0x03, 0x05, 0x01, 0x10, 0x03, 0x1b}) {
		return fmt.Errorf("MPI connect acknowledgement mismatch")
	}
	return c.sendByte(0x10)
}

func (c *mpiClient) setupPDU() error {
	if err := c.sendByte(0x02); err != nil {
		return err
	}
	if err := c.recvByte(0x10); err != nil {
		return err
	}
	if err := c.write(withLRC([]byte{
		0x00, 0x0c, 0x03, 0x03, 0xf1, 0x00, 0x32, 0x01, 0x00, 0x00, 0xff, 0xff, 0x00,
		0x08, 0x00, 0x00, 0xf0, 0x00, 0x00, 0x03, 0x00, 0x03, 0x01, 0x00,
	})); err != nil {
		return err
	}
	if err := c.recvByte(0x10); err != nil {
		return err
	}
	if err := c.recvByte(0x02); err != nil {
		return err
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	ack, err := c.recvFrame()
	if err != nil {
		return err
	}
	if len(ack) < 10 || string(ack[:10]) != string([]byte{0x00, 0x0c, 0x03, 0x03, 0xb0, 0x01, 0x00, 0x10, 0x03, 0xae}) {
		return fmt.Errorf("MPI PDU acknowledgement mismatch")
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	if err := c.recvByte(0x02); err != nil {
		return err
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	if _, err := c.recvFrame(); err != nil {
		return err
	}
	if err := c.sendByte(0x10); err != nil {
		return err
	}
	if err := c.sendByte(0x02); err != nil {
		return err
	}
	if err := c.recvByte(0x10); err != nil {
		return err
	}
	if err := c.write(withLRC([]byte{0x00, 0x0c, 0x03, 0x03, 0xb0, 0x01, 0x00})); err != nil {
		return err
	}
	return c.recvByte(0x10)
}

func (c *mpiClient) Read(address parsedAddress, dataType string) ([]byte, error) {
	if err := c.sendByte(0x02); err != nil {
		return nil, err
	}
	if err := c.recvByte(0x10); err != nil {
		return nil, err
	}
	if err := c.write(c.buildReadRequest(address)); err != nil {
		return nil, err
	}
	if err := c.recvByte(0x10); err != nil {
		return nil, err
	}
	if err := c.recvByte(0x02); err != nil {
		return nil, err
	}
	if err := c.sendByte(0x10); err != nil {
		return nil, err
	}
	ack, err := c.recvFrame()
	if err != nil {
		return nil, err
	}
	if err := c.sendByte(0x10); err != nil {
		return nil, err
	}
	if err := c.recvByte(0x02); err != nil {
		return nil, err
	}
	if err := c.sendByte(0x10); err != nil {
		return nil, err
	}
	response, err := c.recvFrame()
	if err != nil {
		return nil, err
	}
	out, err := parseReadResult(response)
	if err != nil {
		return nil, err
	}
	if err := c.sendByte(0x10); err != nil {
		return nil, err
	}
	if err := c.sendByte(0x02); err != nil {
		return nil, err
	}
	if err := c.recvByte(0x10); err != nil {
		return nil, err
	}
	if err := c.write(ack); err != nil {
		return nil, err
	}
	if err := c.recvByte(0x10); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mpiClient) buildReadRequest(address parsedAddress) []byte {
	c.seq++
	if c.seq == 0 {
		c.seq = 1
	}
	cmd := []byte{0x00, 0x0c, 0x03, 0x03, 0xf1}
	cmd = appendEscaped(cmd, c.seq)
	cmd = append(cmd, 0x32, 0x01, 0x00, 0x00, 0x33, 0x01, 0x00, 0x0e, 0x00, 0x00)
	cmd = append(cmd, 0x04, 0x01, 0x12, 0x0a, 0x10)
	cmd = append(cmd, 0x10)
	typ := address.Type
	start := uint16(address.Start) * 8
	if typ == typeBool {
		start += uint16(address.Bit)
	}
	if address.Area == areaC || address.Area == areaT || address.Area == areaIECC || address.Area == areaIECT || address.Area == areaIECHC {
		typ = address.Area
		start = address.Start
	}
	if address.Area == areaAI || address.Area == areaAQ {
		typ = typeWord
	}
	cmd = appendEscaped(cmd, typ)
	cmd = appendEscaped(cmd, byte(address.Length>>8))
	cmd = appendEscaped(cmd, byte(address.Length))
	cmd = appendEscaped(cmd, byte(address.DB>>8))
	cmd = appendEscaped(cmd, byte(address.DB))
	cmd = appendEscaped(cmd, address.Area)
	cmd = appendEscaped(cmd, 0x00)
	cmd = appendEscaped(cmd, byte(start>>8))
	cmd = appendEscaped(cmd, byte(start))
	return withLRC(cmd)
}

func appendEscaped(dst []byte, value byte) []byte {
	dst = append(dst, value)
	if value == 0x10 {
		dst = append(dst, 0x10)
	}
	return dst
}

func (c *mpiClient) write(data []byte) error {
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	for len(data) > 0 {
		n, err := c.link.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (c *mpiClient) sendByte(value byte) error {
	return c.write([]byte{value})
}

func (c *mpiClient) readByte() (byte, error) {
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}
	var data [1]byte
	if _, err := io.ReadFull(c.link, data[:]); err != nil {
		return 0, err
	}
	return data[0], nil
}

func (c *mpiClient) recvByte(expected byte) error {
	value, err := c.readByte()
	if err != nil {
		return err
	}
	if value != expected {
		return fmt.Errorf("MPI expected byte 0x%02X, got 0x%02X", expected, value)
	}
	return nil
}

func (c *mpiClient) recvFrame() ([]byte, error) {
	frame := make([]byte, 0, 64)
	var sum byte
	for len(frame) < maxFrameSize {
		value, err := c.readByte()
		if err != nil {
			return nil, err
		}
		frame = append(frame, value)
		if len(frame) >= 3 && frame[len(frame)-3] == 0x10 && frame[len(frame)-2] == 0x03 && value == sum {
			return frame, nil
		}
		sum ^= value
	}
	return nil, fmt.Errorf("MPI frame is too long")
}

func withLRC(payload []byte) []byte {
	frame := append(append([]byte{}, payload...), 0x10, 0x03)
	var sum byte
	for _, value := range frame {
		sum ^= value
	}
	return append(frame, sum)
}

func parseReadResult(frame []byte) ([]byte, error) {
	if len(frame) < 0x18 || frame[0] != 0x00 || frame[1] != 0x0c {
		return nil, fmt.Errorf("invalid MPI read response")
	}
	errorPosition := 0x14
	dataPosition := 0x16
	if frame[5] == 0x10 {
		errorPosition = 0x15
		dataPosition = 0x17
	}
	if errorPosition >= len(frame) || dataPosition >= len(frame)-3 {
		return nil, fmt.Errorf("short MPI read response")
	}
	if frame[errorPosition] != 0xff {
		return nil, fmt.Errorf("MPI item error 0x%02X", frame[errorPosition])
	}
	data := make([]byte, 0, len(frame))
	for index := dataPosition; index < len(frame)-3; index++ {
		if frame[index] == 0x10 && index+1 < len(frame)-3 && frame[index+1] == 0x10 {
			index++
		}
		data = append(data, frame[index])
	}
	if len(data) < 2 {
		return nil, fmt.Errorf("short MPI read data")
	}
	return data[2:], nil
}
