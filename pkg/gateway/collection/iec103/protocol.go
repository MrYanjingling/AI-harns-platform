package iec103

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	startFixed = 0x10
	startVar   = 0x68
	endByte    = 0x16
)

func parseAddress(value string) (uint8, uint8, uint8, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("iec103 address must use CA.FUN.NUM format")
	}
	var v [3]uint8
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 8)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid iec103 address %q", value)
		}
		v[i] = uint8(n)
	}
	return v[0], v[1], v[2], nil
}

func buildFixedFrame(ctrl, link uint8) []byte {
	return []byte{startFixed, ctrl, link, ctrl + link, endByte}
}

func buildVariableFrame(ctrl, link uint8, asdu []byte) []byte {
	length := 3 + len(asdu)
	frame := []byte{startVar, byte(length), byte(length), startVar, ctrl, link, 0}
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
		rest := make([]byte, 4)
		if err := collection.ReadFullWithTimeout(r, rest); err != nil {
			return nil, err
		}
		if rest[3] != endByte || uint8(rest[0]+rest[1]) != rest[2] {
			return nil, fmt.Errorf("invalid iec103 fixed frame")
		}
		return nil, nil
	case startVar:
		rest := make([]byte, 3)
		if err := collection.ReadFullWithTimeout(r, rest); err != nil {
			return nil, err
		}
		length := int(rest[0])
		if rest[1] != rest[0] || rest[2] != startVar || length < 3 {
			return nil, fmt.Errorf("invalid iec103 variable header")
		}
		body := make([]byte, length+2)
		if err := collection.ReadFullWithTimeout(r, body); err != nil {
			return nil, err
		}
		if body[length+1] != endByte {
			return nil, fmt.Errorf("invalid iec103 variable end")
		}
		all := append([]byte{startVar, rest[0], rest[1], startVar}, body...)
		var cs uint8
		for _, b := range all[4 : 4+length] {
			cs += b
		}
		if cs != body[length] {
			return nil, fmt.Errorf("invalid iec103 variable checksum")
		}
		return body[3:length], nil
	default:
		return nil, fmt.Errorf("invalid iec103 frame start")
	}
}

func parseASDU(asdu []byte) (map[string]any, error) {
	if len(asdu) < 6 {
		return nil, fmt.Errorf("iec103 ASDU is too short")
	}
	typeID := asdu[0]
	count := int(asdu[1] & 0x7f)
	if count == 0 || count > 127 {
		return nil, fmt.Errorf("iec103 invalid VSQ %d", asdu[1])
	}
	ca, fun, num := asdu[3], asdu[4], asdu[5]
	offset := 6
	values := make(map[string]any)
	for i := 0; i < count; i++ {
		var size int
		switch typeID {
		case 1, 2, 7:
			size = 1
		case 3, 9:
			size = 3
		case 4:
			size = 4
		case 5:
			size = 13
		default:
			return values, nil
		}
		if offset+size > len(asdu) {
			return nil, fmt.Errorf("iec103 object is too short")
		}
		data := asdu[offset : offset+size]
		key := fmt.Sprintf("%d.%d.%d", ca, fun, int(num)+i)
		switch typeID {
		case 3, 9:
			values[key] = int32((int(data[2]&3) << 13) | (int(data[0]) >> 3) | (int(data[1]) << 5))
		case 1, 2:
			values[key] = data[0] & 0x03
		case 4:
			values[key] = math.Float32frombits(binary.LittleEndian.Uint32(data))
		case 5:
			values[key] = strings.TrimSpace(string(data[1:13]))
		default:
			values[key] = data[0]
		}
		offset += size
	}
	return values, nil
}
