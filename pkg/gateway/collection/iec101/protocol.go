package iec101

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"

	"harns-platform/pkg/gateway/collection"
)

const (
	startFixed        = 0x10
	startVar          = 0x68
	endByte           = 0x16
	typeSinglePoint   = 1
	typeFloatMeasure  = 13
	typeSingleCommand = 45
	typeInterrogation = 100
)

func parseIOA(value string) (uint32, error) {
	n, err := strconv.ParseUint(value, 10, 24)
	if err != nil || n > 0xffffff {
		return 0, fmt.Errorf("invalid iec101 IOA %q", value)
	}
	return uint32(n), nil
}

func buildFixedFrame(ctrl, link uint8) []byte {
	cs := ctrl + link
	return []byte{startFixed, ctrl, link, cs, endByte}
}

func buildVariableFrame(ctrl, link uint8, asdu []byte) []byte {
	length := 2 + len(asdu)
	frame := make([]byte, 0, 6+length)
	frame = append(frame, startVar, byte(length), byte(length), startVar, ctrl, link)
	frame = append(frame, asdu...)
	var cs uint8
	for _, b := range frame[4:] {
		cs += b
	}
	frame = append(frame, cs, endByte)
	return frame
}

func buildInterrogationASDU(ca, caLen, ioaLen int) []byte {
	asdu := []byte{typeInterrogation, 1, 6, 0}
	asdu = appendCA(asdu, uint32(ca), caLen)
	asdu = appendIOA(asdu, 0, ioaLen)
	asdu = append(asdu, 20)
	return asdu
}

func buildSingleCommandASDU(ca, caLen, ioaLen int, ioa uint32, value bool) []byte {
	asdu := []byte{typeSingleCommand, 1, 6, 0}
	asdu = appendCA(asdu, uint32(ca), caLen)
	asdu = appendIOA(asdu, ioa, ioaLen)
	if value {
		asdu = append(asdu, 1)
	} else {
		asdu = append(asdu, 0)
	}
	return asdu
}

func buildSetpointASDU(ca, caLen, ioaLen int, ioa uint32, value float32) []byte {
	asdu := []byte{50, 1, 6, 0}
	asdu = appendCA(asdu, uint32(ca), caLen)
	asdu = appendIOA(asdu, ioa, ioaLen)
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, math.Float32bits(value))
	return append(asdu, append(data, 0)...)
}

func appendCA(asdu []byte, ca uint32, length int) []byte {
	if length == 2 {
		return append(asdu, byte(ca), byte(ca>>8))
	}
	return append(asdu, byte(ca))
}

func appendIOA(asdu []byte, ioa uint32, length int) []byte {
	switch length {
	case 1:
		return append(asdu, byte(ioa))
	case 3:
		return append(asdu, byte(ioa), byte(ioa>>8), byte(ioa>>16))
	default:
		return append(asdu, byte(ioa), byte(ioa>>8))
	}
}

func parseASDU(asdu []byte, caLen, ioaLen int) (map[uint32]any, bool, error) {
	if len(asdu) < 4+caLen {
		return nil, false, fmt.Errorf("iec101 ASDU is too short")
	}
	typeID := asdu[0]
	vsq := asdu[1]
	cot := asdu[2] & 0x3f
	if typeID == typeInterrogation {
		return nil, cot == 10, nil
	}
	count := int(vsq & 0x7f)
	if count == 0 || count > 127 {
		return nil, false, fmt.Errorf("iec101 invalid VSQ %d", vsq)
	}
	offset := 4 + caLen
	values := make(map[uint32]any)
	sequential := vsq&0x80 != 0
	var baseIOA uint32
	if sequential {
		if offset+ioaLen > len(asdu) {
			return values, false, fmt.Errorf("iec101 sequential IOA is too short")
		}
		baseIOA = readIOA(asdu[offset:], ioaLen)
		offset += ioaLen
	}
	for i := 0; i < count; i++ {
		var ioa uint32
		if sequential {
			ioa = baseIOA + uint32(i)
		} else {
			if offset+ioaLen > len(asdu) {
				return values, false, fmt.Errorf("iec101 IOA is too short")
			}
			ioa = readIOA(asdu[offset:], ioaLen)
			offset += ioaLen
		}
		switch typeID {
		case typeSinglePoint:
			if offset >= len(asdu) {
				return values, false, fmt.Errorf("iec101 single point is too short")
			}
			values[ioa] = asdu[offset]&1 != 0
			offset++
		case typeFloatMeasure:
			if offset+5 > len(asdu) {
				return values, false, fmt.Errorf("iec101 float measure is too short")
			}
			values[ioa] = math.Float32frombits(binary.LittleEndian.Uint32(asdu[offset:]))
			offset += 5
		default:
			return values, false, nil
		}
	}
	return values, false, nil
}

func readIOA(data []byte, length int) uint32 {
	switch length {
	case 1:
		return uint32(data[0])
	case 3:
		return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
	default:
		return uint32(data[0]) | uint32(data[1])<<8
	}
}

func readLinkFrame(r io.Reader) (byte, []byte, error) {
	head := make([]byte, 1)
	if err := collection.ReadFullWithTimeout(r, head); err != nil {
		return 0, nil, err
	}
	switch head[0] {
	case startFixed:
		rest := make([]byte, 4)
		if err := collection.ReadFullWithTimeout(r, rest); err != nil {
			return 0, nil, err
		}
		if rest[3] != endByte {
			return 0, nil, fmt.Errorf("invalid iec101 fixed frame end")
		}
		if uint8(rest[0]+rest[1]) != rest[2] {
			return 0, nil, fmt.Errorf("invalid iec101 fixed frame checksum")
		}
		return rest[0], nil, nil
	case startVar:
		head2 := make([]byte, 3)
		if err := collection.ReadFullWithTimeout(r, head2); err != nil {
			return 0, nil, err
		}
		length := int(head2[0])
		if head2[1] != head2[0] || head2[2] != startVar || length < 2 {
			return 0, nil, fmt.Errorf("invalid iec101 variable frame header")
		}
		body := make([]byte, length+2)
		if err := collection.ReadFullWithTimeout(r, body); err != nil {
			return 0, nil, err
		}
		if body[length+1] != endByte {
			return 0, nil, fmt.Errorf("invalid iec101 variable frame end")
		}
		all := append([]byte{startVar, head2[0], head2[1], startVar}, body...)
		var cs uint8
		for _, b := range all[4 : 4+length] {
			cs += b
		}
		if cs != body[length] {
			return 0, nil, fmt.Errorf("invalid iec101 variable frame checksum")
		}
		return body[0], body[2 : 2+length-2], nil
	default:
		return 0, nil, fmt.Errorf("invalid iec101 frame start")
	}
}
