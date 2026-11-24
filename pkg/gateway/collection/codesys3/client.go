package codesys3

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"time"

	"harns-platform/pkg/gateway/collection"
)

const passwordChallenge = 0x25659845
const passwordKey = "zeDR96EfU#27vuph7Thub?phaDr*rUbR"

type codesysClient struct {
	conn      net.Conn
	host      string
	port      int
	timeout   time.Duration
	src       []byte
	dst       []byte
	channelID int
	sessionID uint32
	appID     uint32
	ctxID     uint32
	block     int
	ack       int
	msgID     byte
	closed    bool
}

func openCodesysClient(ctx context.Context, host string, port int, username, password string, timeout time.Duration) (*codesysClient, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	if err != nil {
		return nil, err
	}
	client := &codesysClient{
		conn: conn, host: host, port: port, timeout: timeout,
		src: nodeAddress("0.0.0.0", port, 0x2ddf),
		dst: nodeAddress(host, port, 0),
	}
	if err := client.resolve(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.openChannel(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.login(username, password); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.getApplicationID(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *codesysClient) Available() bool { return c != nil && c.conn != nil && !c.closed }
func (c *codesysClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	c.closed = true
	return c.conn.Close()
}

func (c *codesysClient) resolve() error {
	c.msgID++
	request := buildResolveRequest(c.msgID, c.src, c.dst)
	if err := c.sendFrame(request); err != nil {
		return err
	}
	_, payload, src, err := c.recvRouter()
	if err != nil {
		return err
	}
	if len(src) > 0 {
		c.dst = append([]byte{}, src...)
	}
	_ = payload
	return nil
}

func (c *codesysClient) openChannel() error {
	request := buildOpenChannelRequest(1, 0x100000)
	if err := c.sendFrame(buildRouterDatagram(request, c.src, c.dst, serviceChannel, c.nextMsgID())); err != nil {
		return err
	}
	_, payload, _, err := c.recvRouter()
	if err != nil {
		return err
	}
	channelID, _, result, err := parseOpenChannelResponse(payload)
	if err != nil {
		return err
	}
	if result != 0 {
		return fmt.Errorf("CODESYS open channel failed: 0x%04X", result)
	}
	c.channelID = channelID
	return nil
}

func (c *codesysClient) login(username, password string) error {
	children := []*tag{cstrTag(0x10, username)}
	if password != "" {
		children = append(children, &tag{ID: 0x11, Data: hashPassword(password)})
	}
	tags := []*tag{u32Tag(tagCrypto, 1), containerTag(0x81, children...)}
	_, response, err := c.request(sgDevice, 2, 0, tags)
	if err != nil {
		return err
	}
	session := findTag(response, tagSession)
	if session == nil {
		return fmt.Errorf("CODESYS login did not return a session")
	}
	c.sessionID, err = tagU32(session)
	return err
}

func (c *codesysClient) getApplicationID() error {
	_, response, err := c.request(sgIECVarAccess, 7, c.sessionID, nil)
	if err != nil {
		return err
	}
	app := findTag(response, 0x3A)
	if app == nil {
		return fmt.Errorf("CODESYS application id is missing")
	}
	c.appID, err = tagU32(app)
	return err
}

func (c *codesysClient) Read(ctx context.Context, address string, dataType string) (any, error) {
	_ = ctx
	ctxID, err := c.readStart(address)
	if err != nil {
		return nil, err
	}
	data, err := c.readContent(ctxID)
	if err != nil {
		_ = c.readEnd(ctxID)
		return nil, err
	}
	_ = c.readEnd(ctxID)
	return decodeValue(data, dataType)
}

func (c *codesysClient) readStart(address string) (uint32, error) {
	path := []byte(address)
	body := make([]byte, 2+len(path))
	binary.LittleEndian.PutUint16(body[:2], uint16(len(path)))
	copy(body[2:], path)
	for len(body)%4 != 0 {
		body = append(body, 0)
	}
	tags := []*tag{
		{ID: 0x19, Data: body},
		u32Tag(0x18, 0x03000000),
		u32Tag(0x3A, c.appID),
		u32Tag(0x13, 0x48000100),
	}
	_, response, err := c.request(sgIECVarAccess, 1, c.sessionID, tags)
	if err != nil {
		return 0, err
	}
	contextTag := findTag(response, 0x10)
	if contextTag == nil {
		return 0, fmt.Errorf("CODESYS read start has no context")
	}
	return tagU32(contextTag)
}

func (c *codesysClient) readContent(ctxID uint32) ([]byte, error) {
	_, response, err := c.request(sgIECVarAccess, 3, c.sessionID, []*tag{u32Tag(0x10, ctxID)})
	if err != nil {
		return nil, err
	}
	result := findTag(response, 0x1D)
	if result == nil || len(result.Data) < 5 {
		return nil, fmt.Errorf("CODESYS read content is missing")
	}
	return result.Data[4:], nil
}

func (c *codesysClient) readEnd(ctxID uint32) error {
	_, _, err := c.request(sgIECVarAccess, 2, c.sessionID, []*tag{u32Tag(0x10, ctxID)})
	return err
}

func (c *codesysClient) request(group, service int, sessionID uint32, tags []*tag) (int, []*tag, error) {
	message := buildService(group, service, int(sessionID), encodeTags(tags))
	if err := c.sendMessage(message); err != nil {
		return 0, nil, err
	}
	response, err := c.recvMessage()
	if err != nil {
		return 0, nil, err
	}
	responseGroup, _, _, tlv, err := parseService(response)
	if err != nil {
		return 0, nil, err
	}
	parsed, err := parseTags(tlv)
	return responseGroup, parsed, err
}

func (c *codesysClient) sendMessage(message []byte) error {
	c.block++
	firstBudget := 480 - 20
	if len(message) <= firstBudget {
		packet := buildDataFirst(c.channelID, c.block, c.ack, message)
		return c.sendFrame(buildRouterDatagram(packet, c.src, c.dst, serviceChannel, c.nextMsgID()))
	}
	first := message[:firstBudget]
	rest := message[firstBudget:]
	if err := c.sendFrame(buildRouterDatagram(buildDataFirstPartial(c.channelID, c.block, c.ack, message, first), c.src, c.dst, serviceChannel, c.nextMsgID())); err != nil {
		return err
	}
	for len(rest) > 0 {
		c.block++
		chunk := rest
		if len(chunk) > 468 {
			chunk = chunk[:468]
		}
		rest = rest[len(chunk):]
		if err := c.sendFrame(buildRouterDatagram(buildData(c.channelID, c.block, c.ack, chunk), c.src, c.dst, serviceChannel, c.nextMsgID())); err != nil {
			return err
		}
	}
	return nil
}

func buildDataFirstPartial(channelID, block, ack int, whole, first []byte) []byte {
	packet := make([]byte, 20, 20+len(first))
	packet[0] = l4Data
	packet[1] = l4FlagFirst
	binary.LittleEndian.PutUint16(packet[2:4], uint16(channelID))
	binary.LittleEndian.PutUint32(packet[4:8], uint32(block))
	binary.LittleEndian.PutUint32(packet[8:12], uint32(ack))
	binary.LittleEndian.PutUint32(packet[12:16], uint32(len(whole)))
	binary.LittleEndian.PutUint32(packet[16:20], crc32IEEE(whole))
	return append(packet, first...)
}

func (c *codesysClient) recvMessage() ([]byte, error) {
	buffer := make([]byte, 0)
	need := -1
	var want uint32
	for need < 0 || len(buffer) < need {
		_, payload, _, err := c.recvRouter()
		if err != nil {
			return nil, err
		}
		if len(payload) < 2 {
			continue
		}
		switch payload[0] {
		case l4Data:
			if payload[1]&l4FlagFirst != 0 {
				data, length, checksum, _, err := parseDataFirst(payload)
				if err != nil {
					return nil, err
				}
				need, want = length, checksum
				buffer = append(buffer, data...)
			} else {
				data, _, err := parseData(payload)
				if err != nil {
					return nil, err
				}
				buffer = append(buffer, data...)
			}
		case l4Ack, l4Keepalive:
			continue
		default:
			continue
		}
	}
	if len(buffer) < need {
		return nil, fmt.Errorf("short CODESYS message")
	}
	buffer = buffer[:need]
	if crc32IEEE(buffer) != want {
		return nil, fmt.Errorf("CODESYS message checksum mismatch")
	}
	return buffer, nil
}

func (c *codesysClient) sendFrame(data []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.conn.Write(blockFrame(data))
	return err
}

func (c *codesysClient) recvFrame() ([]byte, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	header := make([]byte, 8)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint32(header[:4]) != blockMagic {
		return nil, fmt.Errorf("invalid CODESYS block magic")
	}
	total := int(binary.LittleEndian.Uint32(header[4:8]))
	if total < 8 || total > 1<<20 {
		return nil, fmt.Errorf("invalid CODESYS block length")
	}
	body := make([]byte, total-8)
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (c *codesysClient) recvRouter() (byte, []byte, []byte, error) {
	body, err := c.recvFrame()
	if err != nil {
		return 0, nil, nil, collection.ConnectionLost(err)
	}
	service, payload, src, err := parseRouterDatagram(body)
	return service, payload, src, err
}

func (c *codesysClient) nextMsgID() byte {
	c.msgID++
	return c.msgID
}

func hashPassword(password string) []byte {
	fullLen := 32
	if len(password)+1 >= 32 {
		fullLen = len(password)
		if fullLen%4 != 0 {
			fullLen = (fullLen &^ 3) + 4
		}
	}
	out := make([]byte, fullLen)
	challenge := uint32(passwordChallenge)
	for index := 0; index < fullLen; index++ {
		var value byte
		if index < len(password) {
			value = password[index]
		}
		key := byte(passwordKey[index%len(passwordKey)])
		var salt byte
		if index%4 == 0 {
			salt = byte(challenge)
		}
		out[index] = value ^ (key + salt)
	}
	return out
}

func decodeValue(data []byte, dataType string) (any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty CODESYS value")
	}
	switch strings.ToLower(dataType) {
	case "bit":
		return int(data[0]), nil
	case "bool":
		return data[0] != 0, nil
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
	case "float":
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "string":
		return strings.TrimRight(string(data), "\x00"), nil
	default:
		return data, nil
	}
}
