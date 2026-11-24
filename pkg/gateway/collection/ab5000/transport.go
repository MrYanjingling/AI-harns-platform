package ab5000

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
	enetCommandRegisterSession = 0x0065
	enetCommandSendRRData      = 0x006F
	enetCommandSendUnitData    = 0x0070

	enetItemNull              = 0x0000
	enetItemConnectionBased   = 0x00A1
	enetItemConnectedData     = 0x00B1
	enetItemUnconnectedData   = 0x00B2
	enetHeaderSize            = 24
	enetCommandHeaderSize     = 8
	enetMaxPayload            = 1 << 20
	enetForwardOpenService    = 0x54
	enetForwardOpenResponse   = 0xD4
	enetConnectedTimeoutTicks = 10
	enetForwardOpenBodySize   = 42
	enetForwardOpenPathSize   = 3
)

const cipTimeout = 2 * time.Second

type cipClient struct {
	conn    net.Conn
	timeout time.Duration
	slot    int

	mu             sync.Mutex
	session        uint32
	otConnectionID uint32
	toConnectionID uint32
	sequence       uint16
	context        uint64
	closed         bool
}

func newCIPClient(ctx context.Context, endpoint string, slot int, timeout time.Duration) (*cipClient, error) {
	if timeout <= 0 {
		timeout = cipTimeout
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	client := &cipClient{conn: conn, timeout: timeout, slot: slot}
	if err := client.registerSession(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.forwardOpen(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *cipClient) Available() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.closed && c.session != 0 && c.otConnectionID != 0
}

func (c *cipClient) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *cipClient) registerSession() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	request := buildEnetPacket(enetCommandRegisterSession, 0, 0, []byte{1, 0, 0, 0})
	header, _, err := c.exchangeEnet(request)
	if err != nil {
		return err
	}
	session := binary.LittleEndian.Uint32(header[4:8])
	if session == 0 {
		return fmt.Errorf("AB5000 register session returned zero handle")
	}
	c.session = session
	return nil
}

func (c *cipClient) forwardOpen() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	cip := buildForwardOpenCIP(c.slot)
	payload := buildCommandHeader(2)
	payload = append(payload, buildItem(enetItemNull, nil)...)
	payload = append(payload, buildItem(enetItemUnconnectedData, cip)...)
	request := buildEnetPacket(enetCommandSendRRData, c.session, 0, payload)
	_, response, err := c.exchangeEnet(request)
	if err != nil {
		return err
	}
	cipResponse, err := parseUnconnectedData(response)
	if err != nil {
		return err
	}
	otID, toID, err := parseForwardOpenResponse(cipResponse)
	if err != nil {
		return err
	}
	c.otConnectionID = otID
	c.toConnectionID = toID
	return nil
}

// Exchange sends one CIP request over the connected message and returns the CIP response.
func (c *cipClient) Exchange(ctx context.Context, cipRequest []byte) ([]byte, []byte, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil || c.closed {
		return nil, nil, nil, fmt.Errorf("AB5000 connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	sequence := c.sequence
	c.sequence++
	c.context++
	cipData := make([]byte, 2+len(cipRequest))
	binary.LittleEndian.PutUint16(cipData, sequence)
	copy(cipData[2:], cipRequest)

	payload := buildCommandHeader(2)
	connID := make([]byte, 4)
	binary.LittleEndian.PutUint32(connID, c.otConnectionID)
	payload = append(payload, buildItem(enetItemConnectionBased, connID)...)
	payload = append(payload, buildItem(enetItemConnectedData, cipData)...)
	request := buildEnetPacket(enetCommandSendUnitData, c.session, c.context, payload)

	wireHeader, wirePayload, err := c.exchangeEnet(request)
	if err != nil {
		return nil, request, append(wireHeader, wirePayload...), err
	}
	wire := append(append([]byte(nil), wireHeader...), wirePayload...)
	response, responseSeq, err := parseConnectedData(wirePayload)
	if err != nil {
		return nil, request, wire, err
	}
	if responseSeq != sequence {
		return nil, request, wire, fmt.Errorf("AB5000 sequence mismatch: got %d want %d", responseSeq, sequence)
	}
	return response, request, wire, nil
}

func (c *cipClient) exchangeEnet(request []byte) ([]byte, []byte, error) {
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, nil, err
	}
	if _, err := c.conn.Write(request); err != nil {
		return nil, nil, err
	}
	header := make([]byte, enetHeaderSize)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return header, nil, err
	}
	length := int(binary.LittleEndian.Uint16(header[2:4]))
	if length < 0 || length > enetMaxPayload {
		return header, nil, fmt.Errorf("invalid AB5000 payload length %d", length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return header, payload, err
	}
	if status := binary.LittleEndian.Uint32(header[8:12]); status != 0 {
		return header, payload, fmt.Errorf("AB5000 encapsulation status %d", status)
	}
	return header, payload, nil
}

func buildEnetPacket(command uint16, session uint32, context uint64, payload []byte) []byte {
	packet := make([]byte, enetHeaderSize+len(payload))
	binary.LittleEndian.PutUint16(packet[0:2], command)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(len(payload)))
	binary.LittleEndian.PutUint32(packet[4:8], session)
	binary.LittleEndian.PutUint64(packet[12:20], context)
	copy(packet[enetHeaderSize:], payload)
	return packet
}

func buildCommandHeader(items uint16) []byte {
	header := make([]byte, enetCommandHeaderSize)
	binary.LittleEndian.PutUint16(header[4:6], enetConnectedTimeoutTicks)
	binary.LittleEndian.PutUint16(header[6:8], items)
	return header
}

func buildItem(itemType uint16, data []byte) []byte {
	item := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint16(item[0:2], itemType)
	binary.LittleEndian.PutUint16(item[2:4], uint16(len(data)))
	copy(item[4:], data)
	return item
}

func buildForwardOpenCIP(slot int) []byte {
	cip := []byte{enetForwardOpenService, 0x02, 0x20, 0x06, 0x24, 0x01}
	cip = append(cip, buildForwardOpenBody(slot)...)
	return cip
}

func buildForwardOpenBody(slot int) []byte {
	body := make([]byte, enetForwardOpenBodySize)
	body[0] = 0x05
	body[1] = 0xF7
	binary.LittleEndian.PutUint32(body[2:], 2)
	binary.LittleEndian.PutUint32(body[6:], 0x80FE0001)
	binary.LittleEndian.PutUint16(body[10:], 2)
	binary.LittleEndian.PutUint16(body[12:], 0x051B)
	binary.LittleEndian.PutUint32(body[14:], uint32(time.Now().UnixMilli()))
	body[18] = 2
	binary.LittleEndian.PutUint32(body[22:], 0x001E8480)
	binary.LittleEndian.PutUint16(body[26:], 0x43F4)
	binary.LittleEndian.PutUint32(body[28:], 0x001E8480)
	binary.LittleEndian.PutUint16(body[32:], 0x43F4)
	body[34] = 0xA3
	body[35] = enetForwardOpenPathSize
	body[36] = 0x01
	body[37] = byte(slot)
	body[38] = 0x20
	body[39] = 0x02
	body[40] = 0x24
	body[41] = 0x01
	return body
}

func parseForwardOpenResponse(response []byte) (uint32, uint32, error) {
	if len(response) < 4 {
		return 0, 0, fmt.Errorf("AB5000 forward open response is too short")
	}
	if response[0] != enetForwardOpenResponse {
		return 0, 0, fmt.Errorf("AB5000 forward open service is %02X", response[0])
	}
	if response[2] != 0 {
		return 0, 0, fmt.Errorf("AB5000 forward open status %d", response[2])
	}
	offset := 4 + int(response[3])*2
	if offset+8 > len(response) {
		return 0, 0, fmt.Errorf("AB5000 forward open response body is too short")
	}
	ot := binary.LittleEndian.Uint32(response[offset:])
	to := binary.LittleEndian.Uint32(response[offset+4:])
	if ot == 0 {
		return 0, 0, fmt.Errorf("AB5000 forward open returned zero connection id")
	}
	return ot, to, nil
}

func parseUnconnectedData(payload []byte) ([]byte, error) {
	return parseItemData(payload, enetItemUnconnectedData)
}

func parseConnectedData(payload []byte) ([]byte, uint16, error) {
	data, err := parseItemData(payload, enetItemConnectedData)
	if err != nil {
		return nil, 0, err
	}
	if len(data) < 2 {
		return nil, 0, fmt.Errorf("AB5000 connected data is too short")
	}
	return data[2:], binary.LittleEndian.Uint16(data), nil
}

func parseItemData(payload []byte, want uint16) ([]byte, error) {
	if len(payload) < enetCommandHeaderSize {
		return nil, fmt.Errorf("AB5000 command header is too short")
	}
	count := int(binary.LittleEndian.Uint16(payload[6:8]))
	offset := enetCommandHeaderSize
	for index := 0; index < count; index++ {
		if offset+4 > len(payload) {
			return nil, fmt.Errorf("AB5000 item header is too short")
		}
		itemType := binary.LittleEndian.Uint16(payload[offset:])
		length := int(binary.LittleEndian.Uint16(payload[offset+2:]))
		offset += 4
		if offset+length > len(payload) {
			return nil, fmt.Errorf("AB5000 item data is too short")
		}
		if itemType == want {
			return payload[offset : offset+length], nil
		}
		offset += length
	}
	return nil, fmt.Errorf("AB5000 response has no item %04X", want)
}
