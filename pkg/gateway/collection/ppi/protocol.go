package ppi

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection/meterlink"
	"harns-platform/pkg/gateway/collection/mpi"
)

type ppiClient struct {
	link    *meterlink.Client
	timeout time.Duration
	station byte
}

func openClient(ctx context.Context, link int, location string, port int, station byte, timeout time.Duration, mode *serial.Mode) (*ppiClient, error) {
	item, err := meterlink.Open(ctx, link, location, port, timeout, mode)
	if err != nil {
		return nil, err
	}
	return &ppiClient{link: meterlink.NewClient(item, timeout), timeout: timeout, station: station}, nil
}

func (c *ppiClient) Available() bool { return c != nil && c.link != nil && c.link.Available() }
func (c *ppiClient) Close() error {
	if c == nil || c.link == nil {
		return nil
	}
	return c.link.Close()
}

func (c *ppiClient) Read(address mpi.Address) ([]byte, error) {
	request, err := buildReadRequest(c.station, address)
	if err != nil {
		return nil, err
	}
	if err := c.write(request); err != nil {
		return nil, err
	}
	value, err := c.readByte()
	if err != nil {
		return nil, err
	}
	if value != 0xE5 {
		return nil, fmt.Errorf("MPI/PPI unexpected acknowledgement 0x%02X", value)
	}
	if err := c.write(buildAck(c.station)); err != nil {
		return nil, err
	}
	response, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	return parseReadResponse(response, c.station)
}

func buildReadRequest(station byte, address mpi.Address) ([]byte, error) {
	pdu := []byte{0x32, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0x00, 0x00, 0x04, 0x01, 0x12, 0x0a, 0x10}
	typ := address.Type
	start := uint32(address.Start) * 8
	if typ == 1 {
		start += uint32(address.Bit)
	}
	switch address.Area {
	case 28, 29, 30, 31, 32:
		typ = address.Area
		start = uint32(address.Start)
	case 6, 7:
		typ = 4
	}
	pdu = append(pdu, typ, byte(address.Length>>8), byte(address.Length), byte(address.DB>>8), byte(address.DB), address.Area,
		byte(start>>16), byte(start>>8), byte(start))
	length := len(pdu) + 3
	frame := []byte{0x68, byte(length), byte(length), 0x68, station, 0x00, 0x6c}
	frame = append(frame, pdu...)
	var sum byte
	for _, value := range frame[4:] {
		sum += value
	}
	return append(frame, sum, 0x16), nil
}

func buildAck(station byte) []byte {
	body := []byte{station, 0x00, 0x5c}
	var sum byte
	for _, value := range body {
		sum += value
	}
	return []byte{0x10, body[0], body[1], body[2], sum, 0x16}
}

func (c *ppiClient) readFrame() ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(c.link, header); err != nil {
		return nil, err
	}
	if header[0] != 0x68 || header[3] != 0x68 || header[1] != header[2] {
		return nil, fmt.Errorf("invalid PPI frame header")
	}
	total := int(header[1]) + 6
	if total < 6 || total > 4096 {
		return nil, fmt.Errorf("invalid PPI frame length")
	}
	frame := make([]byte, total)
	copy(frame, header)
	if _, err := io.ReadFull(c.link, frame[4:]); err != nil {
		return nil, err
	}
	if frame[total-1] != 0x16 {
		return nil, fmt.Errorf("invalid PPI frame end")
	}
	var sum byte
	for _, value := range frame[4 : total-2] {
		sum += value
	}
	if sum != frame[total-2] {
		return nil, fmt.Errorf("PPI checksum mismatch")
	}
	return frame, nil
}

func parseReadResponse(frame []byte, station byte) ([]byte, error) {
	if len(frame) < 28 || frame[4] != 0 || frame[5] != station {
		return nil, fmt.Errorf("invalid PPI response")
	}
	if frame[21] != 0xff {
		return nil, fmt.Errorf("PPI item error 0x%02X", frame[21])
	}
	transport := frame[22]
	bitLength := int(frame[23])<<8 | int(frame[24])
	dataLength := (bitLength + 7) / 8
	if transport == 3 {
		dataLength = 1
	}
	if dataLength <= 0 || 25+dataLength > len(frame)-2 {
		return nil, fmt.Errorf("invalid PPI response data length")
	}
	data := make([]byte, dataLength)
	copy(data, frame[25:25+dataLength])
	return data, nil
}

func (c *ppiClient) write(data []byte) error {
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

func (c *ppiClient) readByte() (byte, error) {
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}
	var data [1]byte
	if _, err := io.ReadFull(c.link, data[:]); err != nil {
		return 0, err
	}
	return data[0], nil
}
