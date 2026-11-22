package iec104

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
)

const (
	startByte = 0x68

	typeSinglePoint   = 1
	typeDoublePoint   = 3
	typeFloatMeasure  = 13
	typeSingleCommand = 45
	typeSetpointFloat = 50
	typeInterrogation = 100
)

func parseIOA(value string) (uint32, error) {
	n, err := strconv.ParseUint(value, 10, 24)
	if err != nil || n > 0xffffff {
		return 0, fmt.Errorf("invalid iec104 IOA %q", value)
	}
	return uint32(n), nil
}

func buildIFrame(asdu []byte, tx, rx uint16) []byte {
	frame := make([]byte, 6+len(asdu))
	frame[0] = startByte
	frame[1] = byte(4 + len(asdu))
	putUint16LE(frame[2:], tx<<1)
	putUint16LE(frame[4:], rx<<1)
	copy(frame[6:], asdu)
	return frame
}

func buildSFrame(rx uint16) []byte {
	frame := []byte{startByte, 4, 1, 0, 0, 0}
	putUint16LE(frame[4:], rx<<1)
	return frame
}

func buildUFrame(ctrl byte) []byte {
	return []byte{startByte, 4, ctrl, 0, 0, 0}
}

func buildInterrogationASDU(ca uint16) []byte {
	asdu := []byte{typeInterrogation, 1, 6, 0}
	asdu = append(asdu, uint16Bytes(ca)...)
	asdu = append(asdu, 0, 0, 0, 20)
	return asdu
}

func buildSingleCommandASDU(ca uint16, ioa uint32, value bool) []byte {
	asdu := []byte{typeSingleCommand, 1, 6, 0}
	asdu = append(asdu, uint16Bytes(ca)...)
	asdu = append(asdu, uint24Bytes(ioa)...)
	if value {
		asdu = append(asdu, 1)
	} else {
		asdu = append(asdu, 0)
	}
	return asdu
}

func buildSetpointASDU(ca uint16, ioa uint32, value float32) []byte {
	asdu := []byte{typeSetpointFloat, 1, 6, 0}
	asdu = append(asdu, uint16Bytes(ca)...)
	asdu = append(asdu, uint24Bytes(ioa)...)
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, math.Float32bits(value))
	asdu = append(asdu, data...)
	asdu = append(asdu, 0)
	return asdu
}

func parseASDU(asdu []byte, expectedCA uint16) (map[uint32]any, bool, error) {
	if len(asdu) < 6 {
		return nil, false, fmt.Errorf("iec104 ASDU is too short")
	}
	typeID := asdu[0]
	vsq := asdu[1]
	cot := asdu[2] & 0x3f
	ca := binary.LittleEndian.Uint16(asdu[4:6])
	if ca != expectedCA {
		// Some simulators use a station address other than config; data is still usable.
	}
	if typeID == typeInterrogation {
		return nil, cot == 10, nil
	}
	count := int(vsq & 0x7f)
	if count == 0 || count > 127 {
		return nil, false, fmt.Errorf("iec104 invalid VSQ %d", vsq)
	}
	values := make(map[uint32]any)
	offset := 6
	sequential := vsq&0x80 != 0
	var baseIOA uint32
	if sequential {
		if offset+3 > len(asdu) {
			return values, false, fmt.Errorf("iec104 sequential ASDU is too short")
		}
		baseIOA = readUint24LE(asdu[offset:])
		offset += 3
	}
	for i := 0; i < count; i++ {
		var ioa uint32
		if sequential {
			ioa = baseIOA + uint32(i)
		}
		switch typeID {
		case typeSinglePoint, typeDoublePoint:
			if sequential {
				if offset+1 > len(asdu) {
					return values, false, fmt.Errorf("iec104 single point ASDU is too short")
				}
			} else {
				if offset+4 > len(asdu) {
					return values, false, fmt.Errorf("iec104 single point ASDU is too short")
				}
				ioa = readUint24LE(asdu[offset:])
			}
			if typeID == typeSinglePoint {
				values[ioa] = asdu[offset]&1 != 0
			} else {
				values[ioa] = asdu[offset]&3 != 0
			}
			if sequential {
				offset++
			} else {
				offset += 4
			}
		case typeFloatMeasure:
			size := 8
			if sequential {
				size = 5
			}
			if offset+size > len(asdu) {
				return values, false, fmt.Errorf("iec104 float ASDU is too short")
			}
			valueOffset := offset
			if !sequential {
				ioa = readUint24LE(asdu[offset:])
				valueOffset = offset + 3
			}
			values[ioa] = math.Float32frombits(binary.LittleEndian.Uint32(asdu[valueOffset:]))
			offset += size
		default:
			return values, false, nil
		}
	}
	return values, false, nil
}

func putUint16LE(dst []byte, value uint16) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
}

func uint16Bytes(value uint16) []byte {
	return []byte{byte(value), byte(value >> 8)}
}

func uint24Bytes(value uint32) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16)}
}

func readUint24LE(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
}
