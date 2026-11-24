package df1

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"harns-platform/pkg/gateway/collection/meterlink"
)

type df1Client struct {
	link    *meterlink.Client
	timeout time.Duration
	tns     byte
}

func openDF1Client(ctx context.Context, location string, baud, data int, parity, stop string, timeout time.Duration) (*df1Client, error) {
	link, err := meterlink.Open(ctx, 0, location, 0, timeout, meterlink.SerialMode(baud, data, parity, stop))
	if err != nil {
		return nil, err
	}
	return &df1Client{link: meterlink.NewClient(link, timeout), timeout: timeout}, nil
}

func (c *df1Client) Available() bool { return c != nil && c.link != nil && c.link.Available() }
func (c *df1Client) Close() error {
	if c == nil || c.link == nil {
		return nil
	}
	return c.link.Close()
}

func (c *df1Client) Read(ctx context.Context, address Address, dataType string) (any, error) {
	_ = ctx
	request := c.buildReadRequest(address)
	if err := c.write(request); err != nil {
		return nil, err
	}
	if err := c.readAck(); err != nil {
		return nil, err
	}
	response, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	_ = c.writeAck()
	data, err := parseResponse(response, address)
	if err != nil {
		return nil, err
	}
	return decode(data, address, dataType)
}

func (c *df1Client) buildReadRequest(address Address) []byte {
	c.tns++
	payload := []byte{1, 0, 0x0F, 0x00, c.tns, 0, 0xA2, address.ByteSize, address.FileNum, address.FileType, address.Element, 0}
	return buildFrame(payload)
}

func buildFrame(payload []byte) []byte {
	frame := make([]byte, 0, len(payload)+5)
	frame = append(frame, 0x10, 0x02)
	frame = append(frame, payload...)
	frame = append(frame, 0x10, 0x03)
	crc := df1CRC(frame)
	return append(frame, byte(crc), byte(crc>>8))
}

func (c *df1Client) write(data []byte) error {
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.link.Write(data)
	return err
}

func (c *df1Client) readAck() error {
	data := make([]byte, 2)
	if _, err := io.ReadFull(c.link, data); err != nil {
		return err
	}
	if data[0] != 0x10 || data[1] != 0x06 {
		return fmt.Errorf("invalid DF1 ACK")
	}
	return nil
}

func (c *df1Client) writeAck() error {
	return c.write([]byte{0x10, 0x06})
}

func (c *df1Client) readFrame() ([]byte, error) {
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	frame := make([]byte, 0, 64)
	for len(frame) < 512 {
		var one [1]byte
		if _, err := io.ReadFull(c.link, one[:]); err != nil {
			return nil, err
		}
		frame = append(frame, one[0])
		if len(frame) >= 3 && frame[len(frame)-3] == 0x10 && frame[len(frame)-2] == 0x03 {
			var crc [2]byte
			if _, err := io.ReadFull(c.link, crc[:]); err != nil {
				return nil, err
			}
			frame = append(frame, crc[:]...)
			return frame, nil
		}
	}
	return nil, fmt.Errorf("DF1 frame is too long")
}

func parseResponse(frame []byte, address Address) ([]byte, error) {
	if len(frame) < 12 || frame[0] != 0x10 || frame[1] != 0x02 || frame[len(frame)-4] != 0x10 || frame[len(frame)-3] != 0x03 {
		return nil, fmt.Errorf("invalid DF1 response")
	}
	if df1CRC(frame[:len(frame)-2]) != uint16(frame[len(frame)-2])|uint16(frame[len(frame)-1])<<8 {
		return nil, fmt.Errorf("DF1 response CRC mismatch")
	}
	payload := frame[2 : len(frame)-4]
	if len(payload) < 6 {
		return nil, fmt.Errorf("short DF1 response")
	}
	if payload[2] != 0x4F {
		return nil, fmt.Errorf("unexpected DF1 response command 0x%02X", payload[2])
	}
	if payload[3] != 0 {
		return nil, fmt.Errorf("DF1 response status 0x%02X", payload[3])
	}
	data := payload[6:]
	if address.FileType == fileTimer || address.FileType == fileCounter || address.FileType == fileControl {
		switch address.Sub {
		case "ACC":
			if len(data) >= 4 {
				data = data[2:]
			}
		}
	}
	if address.FileType == fileString {
		if len(data) < 2 {
			return nil, fmt.Errorf("short DF1 string response")
		}
		data = data[2:]
	}
	return data, nil
}

func df1CRC(data []byte) uint16 {
	var crc uint16
	for index, value := range data {
		if index == 0 || index == 1 || index == len(data)-2 {
			continue
		}
		crc ^= uint16(value)
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func decode(data []byte, address Address, dataType string) (any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty DF1 value")
	}
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bit":
		return int((data[0] >> address.Bit) & 1), nil
	case "bool":
		return (data[0]>>address.Bit)&1 != 0, nil
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
		return nil, fmt.Errorf("unsupported DF1 data type %q", dataType)
	}
}
