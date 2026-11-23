package dnp3

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	linkStart1 = 0x05
	linkStart2 = 0x64
	linkHeader = 10
	linkMaxLen = 255

	linkFnResetLink       = 0
	linkFnResetLinkAck    = 1
	linkFnTestLink        = 2
	linkFnConfirmedData   = 3
	linkFnUnconfirmedData = 4
	linkFnRequestStatus   = 9
	linkFnLinkStatus      = 11
)

type linkFrame struct {
	Control byte
	Dest    uint16
	Src     uint16
	Payload []byte
}

func dnp3CRC(data []byte) uint16 {
	var crc uint16
	for _, value := range data {
		crc ^= uint16(value)
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA6BC
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

func buildLinkFrame(dest, src uint16, control byte, payload []byte) ([]byte, error) {
	if len(payload)+5 > linkMaxLen {
		return nil, fmt.Errorf("DNP3 link payload is too large")
	}
	frame := make([]byte, linkHeader, linkHeader+len(payload)+len(payload)/16*2+2)
	frame[0] = linkStart1
	frame[1] = linkStart2
	frame[2] = byte(len(payload) + 5)
	frame[3] = control
	binary.LittleEndian.PutUint16(frame[4:6], dest)
	binary.LittleEndian.PutUint16(frame[6:8], src)
	binary.LittleEndian.PutUint16(frame[8:10], dnp3CRC(frame[:8]))
	for offset := 0; offset < len(payload); offset += 16 {
		end := offset + 16
		if end > len(payload) {
			end = len(payload)
		}
		frame = append(frame, payload[offset:end]...)
		crc := dnp3CRC(payload[offset:end])
		frame = append(frame, byte(crc), byte(crc>>8))
	}
	return frame, nil
}

func readLinkFrame(conn net.Conn, timeout time.Duration) (linkFrame, []byte, error) {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return linkFrame{}, nil, err
	}
	header := make([]byte, linkHeader)
	wire := make([]byte, 0, 300)
	if _, err := io.ReadFull(conn, header); err != nil {
		return linkFrame{}, nil, err
	}
	wire = append(wire, header...)
	if header[0] != linkStart1 || header[1] != linkStart2 {
		return linkFrame{}, wire, fmt.Errorf("invalid DNP3 link start bytes")
	}
	length := int(header[2])
	payloadLen := length - 5
	if payloadLen < 0 {
		return linkFrame{}, wire, fmt.Errorf("invalid DNP3 link length %d", length)
	}
	crcBytes := (payloadLen + 15) / 16 * 2
	body := make([]byte, payloadLen+crcBytes)
	if _, err := io.ReadFull(conn, body); err != nil {
		return linkFrame{}, wire, err
	}
	wire = append(wire, body...)
	payload, err := stripLinkCRC(body, payloadLen)
	if err != nil {
		return linkFrame{}, wire, err
	}
	return linkFrame{
		Control: header[3],
		Dest:    binary.LittleEndian.Uint16(header[4:6]),
		Src:     binary.LittleEndian.Uint16(header[6:8]),
		Payload: payload,
	}, wire, nil
}

func stripLinkCRC(body []byte, payloadLen int) ([]byte, error) {
	out := make([]byte, 0, payloadLen)
	offset := 0
	for remain := payloadLen; remain > 0; {
		size := remain
		if size > 16 {
			size = 16
		}
		if offset+size+2 > len(body) {
			return nil, fmt.Errorf("DNP3 link payload is truncated")
		}
		expected := binary.LittleEndian.Uint16(body[offset+size : offset+size+2])
		actual := dnp3CRC(body[offset : offset+size])
		if expected != actual {
			return nil, fmt.Errorf("DNP3 link CRC mismatch")
		}
		out = append(out, body[offset:offset+size]...)
		offset += size + 2
		remain -= size
	}
	return out, nil
}

func linkFunction(control byte) byte { return control & 0x0F }
