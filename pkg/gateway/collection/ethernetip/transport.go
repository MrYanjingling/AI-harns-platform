package ethernetip

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
	commandRegisterSession = 0x0065
	commandSendRRData      = 0x006F
	commandSendUnitData    = 0x0070

	itemNullAddress      = 0x0000
	itemConnectedAddress = 0x00A1
	itemConnectedData    = 0x00B1
	itemUnconnectedData  = 0x00B2

	encapsulationHeaderSize = 24
	commandHeaderSize       = 8
	commandHeaderTimeout    = 10

	// Forward Open connection parameters copied from Neuron's CIP driver:
	// 2 s RPI and class-3 transport.
	forwardOpenRPI = 2_000_000
)

const cipTimeout = 2 * time.Second

type cipClient struct {
	conn    net.Conn
	timeout time.Duration

	mu             sync.Mutex
	session        uint32
	otConnectionID uint32
	context        uint64
	sequence       uint16
	closed         bool
}

func newCIPClient(ctx context.Context, endpoint string, timeout time.Duration) (*cipClient, error) {
	if timeout <= 0 || timeout > cipTimeout {
		timeout = cipTimeout
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	client := &cipClient{conn: conn, timeout: timeout}
	if err := client.registerSession(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.forwardOpen(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *cipClient) registerSession(ctx context.Context) error {
	_, _, _, session, err := c.request(ctx, commandRegisterSession, 0, []byte{1, 0, 0, 0})
	if err != nil {
		return err
	}
	if session == 0 {
		return fmt.Errorf("ethernet/ip register session returned zero handle")
	}
	c.session = session
	return nil
}

func (c *cipClient) forwardOpen(ctx context.Context) error {
	request := make([]byte, 0, 6+len(forwardOpenBody()))
	request = append(request, cipServiceForwardOpen, 2, 0x20, 0x06, 0x24, 0x01)
	request = append(request, forwardOpenBody()...)
	response, _, _, err := c.sendRRData(ctx, request)
	if err != nil {
		return fmt.Errorf("ethernet/ip forward open: %w", err)
	}
	body, err := parseCIPReply(response, cipServiceForwardOpen)
	if err != nil {
		return fmt.Errorf("ethernet/ip forward open: %w", err)
	}
	if len(body) < 26 {
		return fmt.Errorf("ethernet/ip forward open response is too short")
	}
	connectionID := binary.LittleEndian.Uint32(body)
	if connectionID == 0 {
		return fmt.Errorf("ethernet/ip forward open returned zero connection id")
	}
	c.otConnectionID = connectionID
	return nil
}

// forwardOpenBody matches Neuron's cip_forward_open_req_wrap byte for byte.
func forwardOpenBody() []byte {
	body := make([]byte, 40)
	body[0] = 0x05 // priority/time tick
	body[1] = 0xF7 // timeout ticks
	body[2], body[5] = 0x02, 0x80
	body[6], body[8], body[9] = 0x01, 0xFE, 0x80
	body[10] = 0x02
	body[12], body[13] = 0x1B, 0x05
	binary.LittleEndian.PutUint32(body[14:], uint32(time.Now().UnixMilli()))
	body[18] = 0x02
	binary.LittleEndian.PutUint32(body[22:], forwardOpenRPI)
	body[26], body[27] = 0xF8, 0x43
	binary.LittleEndian.PutUint32(body[28:], forwardOpenRPI)
	body[32], body[33] = 0xF8, 0x43
	body[34] = 0xA3
	body[35] = 0x02
	copy(body[36:], []byte{0x20, 0x02, 0x24, 0x01})
	return body
}

// ReadTag sends one connected Read Tag request and returns the tag bytes after
// the CIP type word.
func (c *cipClient) ReadTag(ctx context.Context, address parsedAddress) ([]byte, []byte, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sequence++
	request, err := readRequest(c.sequence, address)
	if err != nil {
		return nil, nil, nil, err
	}
	response, wireRequest, wireResponse, err := c.sendUnitData(ctx, request)
	if err != nil {
		return nil, wireRequest, wireResponse, err
	}
	data, err := parseReadResponse(response, address)
	return data, wireRequest, wireResponse, err
}

// WriteTag sends one connected Write Tag request.
func (c *cipClient) WriteTag(ctx context.Context, address parsedAddress, data []byte) ([]byte, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sequence++
	request, err := writeRequest(c.sequence, address, data)
	if err != nil {
		return nil, nil, err
	}
	response, wireRequest, wireResponse, err := c.sendUnitData(ctx, request)
	if err != nil {
		return wireRequest, wireResponse, err
	}
	return wireRequest, wireResponse, parseWriteResponse(response)
}

func (c *cipClient) sendRRData(ctx context.Context, cipRequest []byte) ([]byte, []byte, []byte, error) {
	payload := make([]byte, 0, 16+len(cipRequest))
	payload = append(payload, commandHeader(2)...)
	payload = appendItem(payload, itemNullAddress, nil)
	payload = appendItem(payload, itemUnconnectedData, cipRequest)
	response, wireRequest, wireResponse, _, err := c.request(ctx, commandSendRRData, c.session, payload)
	if err != nil {
		return nil, wireRequest, wireResponse, err
	}
	data, err := parseItemData(response, itemUnconnectedData)
	return data, wireRequest, wireResponse, err
}

func (c *cipClient) sendUnitData(ctx context.Context, cipRequest []byte) ([]byte, []byte, []byte, error) {
	addressValue := le32(c.otConnectionID)
	payload := make([]byte, 0, 16+len(cipRequest))
	payload = append(payload, commandHeader(2)...)
	payload = appendItem(payload, itemConnectedAddress, addressValue)
	payload = appendItem(payload, itemConnectedData, cipRequest)
	response, wireRequest, wireResponse, _, err := c.request(ctx, commandSendUnitData, c.session, payload)
	if err != nil {
		return nil, wireRequest, wireResponse, err
	}
	data, err := parseItemData(response, itemConnectedData)
	if err != nil {
		return nil, wireRequest, wireResponse, err
	}
	if len(data) < 2 {
		return nil, wireRequest, wireResponse, fmt.Errorf("ethernet/ip connected response is too short")
	}
	sequence := binary.LittleEndian.Uint16(data)
	if sequence != c.sequence {
		return nil, wireRequest, wireResponse, fmt.Errorf("ethernet/ip connected sequence %d does not match %d", sequence, c.sequence)
	}
	return data[2:], wireRequest, wireResponse, nil
}

func commandHeader(itemCount uint16) []byte {
	header := make([]byte, commandHeaderSize)
	binary.LittleEndian.PutUint16(header[4:], commandHeaderTimeout)
	binary.LittleEndian.PutUint16(header[6:], itemCount)
	return header
}

func appendItem(payload []byte, itemType uint16, data []byte) []byte {
	payload = append(payload, byte(itemType), byte(itemType>>8), byte(len(data)), byte(len(data)>>8))
	return append(payload, data...)
}

func parseItemData(payload []byte, want uint16) ([]byte, error) {
	if len(payload) < commandHeaderSize {
		return nil, fmt.Errorf("ethernet/ip command header is too short")
	}
	count := int(binary.LittleEndian.Uint16(payload[6:]))
	offset := commandHeaderSize
	for index := 0; index < count; index++ {
		if offset+4 > len(payload) {
			return nil, fmt.Errorf("ethernet/ip command item is too short")
		}
		itemType := binary.LittleEndian.Uint16(payload[offset:])
		itemLength := int(binary.LittleEndian.Uint16(payload[offset+2:]))
		offset += 4
		if offset+itemLength > len(payload) {
			return nil, fmt.Errorf("ethernet/ip command item length is invalid")
		}
		data := payload[offset : offset+itemLength]
		offset += itemLength
		if itemType == want {
			return data, nil
		}
	}
	return nil, fmt.Errorf("ethernet/ip command response has no item 0x%04X", want)
}

func (c *cipClient) request(ctx context.Context, command, session uint32, payload []byte) ([]byte, []byte, []byte, uint32, error) {
	if c.conn == nil || c.closed {
		return nil, nil, nil, 0, fmt.Errorf("ethernet/ip connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, 0, err
	}
	c.context++
	wire := make([]byte, encapsulationHeaderSize+len(payload))
	binary.LittleEndian.PutUint16(wire[0:], uint16(command))
	binary.LittleEndian.PutUint16(wire[2:], uint16(len(payload)))
	binary.LittleEndian.PutUint32(wire[4:], session)
	binary.LittleEndian.PutUint64(wire[12:], c.context)
	copy(wire[encapsulationHeaderSize:], payload)
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, nil, nil, 0, err
	}
	if _, err := c.conn.Write(wire); err != nil {
		return nil, wire, nil, 0, err
	}
	header := make([]byte, encapsulationHeaderSize)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, wire, nil, 0, err
	}
	if responseCommand := binary.LittleEndian.Uint16(header[:2]); responseCommand != uint16(command) {
		return nil, wire, nil, 0, fmt.Errorf("ethernet/ip encapsulation command mismatch")
	}
	length := int(binary.LittleEndian.Uint16(header[2:]))
	response := make([]byte, length)
	if _, err := io.ReadFull(c.conn, response); err != nil {
		return nil, wire, nil, 0, err
	}
	status := binary.LittleEndian.Uint32(header[8:])
	if status != 0 {
		return nil, wire, response, 0, fmt.Errorf("ethernet/ip encapsulation status 0x%X", status)
	}
	return response, wire, append(header, response...), binary.LittleEndian.Uint32(header[4:]), nil
}

func (c *cipClient) Available() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.closed
}

func (c *cipClient) Close() error {
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
