package ads

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	adsCommandRead  = 0x0002
	adsCommandWrite = 0x0003

	adsStateFlagRequest = 0x0004

	amsHeaderSize = 32
	amsTCPHeader  = 6
)

// ADSError is a non-zero ADS result code returned by the target.
type ADSError struct {
	Operation string
	Code      uint32
}

func (e *ADSError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("ads %s failed with code 0x%X", e.Operation, e.Code)
}

func adsResultError(operation string, data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("ads %s response is too short", operation)
	}
	code := binary.LittleEndian.Uint32(data)
	if code == 0 {
		return nil
	}
	return &ADSError{Operation: operation, Code: code}
}

type adsClient struct {
	conn       net.Conn
	timeout    time.Duration
	target     [6]byte
	targetPort uint16
	source     [6]byte
	sourcePort uint16

	mu     sync.Mutex
	invoke uint32
	closed bool
}

func newADSClient(ctx context.Context, endpoint string, target [6]byte, targetPort uint16, source [6]byte, sourcePort uint16, timeout time.Duration) (*adsClient, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = time.Duration(DefaultTimeoutMs) * time.Millisecond
	}
	return &adsClient{
		conn:       conn,
		timeout:    timeout,
		target:     target,
		targetPort: targetPort,
		source:     source,
		sourcePort: sourcePort,
	}, nil
}

func (c *adsClient) read(ctx context.Context, indexGroup, indexOffset uint32, length int) ([]byte, []byte, []byte, error) {
	if length < 0 {
		return nil, nil, nil, fmt.Errorf("ads read length is negative")
	}
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:], indexGroup)
	binary.LittleEndian.PutUint32(data[4:], indexOffset)
	binary.LittleEndian.PutUint32(data[8:], uint32(length))
	response, wireRequest, wireResponse, err := c.request(ctx, adsCommandRead, data)
	if err != nil {
		return nil, wireRequest, wireResponse, err
	}
	if len(response) < 8 {
		return nil, wireRequest, wireResponse, fmt.Errorf("ads read response is too short")
	}
	if err := adsResultError("read", response); err != nil {
		return nil, wireRequest, wireResponse, err
	}
	responseLength := int(binary.LittleEndian.Uint32(response[4:]))
	if responseLength < 0 || 8+responseLength > len(response) {
		return nil, wireRequest, wireResponse, fmt.Errorf("ads read response length is invalid")
	}
	return response[8 : 8+responseLength], wireRequest, wireResponse, nil
}

func (c *adsClient) write(ctx context.Context, indexGroup, indexOffset uint32, value []byte) ([]byte, []byte, []byte, error) {
	data := make([]byte, 12+len(value))
	binary.LittleEndian.PutUint32(data[0:], indexGroup)
	binary.LittleEndian.PutUint32(data[4:], indexOffset)
	binary.LittleEndian.PutUint32(data[8:], uint32(len(value)))
	copy(data[12:], value)
	response, wireRequest, wireResponse, err := c.request(ctx, adsCommandWrite, data)
	if err != nil {
		return nil, wireRequest, wireResponse, err
	}
	if err := adsResultError("write", response); err != nil {
		return nil, wireRequest, wireResponse, err
	}
	return response, wireRequest, wireResponse, nil
}

func (c *adsClient) request(ctx context.Context, command uint16, data []byte) ([]byte, []byte, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil || c.closed {
		return nil, nil, nil, fmt.Errorf("ads connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	c.invoke++
	invoke := c.invoke

	amsLength := amsHeaderSize + len(data)
	wire := make([]byte, amsTCPHeader+amsLength)
	binary.LittleEndian.PutUint32(wire[2:], uint32(amsLength))
	offset := amsTCPHeader
	copy(wire[offset:], c.target[:])
	binary.LittleEndian.PutUint16(wire[offset+6:], c.targetPort)
	copy(wire[offset+8:], c.source[:])
	binary.LittleEndian.PutUint16(wire[offset+14:], c.sourcePort)
	binary.LittleEndian.PutUint16(wire[offset+16:], command)
	binary.LittleEndian.PutUint16(wire[offset+18:], adsStateFlagRequest)
	binary.LittleEndian.PutUint32(wire[offset+20:], uint32(len(data)))
	binary.LittleEndian.PutUint32(wire[offset+28:], invoke)
	copy(wire[offset+amsHeaderSize:], data)

	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, wire, nil, err
	}
	if _, err := c.conn.Write(wire); err != nil {
		return nil, wire, nil, err
	}
	header := make([]byte, amsTCPHeader)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, wire, nil, err
	}
	amsResponseLength := int(binary.LittleEndian.Uint32(header[2:]))
	if amsResponseLength < amsHeaderSize || amsResponseLength > 1024*1024 {
		return nil, wire, nil, fmt.Errorf("invalid ads response length %d", amsResponseLength)
	}
	response := make([]byte, amsResponseLength)
	if _, err := io.ReadFull(c.conn, response); err != nil {
		return nil, wire, nil, err
	}
	if binary.LittleEndian.Uint16(response[16:]) != command {
		return nil, wire, response, fmt.Errorf("ads response command mismatch")
	}
	if code := binary.LittleEndian.Uint32(response[24:]); code != 0 {
		return nil, wire, response, &ADSError{Operation: "ams", Code: code}
	}
	if responseInvoke := binary.LittleEndian.Uint32(response[28:]); responseInvoke != invoke {
		return nil, wire, response, fmt.Errorf("ads response invoke id mismatch")
	}
	length := int(binary.LittleEndian.Uint32(response[20:]))
	if length < 0 || amsHeaderSize+length > len(response) {
		return nil, wire, response, fmt.Errorf("ads response data length is invalid")
	}
	return response[amsHeaderSize : amsHeaderSize+length], wire, response, nil
}

func (c *adsClient) Available() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.closed
}

func (c *adsClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	c.closed = true
	conn := c.conn
	c.conn = nil
	return conn.Close()
}
