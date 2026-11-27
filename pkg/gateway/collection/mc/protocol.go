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

	a1eCmdBitRead   byte = 0x00
	a1eCmdWordRead  byte = 0x01
	a1eCmdBitWrite  byte = 0x02
	a1eCmdWordWrite byte = 0x03
)

func a1eAreaCode(area byte) uint16 {
	switch area {
	case 0x9C: // X
		return 0x5820
	case 0x9D: // Y
		return 0x5920
	case 0x90: // M
		return 0x4D20
	case 0x92: // L
		return 0x4C20
	case 0x93: // F
		return 0x4620
	case 0x94: // V
		return 0x5620
	case 0xA0: // B
		return 0x4220
	case 0x98: // S
		return 0x5320
	case 0xA8: // D
		return 0x4420
	case 0xB4: // W
		return 0x5720
	case 0xAF, 0xB0: // R, ZR
		return 0x5220
	case 0xCC: // Z
		return 0x5A20
	case 0xA2: // DX
		return 0x4458
	case 0xA3: // DY
		return 0x4459
	case 0xC1: // TS
		return 0x5453
	case 0xC0: // TC
		return 0x5443
	case 0xC2: // TN
		return 0x544E
	case 0xC4: // CS
		return 0x4353
	case 0xC3: // CC
		return 0x4343
	case 0xC5: // CN
		return 0x434E
	case 0xA1: // SB
		return 0x5342
	case 0xB5: // SW
		return 0x5357
	case 0x91: // SM
		return 0x534D
	case 0xA9: // SD
		return 0x5344
	case 0xC7: // SS
		return 0x5353
	case 0xC6: // SC
		return 0x5343
	case 0xC8: // SN
		return 0x534E
	default:
		return 0x4420
	}
}

func a1eExpectedDataLength(request []byte) int {
	if len(request) < 12 {
		return 0
	}
	count := int(request[10])
	if count == 0 {
		count = 256
	}
	switch request[0] {
	case a1eCmdBitRead:
		return (count + 1) / 2
	case a1eCmdWordRead:
		return count * 2
	default:
		return 0
	}
}

func buildRequest(family Family, serial uint16, area byte, word bool, start uint32, count int, data []byte) []byte {
	if family == Family1E {
		cmd := a1eCmdBitRead
		if word {
			if data != nil {
				cmd = a1eCmdWordWrite
			} else {
				cmd = a1eCmdWordRead
			}
		} else if data != nil {
			cmd = a1eCmdBitWrite
		}
		frame := make([]byte, 0, 12+len(data))
		frame = append(frame, cmd, 0xFF, 0x0A, 0x00)
		frame = append(frame, byte(start), byte(start>>8), byte(start>>16), byte(start>>24))
		frame = appendU16(frame, a1eAreaCode(area))
		frame = append(frame, byte(count), 0x00)
		frame = append(frame, data...)
		return frame
	}
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
	case Family1E:
		if len(response) < 2 {
			return nil, fmt.Errorf("MC 1E response is too short")
		}
		if response[0] < 0x80 || response[0] > 0x83 {
			return nil, fmt.Errorf("MC 1E response subheader 0x%02X mismatch", response[0])
		}
		if response[1] != 0x00 {
			return nil, fmt.Errorf("MC 1E response end code 0x%02X", response[1])
		}
		return response[2:], nil
	case Family3E:
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
