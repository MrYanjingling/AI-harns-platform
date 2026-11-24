package bacnet

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"
)

type bacClient struct {
	conn    *net.UDPConn
	remote  *net.UDPAddr
	timeout time.Duration
	invoke  uint8
}

func openClient(ctx context.Context, host string, port, srcPort int, bbmd bool, timeout time.Duration) (*bacClient, error) {
	_ = ctx
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid BACnet port")
	}
	if srcPort < 0 || srcPort > 65535 {
		return nil, fmt.Errorf("invalid BACnet source port")
	}
	remote, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	local := &net.UDPAddr{IP: net.IPv4zero, Port: srcPort}
	conn, err := net.ListenUDP("udp4", local)
	if err != nil {
		return nil, err
	}
	item := &bacClient{conn: conn, remote: remote, timeout: timeout}
	if bbmd {
		if _, err := conn.WriteToUDP(registerForeignDevice(), remote); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return item, nil
}

func registerForeignDevice() []byte {
	return []byte{0x81, 0x05, 0x00, 0x06, 0x02, 0x58}
}

func (c *bacClient) Available() bool { return c != nil && c.conn != nil }

func (c *bacClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *bacClient) Read(point Point) (any, error) {
	invoke := c.nextInvoke()
	apdu, err := c.request(readRequest(point.ObjectType, point.Instance, point.Property, invoke))
	if err != nil {
		return nil, err
	}
	return parseReadResponse(apdu)
}

func (c *bacClient) Write(point Point, value any, dataType string) error {
	invoke := c.nextInvoke()
	apdu, err := writeRequest(point.ObjectType, point.Instance, point.Property, invoke, value, dataType)
	if err != nil {
		return err
	}
	response, err := c.request(apdu)
	if err != nil {
		return err
	}
	return parseWriteResponse(response)
}

func (c *bacClient) nextInvoke() uint8 {
	c.invoke++
	return c.invoke
}

func (c *bacClient) request(apdu []byte) ([]byte, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("BACnet client is not connected")
	}
	packet := wrapBVLC(append([]byte{0x01, 0x04}, apdu...))
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.WriteToUDP(packet, c.remote); err != nil {
		return nil, err
	}
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	buffer := make([]byte, 1500)
	for {
		n, _, err := c.conn.ReadFromUDP(buffer)
		if err != nil {
			return nil, err
		}
		response, err := extractAPDU(buffer[:n])
		if err != nil || len(response) < 2 {
			continue
		}
		if response[1] == apdu[2] {
			return response, nil
		}
	}
}

func wrapBVLC(npdu []byte) []byte {
	packet := []byte{0x81, 0x0A, 0, 0}
	binary.BigEndian.PutUint16(packet[2:], uint16(len(npdu)+4))
	return append(packet, npdu...)
}

func extractAPDU(frame []byte) ([]byte, error) {
	if len(frame) < 6 || frame[0] != 0x81 {
		return nil, fmt.Errorf("invalid BACnet BVLC")
	}
	length := int(binary.BigEndian.Uint16(frame[2:4]))
	if length < 6 || length > len(frame) {
		return nil, fmt.Errorf("invalid BACnet BVLC length")
	}
	frame = frame[:length]
	index := 4
	if frame[1] == 0x04 {
		if len(frame) < 10 {
			return nil, fmt.Errorf("invalid BACnet forwarded NPDU")
		}
		index = 10
	}
	if frame[1] != 0x0A && frame[1] != 0x04 {
		return nil, fmt.Errorf("unsupported BACnet BVLC function")
	}
	if len(frame) < index+2 || frame[index] != 0x01 {
		return nil, fmt.Errorf("invalid BACnet NPDU")
	}
	control := frame[index+1]
	index += 2
	if control&0x80 != 0 {
		return nil, fmt.Errorf("BACnet network layer message is unsupported")
	}
	if control&0x20 != 0 {
		if len(frame) < index+3 {
			return nil, fmt.Errorf("truncated BACnet destination")
		}
		index += 3 + int(frame[index+2]) + 1
	}
	if control&0x08 != 0 {
		if len(frame) < index+3 {
			return nil, fmt.Errorf("truncated BACnet source")
		}
		index += 3 + int(frame[index+2])
	}
	if len(frame) <= index {
		return nil, fmt.Errorf("missing BACnet APDU")
	}
	return frame[index:], nil
}

func parseReadResponse(apdu []byte) (any, error) {
	if len(apdu) < 3 {
		return nil, fmt.Errorf("truncated BACnet response")
	}
	switch apdu[0] >> 4 {
	case 0x5:
		return nil, apduError(apdu)
	case 0x6:
		return nil, fmt.Errorf("BACnet reject reason %d", apdu[2])
	case 0x7:
		return nil, fmt.Errorf("BACnet abort reason %d", apdu[2])
	}
	if apdu[0]>>4 != 0x3 || apdu[2] != 12 {
		return nil, fmt.Errorf("unexpected BACnet read response")
	}
	index := 3
	_, next, err := skipContextTag(apdu, index)
	if err != nil {
		return nil, err
	}
	index = next
	_, next, err = skipContextTag(apdu, index)
	if err != nil {
		return nil, err
	}
	index = next
	number, next, err := skipContextTag(apdu, index)
	if err != nil {
		return nil, err
	}
	index = next
	if number == 2 {
		number, index, err = skipContextTag(apdu, index)
		if err != nil {
			return nil, err
		}
	}
	if number != 3 {
		return nil, fmt.Errorf("missing BACnet opening tag")
	}
	value, _, err := decodeApplicationValue(apdu[index:])
	return value, err
}

func parseWriteResponse(apdu []byte) error {
	if len(apdu) < 3 {
		return fmt.Errorf("truncated BACnet response")
	}
	switch apdu[0] >> 4 {
	case 0x5:
		return apduError(apdu)
	case 0x2:
		if apdu[2] == 15 {
			return nil
		}
		return fmt.Errorf("unexpected BACnet write service %d", apdu[2])
	case 0x6:
		return fmt.Errorf("BACnet reject reason %d", apdu[2])
	case 0x7:
		return fmt.Errorf("BACnet abort reason %d", apdu[2])
	default:
		return fmt.Errorf("unexpected BACnet write response")
	}
}

func apduError(apdu []byte) error {
	if len(apdu) < 5 {
		return fmt.Errorf("truncated BACnet error")
	}
	return fmt.Errorf("BACnet error class %d code %d", apdu[3], apdu[4])
}

func skipContextTag(data []byte, index int) (byte, int, error) {
	if index >= len(data) {
		return 0, index, fmt.Errorf("truncated BACnet context tag")
	}
	header := data[index]
	number := header >> 4
	if header&0x08 == 0 {
		return 0, index, fmt.Errorf("expected BACnet context tag")
	}
	length := int(header & 0x07)
	index++
	if length == 6 || length == 7 {
		return number, index, nil
	}
	if length == 5 {
		if index >= len(data) {
			return 0, index, fmt.Errorf("truncated BACnet context length")
		}
		length = int(data[index])
		index++
		if length == 254 {
			if index+1 >= len(data) {
				return 0, index, fmt.Errorf("truncated BACnet extended context length")
			}
			length = int(data[index])<<8 | int(data[index+1])
			index += 2
		}
	}
	if index+length > len(data) {
		return 0, index, fmt.Errorf("truncated BACnet context value")
	}
	return number, index + length, nil
}
