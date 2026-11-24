package snmp

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	tagInteger        = 0x02
	tagOctetString    = 0x04
	tagNull           = 0x05
	tagOID            = 0x06
	tagSequence       = 0x30
	tagGetResponse    = 0xA2
	tagNoSuchObject   = 0x80
	tagNoSuchInstance = 0x81
	tagEndOfMibView   = 0x82
	tagIPAddress      = 0x40
	tagCounter32      = 0x41
	tagGauge32        = 0x42
	tagTimeTicks      = 0x43
	tagOpaque         = 0x44
	tagCounter64      = 0x46
)

type client struct {
	conn      *net.UDPConn
	remote    *net.UDPAddr
	timeout   time.Duration
	mu        sync.Mutex
	requestID uint8
	nextID    uint8
}

func openClient(ctx context.Context, host string, port, localPort int, timeout time.Duration) (*client, error) {
	_ = ctx
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid SNMP port")
	}
	if localPort < 0 || localPort > 65535 {
		return nil, fmt.Errorf("invalid SNMP local port")
	}
	remote, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: localPort})
	if err != nil {
		return nil, err
	}
	item := &client{conn: conn, remote: remote, timeout: timeout, nextID: 1}
	return item, nil
}

func (c *client) Available() bool { return c != nil && c.conn != nil }

func (c *client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *client) Read(address Address) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, fmt.Errorf("SNMP connection is closed")
	}
	id := c.nextID
	c.nextID++
	if c.nextID == 0 {
		c.nextID = 1
	}
	request := buildGetRequest(address, id)
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.WriteToUDP(request, c.remote); err != nil {
		return nil, err
	}
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	buffer := make([]byte, 65535)
	for {
		n, _, err := c.conn.ReadFromUDP(buffer)
		if err != nil {
			return nil, err
		}
		value, idValue, err := parseResponse(buffer[:n])
		if err != nil {
			continue
		}
		if idValue != id {
			continue
		}
		return value, nil
	}
}

func buildGetRequest(address Address, requestID uint8) []byte {
	varbind := tlv(tagSequence, append(tlv(tagOID, address.Encoded), tlv(tagNull, nil)...))
	varbinds := tlv(tagSequence, varbind)
	pdu := tlv(0xA0, concat(
		tlv(tagInteger, []byte{requestID}),
		tlv(tagInteger, []byte{0}),
		tlv(tagInteger, []byte{0}),
		varbinds,
	))
	message := tlv(tagSequence, concat(
		tlv(tagInteger, []byte{1}),
		tlv(tagOctetString, []byte(address.Community)),
		pdu,
	))
	return message
}

func parseResponse(data []byte) (any, uint8, error) {
	message, rest, err := readTLV(data)
	if err != nil || len(rest) != 0 || message.Tag != tagSequence {
		return nil, 0, fmt.Errorf("invalid SNMP message")
	}
	version, rest, err := readTLV(message.Data)
	if err != nil || version.Tag != tagInteger || len(version.Data) != 1 || version.Data[0] != 1 {
		return nil, 0, fmt.Errorf("invalid SNMP version")
	}
	community, rest, err := readTLV(rest)
	if err != nil || community.Tag != tagOctetString {
		return nil, 0, fmt.Errorf("invalid SNMP community")
	}
	pdu, _, err := readTLV(rest)
	if err != nil || pdu.Tag != tagGetResponse {
		return nil, 0, fmt.Errorf("invalid SNMP PDU")
	}
	requestID, rest, err := readTLV(pdu.Data)
	if err != nil || requestID.Tag != tagInteger || len(requestID.Data) != 1 {
		return nil, 0, fmt.Errorf("invalid SNMP request id")
	}
	errorStatus, rest, err := readTLV(rest)
	if err != nil || errorStatus.Tag != tagInteger || len(errorStatus.Data) != 1 {
		return nil, 0, fmt.Errorf("invalid SNMP error status")
	}
	if errorStatus.Data[0] != 0 {
		return nil, requestID.Data[0], fmt.Errorf("SNMP error status %d", errorStatus.Data[0])
	}
	errorIndex, rest, err := readTLV(rest)
	if err != nil || errorIndex.Tag != tagInteger {
		return nil, 0, fmt.Errorf("invalid SNMP error index")
	}
	_ = errorIndex
	varbinds, _, err := readTLV(rest)
	if err != nil || varbinds.Tag != tagSequence {
		return nil, 0, fmt.Errorf("invalid SNMP varbinds")
	}
	varbind, _, err := readTLV(varbinds.Data)
	if err != nil || varbind.Tag != tagSequence {
		return nil, 0, fmt.Errorf("invalid SNMP varbind")
	}
	_, rest, err = readTLV(varbind.Data)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid SNMP OID")
	}
	value, _, err := readTLV(rest)
	if err != nil {
		return nil, 0, err
	}
	result, err := decodeValue(value)
	return result, requestID.Data[0], err
}

type element struct {
	Tag  byte
	Data []byte
}

func readTLV(data []byte) (element, []byte, error) {
	if len(data) < 2 {
		return element{}, nil, fmt.Errorf("truncated BER")
	}
	tag := data[0]
	length, index, err := readLength(data[1:])
	if err != nil {
		return element{}, nil, err
	}
	index++
	if index+length > len(data) {
		return element{}, nil, fmt.Errorf("truncated BER value")
	}
	return element{Tag: tag, Data: data[index : index+length]}, data[index+length:], nil
}

func readLength(data []byte) (int, int, error) {
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("missing BER length")
	}
	first := data[0]
	if first&0x80 == 0 {
		return int(first), 1, nil
	}
	count := int(first & 0x7f)
	if count == 0 || count > 4 || len(data) < count+1 {
		return 0, 0, fmt.Errorf("invalid BER length")
	}
	length := 0
	for i := 1; i <= count; i++ {
		length = length<<8 | int(data[i])
	}
	return length, count + 1, nil
}

func tlv(tag byte, data []byte) []byte {
	header := []byte{tag}
	length := len(data)
	switch {
	case length < 0x80:
		header = append(header, byte(length))
	case length <= 0xFF:
		header = append(header, 0x81, byte(length))
	case length <= 0xFFFF:
		header = append(header, 0x82, byte(length>>8), byte(length))
	default:
		header = append(header, 0x83, byte(length>>16), byte(length>>8), byte(length))
	}
	return append(header, data...)
}

func concat(parts ...[]byte) []byte {
	var result []byte
	for _, part := range parts {
		result = append(result, part...)
	}
	return result
}

func decodeValue(value element) (any, error) {
	switch value.Tag {
	case 0x00, tagNull:
		return nil, nil
	case 0x01:
		return len(value.Data) > 0 && value.Data[0] != 0, nil
	case tagInteger:
		return signedInteger(value.Data), nil
	case tagOctetString:
		return strings.TrimRight(string(value.Data), "\x00"), nil
	case tagOID:
		return decodeOID(value.Data)
	case tagIPAddress:
		if len(value.Data) != 4 {
			return nil, fmt.Errorf("invalid SNMP IP address")
		}
		return fmt.Sprintf("%d.%d.%d.%d", value.Data[0], value.Data[1], value.Data[2], value.Data[3]), nil
	case tagCounter32, tagGauge32, tagTimeTicks, tagOpaque, tagCounter64:
		return unsignedInteger(value.Data), nil
	case 0x09:
		if len(value.Data) != 4 {
			return nil, fmt.Errorf("invalid SNMP REAL")
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(value.Data))), nil
	case tagNoSuchInstance:
		return nil, fmt.Errorf("SNMP noSuchInstance")
	case tagNoSuchObject:
		return nil, fmt.Errorf("SNMP noSuchObject")
	case tagEndOfMibView:
		return nil, fmt.Errorf("SNMP endOfMibView")
	default:
		return nil, fmt.Errorf("unsupported SNMP value tag 0x%X", value.Tag)
	}
}

func signedInteger(data []byte) int64 {
	if len(data) == 0 {
		return 0
	}
	value := int64(int8(data[0]))
	for _, item := range data[1:] {
		value = value<<8 | int64(item)
	}
	return value
}

func unsignedInteger(data []byte) uint64 {
	var value uint64
	for _, item := range data {
		value = value<<8 | uint64(item)
	}
	return value
}
