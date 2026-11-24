package knx

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

func (c *knxClient) Write(group uint16, address Address, dataType string, value any) error {
	payload, err := encodeWriteValue(address, dataType, value)
	if err != nil {
		return err
	}
	destination := c.dataEndpoint
	if destination == nil {
		destination = c.gateway
	}
	if err := c.writePacket(buildTunnelWrite(c.channel, c.sendSeq, group, payload), destination); err != nil {
		return err
	}
	deadline := time.Now().Add(c.timeout)
	for {
		packet, _, err := c.readPacket(deadline)
		if err != nil {
			return err
		}
		service, total, err := parseHeader(packet)
		if err != nil || total > len(packet) {
			continue
		}
		switch service {
		case serviceTunnelingAck:
			if len(packet) >= 10 && packet[7] == c.channel && packet[8] == c.sendSeq {
				c.sendSeq++
				return nil
			}
		case serviceTunnelingRequest:
			if len(packet) < 10 || packet[7] != c.channel {
				continue
			}
			_ = c.writePacket(buildTunnelAck(c.channel, packet[8]), destination)
			if packet[8] == c.recvSeq {
				c.recvSeq++
			}
			return nil
		}
	}
}

func buildTunnelWrite(channel, sequence byte, group uint16, data []byte) []byte {
	cemi := []byte{0x11, 0x00, 0xbc, 0xe0, 0x00, 0x00, byte(group >> 8), byte(group), byte(len(data)), 0x00}
	cemi = append(cemi, data...)
	request := makeHeader(serviceTunnelingRequest, 10+len(cemi))
	request = append(request, 0x04, channel, sequence, 0x00)
	return append(request, cemi...)
}

func encodeWriteValue(address Address, dataType string, value any) ([]byte, error) {
	switch dataType {
	case "bit", "BIT", "bool", "BOOL":
		bit, ok := value.(bool)
		if !ok {
			number, ok := toUint64(value)
			if !ok || number > 1 {
				return nil, fmt.Errorf("KNX bit value is invalid")
			}
			bit = number == 1
		}
		if bit {
			return []byte{0x81}, nil
		}
		return []byte{0x80}, nil
	case "uint8", "UINT8":
		number, ok := toUint64(value)
		if !ok {
			return nil, fmt.Errorf("KNX uint8 value is invalid")
		}
		if address.NBit > 0 {
			mask := uint64(1<<address.NBit) - 1
			if number > mask {
				return nil, fmt.Errorf("KNX %d-bit value is out of range", address.NBit)
			}
			return []byte{0x80 | byte(number)}, nil
		}
		if number > 255 {
			return nil, fmt.Errorf("KNX uint8 value is out of range")
		}
		return []byte{0x80, byte(number)}, nil
	case "int8", "INT8":
		number, ok := toInt64(value)
		if !ok || number < -128 || number > 127 {
			return nil, fmt.Errorf("KNX int8 value is out of range")
		}
		return []byte{0x80, byte(int8(number))}, nil
	case "uint16", "UINT16":
		number, ok := toUint64(value)
		if !ok || number > math.MaxUint16 {
			return nil, fmt.Errorf("KNX uint16 value is out of range")
		}
		data := []byte{0x80, 0, 0}
		binary.BigEndian.PutUint16(data[1:], uint16(number))
		return data, nil
	case "int16", "INT16":
		number, ok := toInt64(value)
		if !ok || number < math.MinInt16 || number > math.MaxInt16 {
			return nil, fmt.Errorf("KNX int16 value is out of range")
		}
		data := []byte{0x80, 0, 0}
		binary.BigEndian.PutUint16(data[1:], uint16(int16(number)))
		return data, nil
	case "uint32", "UINT32":
		number, ok := toUint64(value)
		if !ok || number > math.MaxUint32 {
			return nil, fmt.Errorf("KNX uint32 value is out of range")
		}
		data := []byte{0x80, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(data[1:], uint32(number))
		return data, nil
	case "int32", "INT32":
		number, ok := toInt64(value)
		if !ok || number < math.MinInt32 || number > math.MaxInt32 {
			return nil, fmt.Errorf("KNX int32 value is out of range")
		}
		data := []byte{0x80, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(data[1:], uint32(int32(number)))
		return data, nil
	case "float", "FLOAT", "float32", "FLOAT32":
		number, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("KNX float value is invalid")
		}
		data := []byte{0x80, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(data[1:], math.Float32bits(float32(number)))
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported KNX data type %q", dataType)
	}
}

func toUint64(value any) (uint64, bool) {
	switch number := value.(type) {
	case uint8:
		return uint64(number), true
	case uint16:
		return uint64(number), true
	case uint32:
		return uint64(number), true
	case uint64:
		return number, true
	case int:
		if number < 0 {
			return 0, false
		}
		return uint64(number), true
	case int8:
		if number < 0 {
			return 0, false
		}
		return uint64(number), true
	case int16:
		if number < 0 {
			return 0, false
		}
		return uint64(number), true
	case int32:
		if number < 0 {
			return 0, false
		}
		return uint64(number), true
	case int64:
		if number < 0 {
			return 0, false
		}
		return uint64(number), true
	case float32:
		if number < 0 || float32(uint64(number)) != number {
			return 0, false
		}
		return uint64(number), true
	case float64:
		if number < 0 || float64(uint64(number)) != number {
			return 0, false
		}
		return uint64(number), true
	default:
		return 0, false
	}
}

func toInt64(value any) (int64, bool) {
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
	case uint8:
		return int64(number), true
	case uint16:
		return int64(number), true
	case uint32:
		return int64(number), true
	case uint64:
		if number > math.MaxInt64 {
			return 0, false
		}
		return int64(number), true
	case float32:
		if float32(int64(number)) != number {
			return 0, false
		}
		return int64(number), true
	case float64:
		if float64(int64(number)) != number {
			return 0, false
		}
		return int64(number), true
	default:
		return 0, false
	}
}

func toFloat64(value any) (float64, bool) {
	switch number := value.(type) {
	case float32:
		return float64(number), true
	case float64:
		return number, true
	default:
		integer, ok := toInt64(value)
		return float64(integer), ok
	}
}
