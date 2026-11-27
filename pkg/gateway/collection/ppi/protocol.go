package ppi

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"

	"harns-platform/pkg/gateway/collection"
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

func (c *ppiClient) Write(address mpi.Address, payload []byte) error {
	request, err := buildWriteRequest(c.station, address, payload)
	if err != nil {
		return err
	}
	if err := c.write(request); err != nil {
		return err
	}
	value, err := c.readByte()
	if err != nil {
		return err
	}
	if value != 0xE5 {
		return fmt.Errorf("MPI/PPI unexpected acknowledgement 0x%02X", value)
	}
	if err := c.write(buildAck(c.station)); err != nil {
		return err
	}
	response, err := c.readFrame()
	if err != nil {
		return err
	}
	return parseWriteResponse(response, c.station)
}

func buildWriteRequest(station byte, address mpi.Address, payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("PPI write payload is empty")
	}
	var (
		typ       byte
		count     uint16
		start     uint32
		transport byte
		bitLen    uint16
		data      []byte
	)
	if address.Type == 1 {
		typ = 1
		count = 1
		start = uint32(address.Start)*8 + uint32(address.Bit)
		transport = 0x03
		bitLen = 1
		data = []byte{payload[0] & 1}
	} else {
		typ = 2
		count = uint16(len(payload))
		start = uint32(address.Start) * 8
		if address.Area >= 28 && address.Area <= 32 {
			typ = address.Area
			start = uint32(address.Start)
		}
		transport = 0x04
		bitLen = uint16(len(payload) * 8)
		data = payload
	}
	dataLen := 4 + len(data)
	pdu := make([]byte, 0, 24+dataLen)
	pdu = append(pdu,
		0x32, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x0e,
		byte(dataLen>>8), byte(dataLen),
		0x05, 0x01, 0x12, 0x0a, 0x10,
		typ, byte(count>>8), byte(count), byte(address.DB>>8), byte(address.DB), address.Area,
		byte(start>>16), byte(start>>8), byte(start),
		0x00, transport, byte(bitLen>>8), byte(bitLen),
	)
	pdu = append(pdu, data...)
	length := len(pdu) + 3
	if length > 255 {
		return nil, fmt.Errorf("PPI write frame length %d exceeds 255", length)
	}
	frame := make([]byte, 0, length+6)
	frame = append(frame, 0x68, byte(length), byte(length), 0x68, station, 0x00, 0x7c)
	frame = append(frame, pdu...)
	var sum byte
	for _, value := range frame[4:] {
		sum += value
	}
	return append(frame, sum, 0x16), nil
}

func parseWriteResponse(frame []byte, station byte) error {
	if len(frame) < 24 || frame[4] != 0 || frame[5] != station {
		return fmt.Errorf("invalid PPI write response")
	}
	if frame[19] != 0x05 {
		return fmt.Errorf("unexpected PPI write response function 0x%02X", frame[19])
	}
	if frame[21] != 0xff {
		return fmt.Errorf("PPI write item error 0x%02X", frame[21])
	}
	return nil
}

func encodeWritePayload(address mpi.Address, variable *collection.Variable, raw any) ([]byte, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	switch strings.ToLower(strings.TrimSpace(variable.DataType)) {
	case "bool", "bit":
		v, err := toBool(raw)
		if err != nil {
			return nil, err
		}
		if v {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	case "int16":
		v, err := toInt64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(int16(v)))
		return buf, nil
	case "uint16":
		v, err := toUint64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(v))
		return buf, nil
	case "int32":
		v, err := toInt64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(int32(v)))
		return buf, nil
	case "uint32":
		v, err := toUint64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(v))
		return buf, nil
	case "int64":
		v, err := toInt64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(v))
		return buf, nil
	case "uint64":
		v, err := toUint64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, v)
		return buf, nil
	case "float32":
		v, err := toFloat64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, math.Float32bits(float32(v)))
		return buf, nil
	case "float64":
		v, err := toFloat64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, math.Float64bits(v))
		return buf, nil
	case "string":
		s, ok := raw.(string)
		if !ok {
			s = fmt.Sprint(raw)
		}
		length := int(address.Length)
		if length <= 0 {
			length = len(s)
		}
		buf := make([]byte, length)
		copy(buf, s)
		return buf, nil
	case "byte_array":
		switch v := raw.(type) {
		case []byte:
			return append([]byte(nil), v...), nil
		case []int:
			buf := make([]byte, len(v))
			for i, b := range v {
				buf[i] = byte(b)
			}
			return buf, nil
		default:
			return nil, fmt.Errorf("unsupported PPI byte_array value %T", raw)
		}
	default:
		return nil, fmt.Errorf("unsupported PPI write data type %q", variable.DataType)
	}
}

func toBool(raw any) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case int8:
		return v != 0, nil
	case int16:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case uint:
		return v != 0, nil
	case uint8:
		return v != 0, nil
	case uint16:
		return v != 0, nil
	case uint32:
		return v != 0, nil
	case uint64:
		return v != 0, nil
	case float32:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(v))
	default:
		return false, fmt.Errorf("invalid PPI bool value %T", raw)
	}
}

func toInt64(raw any) (int64, error) {
	switch v := raw.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	default:
		return 0, fmt.Errorf("invalid PPI integer value %T", raw)
	}
}

func toUint64(raw any) (uint64, error) {
	switch v := raw.(type) {
	case uint:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case float32:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %v", v)
		}
		return uint64(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %v", v)
		}
		return uint64(v), nil
	case string:
		return strconv.ParseUint(strings.TrimSpace(v), 10, 64)
	default:
		return 0, fmt.Errorf("invalid PPI unsigned value %T", raw)
	}
}

func toFloat64(raw any) (float64, error) {
	switch v := raw.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return 0, fmt.Errorf("invalid PPI float value %T", raw)
	}
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
