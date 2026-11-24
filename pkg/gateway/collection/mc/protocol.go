package mc

import (
	"encoding/binary"
	"fmt"
)

const (
	cmdRead  = 0x0401
	cmdWrite = 0x1401

	subWord = 0x0000
	subBit  = 0x0001
)

func buildRequest(family Family, serial uint16, area byte, word bool, start uint32, count int, data []byte) []byte {
	sub := uint16(subWord)
	if !word {
		sub = subBit
	}
	command := uint16(cmdRead)
	if data != nil {
		command = cmdWrite
	}
	length := uint16(12 + len(data))
	frame := make([]byte, 0, 25+len(data))
	switch family {
	case Family1E:
		frame = append(frame, 0x50, 0x00, 0xFF, 0xFF, 0xFF, 0x03, 0xFF)
		frame = appendU16(frame, length)
		frame = appendU16(frame, 0x000C)
	case Family3E:
		frame = append(frame, 0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00)
		frame = appendU16(frame, length)
		frame = appendU16(frame, 0x0000)
	case Family4E:
		frame = append(frame, 0x54, 0x00)
		frame = appendU16(frame, serial)
		frame = append(frame, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00)
		frame = appendU16(frame, length)
		frame = appendU16(frame, 0x0000)
	default:
		return nil
	}
	frame = appendU16(frame, command)
	frame = appendU16(frame, sub)
	frame = append(frame, byte(start), byte(start>>8), byte(start>>16), area)
	frame = appendU16(frame, uint16(count))
	frame = append(frame, data...)
	return frame
}

func buildReadRequest(family Family, serial uint16, area byte, word bool, start uint32, count int) []byte {
	return buildRequest(family, serial, area, word, start, count, nil)
}

func buildWriteRequest(family Family, serial uint16, area byte, word bool, start uint32, count int, data []byte) []byte {
	return buildRequest(family, serial, area, word, start, count, data)
}

func parseResponse(family Family, serial uint16, response []byte) ([]byte, error) {
	switch family {
	case Family1E, Family3E:
		if len(response) < 11 {
			return nil, fmt.Errorf("MC response is too short")
		}
		if response[0] != 0xD0 || response[1] != 0x00 {
			return nil, fmt.Errorf("MC response subheader mismatch")
		}
		length := int(binary.LittleEndian.Uint16(response[7:9]))
		if len(response) < 9+length {
			return nil, fmt.Errorf("MC response length %d is shorter than header", len(response))
		}
		if length < 2 {
			return nil, fmt.Errorf("MC response length %d is invalid", length)
		}
		endCode := binary.LittleEndian.Uint16(response[9:11])
		if endCode != 0 {
			return nil, fmt.Errorf("MC response end code 0x%04X", endCode)
		}
		return response[11 : 9+length], nil
	case Family4E:
		if len(response) < 15 {
			return nil, fmt.Errorf("MC 4E response is too short")
		}
		if response[0] != 0xD4 || response[1] != 0x00 {
			return nil, fmt.Errorf("MC 4E response subheader mismatch")
		}
		if got := binary.LittleEndian.Uint16(response[2:4]); got != serial {
			return nil, fmt.Errorf("MC 4E response serial %d does not match %d", got, serial)
		}
		length := int(binary.LittleEndian.Uint16(response[11:13]))
		if len(response) < 13+length {
			return nil, fmt.Errorf("MC 4E response length %d is shorter than header", len(response))
		}
		if length < 2 {
			return nil, fmt.Errorf("MC 4E response length %d is invalid", length)
		}
		endCode := binary.LittleEndian.Uint16(response[13:15])
		if endCode != 0 {
			return nil, fmt.Errorf("MC 4E response end code 0x%04X", endCode)
		}
		return response[15 : 13+length], nil
	default:
		return nil, fmt.Errorf("MC family %s is not supported", family)
	}
}

func responseHeaderSize(family Family) int {
	if family == Family4E {
		return 13
	}
	return 9
}

func responseLengthOffset(family Family) int {
	if family == Family4E {
		return 11
	}
	return 7
}

func appendU16(dst []byte, value uint16) []byte {
	return append(dst, byte(value), byte(value>>8))
}
