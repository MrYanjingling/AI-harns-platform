package srtp

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

const (
	headerSize       = 56
	dataOffset       = 0x2c
	simpleDataOffset = 56

	serviceRead  = 4
	serviceWrite = 7
	successType  = 0xD4
)

var requestBase = []byte{
	0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0xC0,
	0x00, 0x00, 0x00, 0x00, 0x10, 0x0E, 0x00, 0x00,
	0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

var secondHandshake = []byte{
	0x08, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0xC0,
	0x00, 0x00, 0x00, 0x00, 0x10, 0x0E, 0x00, 0x00,
	0x01, 0x01, 0x4F, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

type client struct {
	conn      net.Conn
	timeout   time.Duration
	mu        sync.Mutex
	handshake bool
	simple    bool
}

func openClient(ctx context.Context, host string, port int, timeout time.Duration) (*client, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid SRTP port")
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	if err != nil {
		return nil, err
	}
	item := &client{conn: conn, timeout: timeout}
	if err := item.doHandshake(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return item, nil
}

func (c *client) Available() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *client) doHandshake() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.handshake {
		return nil
	}
	if err := c.writeLocked(make([]byte, headerSize)); err != nil {
		return err
	}
	reply := make([]byte, headerSize)
	if err := c.readLocked(reply); err != nil {
		return err
	}
	if reply[0] != 1 {
		return fmt.Errorf("SRTP first handshake failed with byte 0x%X", reply[0])
	}
	if err := c.writeLocked(secondHandshake); err != nil {
		return err
	}
	if err := c.readLocked(reply); err != nil {
		return err
	}
	if reply[0] != 1 && reply[0] != 2 {
		return fmt.Errorf("SRTP second handshake failed with byte 0x%X", reply[0])
	}
	c.simple = reply[0] == 1
	c.handshake = true
	return nil
}

func (c *client) Read(address Address) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.handshake {
		return nil, fmt.Errorf("SRTP handshake is not complete")
	}
	request := buildRequest(serviceRead, address, nil, address.ByteLength)
	if err := c.writeLocked(request); err != nil {
		return nil, err
	}
	if c.simple {
		size := simpleDataOffset + int(address.ByteLength)
		response := make([]byte, size)
		if err := c.readLocked(response); err != nil {
			return nil, err
		}
		return decodeValue(response[simpleDataOffset:], address)
	}
	size := headerSize
	if address.ByteLength >= 7 {
		size = dataOffset + int(address.ByteLength)
	}
	response := make([]byte, size)
	if err := c.readLocked(response); err != nil {
		return nil, err
	}
	if err := checkResponse(response); err != nil {
		return nil, err
	}
	return decodeValue(response[dataOffset:dataOffset+int(address.ByteLength)], address)
}

func (c *client) Write(address Address, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.handshake {
		return fmt.Errorf("SRTP handshake is not complete")
	}
	if address.IsWord && address.DataType == "bit" {
		current, err := c.readLockedAddress(address)
		if err != nil {
			return err
		}
		number := binary.LittleEndian.Uint16(current)
		bit := uint16(1) << uint(address.Bit)
		if truthy(value) {
			number |= bit
		} else {
			number &^= bit
		}
		value = number
	}
	data, err := encodeValue(value, address)
	if err != nil {
		return err
	}
	request := buildRequest(serviceWrite, address, data, address.ByteLength)
	if c.simple {
		request = buildSimpleWriteRequest(address, data)
	}
	if err := c.writeLocked(request); err != nil {
		return err
	}
	response := make([]byte, headerSize)
	if err := c.readLocked(response); err != nil {
		return err
	}
	if c.simple {
		return nil
	}
	return checkResponse(response)
}

func (c *client) readLockedAddress(address Address) ([]byte, error) {
	request := buildRequest(serviceRead, address, nil, address.ByteLength)
	if err := c.writeLocked(request); err != nil {
		return nil, err
	}
	if c.simple {
		size := simpleDataOffset + int(address.ByteLength)
		response := make([]byte, size)
		if err := c.readLocked(response); err != nil {
			return nil, err
		}
		return response[simpleDataOffset:], nil
	}
	size := headerSize
	if address.ByteLength >= 7 {
		size = dataOffset + int(address.ByteLength)
	}
	response := make([]byte, size)
	if err := c.readLocked(response); err != nil {
		return nil, err
	}
	if err := checkResponse(response); err != nil {
		return nil, err
	}
	return response[dataOffset : dataOffset+int(address.ByteLength)], nil
}

func (c *client) writeLocked(data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("SRTP connection is closed")
	}
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.conn.Write(data)
	return err
}

func (c *client) readLocked(data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("SRTP connection is closed")
	}
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := io.ReadFull(c.conn, data)
	return err
}

func buildRequest(service uint8, address Address, data []byte, byteLength uint16) []byte {
	frame := append([]byte(nil), requestBase...)
	frame[0x2a] = service
	frame[0x2b] = address.Memory
	binary.LittleEndian.PutUint16(frame[0x2c:], address.Start)
	binary.LittleEndian.PutUint16(frame[0x2e:], address.Quantity)
	if service == serviceWrite && len(data) > 8 {
		frame = append(frame, make([]byte, len(data))...)
	}
	copy(frame[0x30:], data)
	return frame
}

func buildSimpleWriteRequest(address Address, data []byte) []byte {
	frame := buildRequest(serviceWrite, address, nil, address.ByteLength)
	frame[0x32] = serviceWrite
	frame[0x33] = address.Memory
	binary.LittleEndian.PutUint16(frame[0x34:], address.Start)
	binary.LittleEndian.PutUint16(frame[0x36:], address.Quantity)
	frame = append(frame, make([]byte, len(data))...)
	copy(frame[0x38:], data)
	return frame
}

func checkResponse(response []byte) error {
	if len(response) < headerSize {
		return fmt.Errorf("SRTP response is too short")
	}
	if response[0x1f] != successType {
		return fmt.Errorf("SRTP PLC rejected request: status=%d minor=%d", response[0x2a], response[0x2b])
	}
	return nil
}

func decodeValue(data []byte, address Address) (any, error) {
	if len(data) < int(address.ByteLength) {
		return nil, fmt.Errorf("SRTP response data is too short")
	}
	if address.StringLen > 0 {
		text := data[:address.StringLen]
		end := len(text)
		for i, item := range text {
			if item == 0 {
				end = i
				break
			}
		}
		return string(text[:end]), nil
	}
	switch address.DataType {
	case "bool", "bit":
		if address.IsWord && address.Bit > 0 {
			word := binary.LittleEndian.Uint16(data)
			return (word>>uint(address.Bit))&1 != 0, nil
		}
		return decodeBool(data, address)
	case "int8":
		return int8(data[0]), nil
	case "uint8":
		return data[0], nil
	case "int16":
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "uint16":
		return binary.LittleEndian.Uint16(data), nil
	case "int32":
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "uint32":
		return binary.LittleEndian.Uint32(data), nil
	case "int64":
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "uint64":
		return binary.LittleEndian.Uint64(data), nil
	case "float32":
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "float64":
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	default:
		return nil, fmt.Errorf("unsupported SRTP data type")
	}
}

func decodeBool(data []byte, address Address) (bool, error) {
	if address.IsWord {
		return binary.LittleEndian.Uint16(data) != 0, nil
	}
	return data[0]&1 != 0, nil
}

func encodeValue(value any, address Address) ([]byte, error) {
	data := make([]byte, address.ByteLength)
	switch address.DataType {
	case "bool", "bit":
		if address.IsWord {
			number, ok := numericUint(value)
			if !ok {
				if !truthy(value) {
					return data, nil
				}
				number = 1
			}
			binary.LittleEndian.PutUint16(data, uint16(number))
			return data, nil
		}
		if truthy(value) {
			data[0] = 1
		}
		return data, nil
	case "int8":
		number, ok := numericInt(value)
		if !ok {
			return nil, fmt.Errorf("SRTP int8 value is invalid")
		}
		data[0] = byte(number)
	case "uint8":
		number, ok := numericUint(value)
		if !ok {
			return nil, fmt.Errorf("SRTP uint8 value is invalid")
		}
		data[0] = byte(number)
	case "int16":
		number, ok := numericInt(value)
		if !ok {
			return nil, fmt.Errorf("SRTP int16 value is invalid")
		}
		binary.LittleEndian.PutUint16(data, uint16(number))
	case "uint16":
		number, ok := numericUint(value)
		if !ok {
			return nil, fmt.Errorf("SRTP uint16 value is invalid")
		}
		binary.LittleEndian.PutUint16(data, uint16(number))
	case "int32":
		number, ok := numericInt(value)
		if !ok {
			return nil, fmt.Errorf("SRTP int32 value is invalid")
		}
		binary.LittleEndian.PutUint32(data, uint32(number))
	case "uint32":
		number, ok := numericUint(value)
		if !ok {
			return nil, fmt.Errorf("SRTP uint32 value is invalid")
		}
		binary.LittleEndian.PutUint32(data, uint32(number))
	case "int64":
		number, ok := numericInt(value)
		if !ok {
			return nil, fmt.Errorf("SRTP int64 value is invalid")
		}
		binary.LittleEndian.PutUint64(data, uint64(number))
	case "uint64":
		number, ok := numericUint(value)
		if !ok {
			return nil, fmt.Errorf("SRTP uint64 value is invalid")
		}
		binary.LittleEndian.PutUint64(data, number)
	case "float32":
		number, ok := numericFloat(value)
		if !ok {
			return nil, fmt.Errorf("SRTP float value is invalid")
		}
		binary.LittleEndian.PutUint32(data, math.Float32bits(float32(number)))
	case "float64":
		number, ok := numericFloat(value)
		if !ok {
			return nil, fmt.Errorf("SRTP double value is invalid")
		}
		binary.LittleEndian.PutUint64(data, math.Float64bits(number))
	case "string":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("SRTP string value is invalid")
		}
		if len(text) > address.StringLen {
			return nil, fmt.Errorf("SRTP string is too long")
		}
		copy(data, text)
	default:
		return nil, fmt.Errorf("unsupported SRTP data type")
	}
	return data, nil
}

func numericInt(value any) (int64, bool) {
	switch number := value.(type) {
	case int:
		return int64(number), true
	case int8:
		return int64(number), true
	case int16:
		return int64(number), true
	case int32:
		return int64(number), true
	case int64:
		return number, true
	default:
		return 0, false
	}
}
func numericUint(value any) (uint64, bool) {
	switch number := value.(type) {
	case int:
		return uint64(number), number >= 0
	case uint8:
		return uint64(number), true
	case uint16:
		return uint64(number), true
	case uint32:
		return uint64(number), true
	case uint64:
		return number, true
	default:
		return 0, false
	}
}
func numericFloat(value any) (float64, bool) {
	switch number := value.(type) {
	case float32:
		return float64(number), true
	case float64:
		return number, true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint64:
		return float64(number), true
	default:
		return 0, false
	}
}
func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case int:
		return typed != 0
	case int8:
		return typed != 0
	case int16:
		return typed != 0
	case int32:
		return typed != 0
	case int64:
		return typed != 0
	case uint8:
		return typed != 0
	case uint16:
		return typed != 0
	case uint32:
		return typed != 0
	case uint64:
		return typed != 0
	default:
		return false
	}
}
