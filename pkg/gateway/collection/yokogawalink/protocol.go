package yokogawalink

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	cmdBitRead   byte = 0x01
	cmdBitWrite  byte = 0x02
	cmdWordRead  byte = 0x11
	cmdWordWrite byte = 0x12

	maxBatchWords = 64
	maxBatchBits  = 256
)

type areaSpec struct {
	code  uint16
	isBit bool
}

// Device codes per Yokogawa FA-M3 Personal Computer Link Manual (IM 34M6H41-02E).
var areaTable = map[byte]areaSpec{
	'X': {0x0018, true},
	'Y': {0x0019, true},
	'I': {0x0009, true},
	'E': {0x0005, true},
	'L': {0x000C, true},
	'M': {0x000D, true},
	'D': {0x0004, false},
	'B': {0x0002, false},
	'F': {0x0006, false},
	'R': {0x0012, false},
	'W': {0x0017, false},
	'Z': {0x001A, false},
}

type pointAddress struct {
	Area      byte
	AreaCode  uint16
	Offset    uint32
	Bit       int
	IsBitArea bool
	Count     int // words for word area, 1 for bit area
	DataType  string
}

func parseAddress(raw, dataType string) (*pointAddress, error) {
	text := strings.ToUpper(strings.TrimSpace(raw))
	if len(text) < 2 {
		return nil, fmt.Errorf("invalid yokogawa address %q", raw)
	}
	spec, ok := areaTable[text[0]]
	if !ok {
		return nil, fmt.Errorf("unsupported yokogawa area in %q", raw)
	}
	rest := text[1:]
	bit := 0
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		b, err := strconv.Atoi(rest[dot+1:])
		if err != nil || b < 0 || b > 15 {
			return nil, fmt.Errorf("invalid yokogawa bit index in %q", raw)
		}
		bit = b
		rest = rest[:dot]
	}
	num, err := strconv.ParseUint(rest, 10, 32)
	if err != nil || num > 999999 {
		return nil, fmt.Errorf("invalid yokogawa offset in %q", raw)
	}
	kind := strings.ToLower(strings.TrimSpace(dataType))
	addr := &pointAddress{
		Area:      text[0],
		AreaCode:  spec.code,
		Offset:    uint32(num),
		Bit:       bit,
		IsBitArea: spec.isBit,
		DataType:  kind,
	}
	switch kind {
	case "bool", "bit":
		addr.Count = 1
	case "int16", "uint16":
		addr.Count = 1
	case "int32", "uint32", "float32":
		addr.Count = 2
	case "int64", "uint64", "float64":
		addr.Count = 4
	case "string":
		addr.Count = 8
	default:
		return nil, fmt.Errorf("unsupported yokogawa data type %q", dataType)
	}
	return addr, nil
}

type readBatch struct {
	area      byte
	areaCode  uint16
	isBitArea bool
	start     uint32
	count     int
	items     []*batchItem
}

type batchItem struct {
	variable *collection.Variable
	addr     *pointAddress
	offset   int
}

func buildReadBatches(variables []*collection.Variable) ([]*readBatch, error) {
	items := make([]*batchItem, 0, len(variables))
	for _, v := range variables {
		if v == nil || v.IsCalculated || v.AccessMode == collection.AccessModeWrite {
			continue
		}
		addr, err := parseAddress(v.Address, v.DataType)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", v.Name, err)
		}
		items = append(items, &batchItem{variable: v, addr: addr})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].addr.Area != items[j].addr.Area {
			return items[i].addr.Area < items[j].addr.Area
		}
		return items[i].addr.Offset < items[j].addr.Offset
	})
	var batches []*readBatch
	for _, item := range items {
		a := item.addr
		maxSpan := maxBatchWords
		if a.IsBitArea {
			maxSpan = maxBatchBits
		}
		if len(batches) > 0 {
			last := batches[len(batches)-1]
			lastEnd := last.start + uint32(last.count)
			newEnd := a.Offset + uint32(a.Count)
			if last.area == a.Area && last.isBitArea == a.IsBitArea && a.Offset <= lastEnd && int(newEnd-last.start) <= maxSpan {
				item.offset = int(a.Offset - last.start)
				if newEnd > lastEnd {
					last.count = int(newEnd - last.start)
				}
				last.items = append(last.items, item)
				continue
			}
		}
		item.offset = 0
		batches = append(batches, &readBatch{
			area:      a.Area,
			areaCode:  a.AreaCode,
			isBitArea: a.IsBitArea,
			start:     a.Offset,
			count:     a.Count,
			items:     []*batchItem{item},
		})
	}
	return batches, nil
}

func buildReadRequest(cpu int, b *readBatch) []byte {
	cmd := cmdWordRead
	if b.isBitArea {
		cmd = cmdBitRead
	}
	frame := make([]byte, 12)
	frame[0] = cmd
	frame[1] = byte(cpu)
	binary.BigEndian.PutUint16(frame[2:4], 8)
	binary.BigEndian.PutUint16(frame[4:6], b.areaCode)
	binary.BigEndian.PutUint32(frame[6:10], b.start)
	binary.BigEndian.PutUint16(frame[10:12], uint16(b.count))
	return frame
}

func buildWriteRequest(cpu int, addr *pointAddress, data []byte) []byte {
	cmd := cmdWordWrite
	if addr.IsBitArea {
		cmd = cmdBitWrite
	}
	frame := make([]byte, 12+len(data))
	frame[0] = cmd
	frame[1] = byte(cpu)
	binary.BigEndian.PutUint16(frame[2:4], uint16(8+len(data)))
	binary.BigEndian.PutUint16(frame[4:6], addr.AreaCode)
	binary.BigEndian.PutUint32(frame[6:10], addr.Offset)
	binary.BigEndian.PutUint16(frame[10:12], uint16(addr.Count))
	copy(frame[12:], data)
	return frame
}

func parseResponse(expectedCmd byte, resp []byte) ([]byte, error) {
	if len(resp) < 4 {
		return nil, fmt.Errorf("yokogawa response is too short (%d bytes)", len(resp))
	}
	if resp[0] != (expectedCmd | 0x80) {
		return nil, fmt.Errorf("yokogawa response cmd 0x%02X mismatch, want 0x%02X", resp[0], expectedCmd|0x80)
	}
	if resp[1] != 0x00 {
		return nil, fmt.Errorf("yokogawa PLC error code 0x%02X", resp[1])
	}
	dataLen := int(binary.BigEndian.Uint16(resp[2:4]))
	if len(resp) < 4+dataLen {
		return nil, fmt.Errorf("yokogawa response truncated: got %d, want %d", len(resp)-4, dataLen)
	}
	return resp[4 : 4+dataLen], nil
}

func decodeValue(addr *pointAddress, payload []byte, offset int) (any, error) {
	if addr.IsBitArea {
		if offset >= len(payload) {
			return nil, fmt.Errorf("yokogawa bit payload too short")
		}
		return payload[offset] != 0, nil
	}
	byteOff := offset * 2
	need := addr.Count * 2
	if byteOff+need > len(payload) {
		return nil, fmt.Errorf("yokogawa word payload too short")
	}
	data := payload[byteOff : byteOff+need]
	switch addr.DataType {
	case "bool", "bit":
		w := binary.BigEndian.Uint16(data)
		return (w & (1 << uint(addr.Bit))) != 0, nil
	case "int16":
		return int16(binary.BigEndian.Uint16(data)), nil
	case "uint16":
		return binary.BigEndian.Uint16(data), nil
	case "int32":
		return int32(binary.BigEndian.Uint32(data)), nil
	case "uint32":
		return binary.BigEndian.Uint32(data), nil
	case "int64":
		return int64(binary.BigEndian.Uint64(data)), nil
	case "uint64":
		return binary.BigEndian.Uint64(data), nil
	case "float32":
		return math.Float32frombits(binary.BigEndian.Uint32(data)), nil
	case "float64":
		return math.Float64frombits(binary.BigEndian.Uint64(data)), nil
	case "string":
		return strings.TrimRight(string(data), "\x00"), nil
	default:
		return nil, fmt.Errorf("unsupported yokogawa data type %q", addr.DataType)
	}
}

func encodeValue(addr *pointAddress, value any) ([]byte, error) {
	if addr.IsBitArea {
		b, ok := toBool(value)
		if !ok {
			return nil, fmt.Errorf("invalid bool %v", value)
		}
		if b {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	}
	switch addr.DataType {
	case "int16", "uint16":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(n))
		return buf, nil
	case "int32", "uint32":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(n))
		return buf, nil
	case "int64", "uint64":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(n))
		return buf, nil
	case "float32":
		f, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("invalid float %v", value)
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, math.Float32bits(float32(f)))
		return buf, nil
	case "float64":
		f, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("invalid float %v", value)
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, math.Float64bits(f))
		return buf, nil
	case "string":
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string %v", value)
		}
		buf := make([]byte, addr.Count*2)
		copy(buf, []byte(s))
		return buf, nil
	default:
		return nil, fmt.Errorf("unsupported yokogawa write data type %q", addr.DataType)
	}
}

func toBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case int:
		return x != 0, true
	case int64:
		return x != 0, true
	case float64:
		return x != 0, true
	default:
		return false, false
	}
}

func toInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint:
		return int64(x), true
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		return int64(x), true
	case float32:
		return int64(x), true
	case float64:
		return int64(x), true
	default:
		return 0, false
	}
}

func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float32:
		return float64(x), true
	case float64:
		return x, true
	default:
		n, ok := toInt64(v)
		return float64(n), ok
	}
}
