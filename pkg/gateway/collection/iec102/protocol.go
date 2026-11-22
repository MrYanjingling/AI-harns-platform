package iec102

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	startFixed = 0x10
	startVar   = 0x68
	endByte    = 0x16
)

func parseAddress(value string) (uint8, uint8, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("iec102 address must use RA.IOA format")
	}
	ra, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid iec102 RA %q", value)
	}
	ioa, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid iec102 IOA %q", value)
	}
	return uint8(ra), uint8(ioa), nil
}

func buildFixedFrame(ctrl uint8, link uint16) []byte {
	cs := ctrl + uint8(link) + uint8(link>>8)
	return []byte{startFixed, ctrl, byte(link), byte(link >> 8), cs, endByte}
}

func buildVariableFrame(ctrl uint8, link uint16, asdu []byte) []byte {
	length := 3 + len(asdu)
	frame := []byte{startVar, byte(length), byte(length), startVar, ctrl, byte(link), byte(link >> 8)}
	frame = append(frame, asdu...)
	var cs uint8
	for _, b := range frame[4:] {
		cs += b
	}
	return append(frame, cs, endByte)
}

func readLinkFrame(r io.Reader) ([]byte, error) {
	head := make([]byte, 1)
	if err := collection.ReadFullWithTimeout(r, head); err != nil {
		return nil, err
	}
	switch head[0] {
	case startFixed:
		rest := make([]byte, 5)
		if err := collection.ReadFullWithTimeout(r, rest); err != nil {
			return nil, err
		}
		if rest[4] != endByte || uint8(rest[0]+rest[1]+rest[2]) != rest[3] {
			return nil, fmt.Errorf("invalid iec102 fixed frame")
		}
		return nil, nil
	case startVar:
		rest := make([]byte, 3)
		if err := collection.ReadFullWithTimeout(r, rest); err != nil {
			return nil, err
		}
		length := int(rest[0])
		if rest[1] != rest[0] || rest[2] != startVar || length < 3 {
			return nil, fmt.Errorf("invalid iec102 variable frame header")
		}
		body := make([]byte, length+2)
		if err := collection.ReadFullWithTimeout(r, body); err != nil {
			return nil, err
		}
		if body[length+1] != endByte {
			return nil, fmt.Errorf("invalid iec102 variable frame end")
		}
		all := append([]byte{startVar, rest[0], rest[1], startVar}, body...)
		var cs uint8
		for _, b := range all[4 : 4+length] {
			cs += b
		}
		if cs != body[length] {
			return nil, fmt.Errorf("invalid iec102 variable frame checksum")
		}
		return body[3:length], nil
	default:
		return nil, fmt.Errorf("invalid iec102 frame start")
	}
}

func parseASDU(asdu []byte) (map[string]any, error) {
	if len(asdu) < 7 {
		return nil, fmt.Errorf("iec102 ASDU is too short")
	}
	typeID := asdu[0]
	count := int(asdu[1] & 0x7f)
	if count == 0 || count > 127 {
		return nil, fmt.Errorf("iec102 invalid VSQ %d", asdu[1])
	}
	ra := asdu[5]
	offset := 6
	values := make(map[string]any)
	for i := 0; i < count; i++ {
		if offset >= len(asdu) {
			return nil, fmt.Errorf("iec102 IOA is missing")
		}
		ioa := asdu[offset]
		offset++
		var size int
		switch typeID {
		case 2, 5, 8, 11:
			size = 5
		case 3, 6, 9, 12:
			size = 4
		case 4, 7, 10, 13:
			size = 3
		case 1:
			size = 1
		default:
			return values, nil
		}
		if offset+size > len(asdu) {
			return nil, fmt.Errorf("iec102 object data is too short")
		}
		switch typeID {
		case 1:
			values[fmt.Sprintf("%d.%d", ra, ioa)] = asdu[offset]&1 != 0
		default:
			values[fmt.Sprintf("%d.%d", ra, ioa)] = binary.LittleEndian.Uint32(append(asdu[offset:offset+size], 0, 0, 0, 0))
		}
		offset += size
	}
	return values, nil
}
