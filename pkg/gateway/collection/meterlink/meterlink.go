// Package meterlink provides shared serial/TCP framing helpers for meter protocols.
package meterlink

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
)

// Link is one serial or TCP connection with a read deadline.
type Link interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
}

type serialLink struct {
	serial.Port
}

func (s *serialLink) SetDeadline(t time.Time) error {
	timeout := time.Until(t)
	if timeout < 0 {
		timeout = 0
	}
	return s.SetReadTimeout(timeout)
}

type Client struct {
	link    Link
	timeout time.Duration
}

// NewClient wraps an already opened link with the shared timeout.
func NewClient(link Link, timeout time.Duration) *Client {
	return &Client{link: link, timeout: timeout}
}

func (c *Client) Available() bool { return c != nil && c.link != nil }
func (c *Client) Read(data []byte) (int, error) {
	if c == nil || c.link == nil {
		return 0, fmt.Errorf("meter link is not connected")
	}
	return c.link.Read(data)
}
func (c *Client) Write(data []byte) (int, error) {
	if c == nil || c.link == nil {
		return 0, fmt.Errorf("meter link is not connected")
	}
	return c.link.Write(data)
}
func (c *Client) SetDeadline(deadline time.Time) error {
	if c == nil || c.link == nil {
		return fmt.Errorf("meter link is not connected")
	}
	return c.link.SetDeadline(deadline)
}
func (c *Client) Close() error {
	if c == nil || c.link == nil {
		return nil
	}
	return c.link.Close()
}

// Open creates one serial or TCP link. link=0 serial, link=1 TCP.
func Open(ctx context.Context, link int, location string, port int, timeout time.Duration, mode *serial.Mode) (Link, error) {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if link == 1 {
		address := location
		if !strings.Contains(address, ":") {
			address = net.JoinHostPort(location, fmt.Sprintf("%d", port))
		}
		dialer := net.Dialer{Timeout: timeout}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
	if mode == nil {
		mode = &serial.Mode{BaudRate: 9600, DataBits: 8, Parity: serial.EvenParity, StopBits: serial.OneStopBit}
	}
	handle, err := serial.Open(collection.NormalizeSerialPort(location), mode)
	if err != nil {
		return nil, err
	}
	return &serialLink{Port: handle}, nil
}

func NewPool(ctx context.Context, link int, location string, port int, timeout time.Duration, mode *serial.Mode) (*connpool.Pool, error) {
	return connpool.NewPool(ctx, 1, func(ctx context.Context) (connpool.Client, error) {
		item, err := Open(ctx, link, location, port, timeout, mode)
		if err != nil {
			return nil, err
		}
		return &Client{link: item, timeout: timeout}, nil
	})
}

func SerialMode(baud, data int, parity, stop string) *serial.Mode {
	if baud <= 0 {
		baud = 9600
	}
	if data <= 0 {
		data = 8
	}
	mode := &serial.Mode{BaudRate: baud, DataBits: data, Parity: serial.EvenParity, StopBits: serial.OneStopBit}
	switch strings.ToLower(strings.TrimSpace(parity)) {
	case "none", "n", "0", "noparity":
		mode.Parity = serial.NoParity
	case "odd", "o", "1", "oddparity":
		mode.Parity = serial.OddParity
	case "even", "e", "2", "evenparity":
		mode.Parity = serial.EvenParity
	}
	switch strings.TrimSpace(stop) {
	case "2":
		mode.StopBits = serial.TwoStopBits
	case "1.5":
		mode.StopBits = serial.OnePointFiveStopBits
	}
	return mode
}

func WriteFrame(client *Client, frame []byte, timeout time.Duration) error {
	if client == nil || client.link == nil {
		return fmt.Errorf("meter link is not connected")
	}
	if timeout <= 0 {
		timeout = client.timeout
	}
	if err := client.link.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	_, err := client.link.Write(frame)
	return err
}

// ReadFrame skips FE preamble bytes and returns one 68 ... 16 frame. It stops at the first 0x16 byte.
func ReadFrame(client *Client, timeout time.Duration, maxLen int) ([]byte, error) {
	return readFrame(client, timeout, maxLen, -1)
}

// ReadFrameWithLength reads a frame using the length byte at lengthIndex.
func ReadFrameWithLength(client *Client, timeout time.Duration, maxLen, lengthIndex int) ([]byte, error) {
	return readFrame(client, timeout, maxLen, lengthIndex)
}

// ReadFrameWithLengthLE reads a frame whose 2-byte little-endian length at lengthIndex is total-2.
func ReadFrameWithLengthLE(client *Client, timeout time.Duration, maxLen, lengthIndex int) ([]byte, error) {
	if client == nil || client.link == nil {
		return nil, fmt.Errorf("meter link is not connected")
	}
	if timeout <= 0 {
		timeout = client.timeout
	}
	if maxLen <= 0 {
		maxLen = 512
	}
	if err := client.link.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	buf := make([]byte, 0, 128)
	started := false
	one := make([]byte, 1)
	for len(buf) < maxLen {
		n, err := client.link.Read(one)
		if err != nil {
			return nil, err
		}
		if n != 1 {
			continue
		}
		value := one[0]
		if !started {
			if value != 0x68 {
				continue
			}
			started = true
		}
		buf = append(buf, value)
		if len(buf) < lengthIndex+2 {
			continue
		}
		total := int(binary.LittleEndian.Uint16(buf[lengthIndex:lengthIndex+2])) + 2
		if total < lengthIndex+2 || total > maxLen {
			return nil, fmt.Errorf("meter frame length mismatch")
		}
		if len(buf) == total {
			return buf, nil
		}
		if len(buf) > total {
			return nil, fmt.Errorf("meter frame length mismatch")
		}
	}
	return nil, fmt.Errorf("meter frame is too long")
}

func readFrame(client *Client, timeout time.Duration, maxLen, lengthIndex int) ([]byte, error) {
	if client == nil || client.link == nil {
		return nil, fmt.Errorf("meter link is not connected")
	}
	if timeout <= 0 {
		timeout = client.timeout
	}
	if maxLen <= 0 {
		maxLen = 512
	}
	if err := client.link.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	buf := make([]byte, 0, 128)
	started := false
	one := make([]byte, 1)
	for len(buf) < maxLen {
		n, err := client.link.Read(one)
		if err != nil {
			return nil, err
		}
		if n != 1 {
			continue
		}
		value := one[0]
		if !started {
			if value != 0x68 {
				continue
			}
			started = true
		}
		buf = append(buf, value)
		if lengthIndex >= 0 {
			if len(buf) <= lengthIndex {
				continue
			}
			total := lengthIndex + 1 + int(buf[lengthIndex]) + 2
			if len(buf) == total {
				return buf, nil
			}
			if len(buf) > total {
				return nil, fmt.Errorf("meter frame length mismatch")
			}
			continue
		}
		if len(buf) >= 2 && value == 0x16 {
			return buf, nil
		}
	}
	return nil, fmt.Errorf("meter frame is too long")
}

func ChecksumSum8(data []byte) byte {
	var sum byte
	for _, value := range data {
		sum += value
	}
	return sum
}

func EncodeBCDAddress(value uint64, size int) []byte {
	out := make([]byte, size)
	for index := 0; index < size; index++ {
		low := byte(value % 10)
		value /= 10
		high := byte(value % 10)
		value /= 10
		out[index] = high<<4 | low
	}
	return out
}

func DecodeBCD(data []byte) string {
	digits := make([]byte, 0, len(data)*2)
	for index := len(data) - 1; index >= 0; index-- {
		digits = append(digits, '0'+data[index]>>4, '0'+data[index]&0x0F)
	}
	return string(digits)
}

func Unmask(data []byte) []byte {
	out := make([]byte, len(data))
	for index, value := range data {
		out[index] = value - 0x33
	}
	return out
}

func Mask(data []byte) []byte {
	out := make([]byte, len(data))
	for index, value := range data {
		out[index] = value + 0x33
	}
	return out
}
