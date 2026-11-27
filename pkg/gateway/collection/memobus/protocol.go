package memobus

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
	headerSize  = 12
	msgTypeCmd  = 0x11
	msgTypeResp = 0x19

	mfcExtended byte = 0x20

	sfcReadHoldReg    byte = 0x09
	sfcWriteHoldReg   byte = 0x0B
	sfcReadHoldRegExt byte = 0x0D
	sfcWriteHoldRegEx byte = 0x0E

	maxBatchRegs = 100
)

type pointAddress struct {
	Reg      uint32
	Bit      int
	HasBit   bool
	Words    int
	DataType string
}

// parseAddress parses Yaskawa MP register addresses such as MW0, ML10, MF20, MB0.3,
// or direct numeric register offsets (0, 100, 40001).
func parseAddress(raw, dataType string) (*pointAddress, error) {
	text := strings.ToUpper(strings.TrimSpace(raw))
	if text == "" {
		return nil, fmt.Errorf("memobus address is empty")
	}
	for _, prefix := range []string{"MW", "ML", "MF", "MB", "IW", "OW", "SW", "DW", "M", "D", "R"} {
		if strings.HasPrefix(text, prefix) {
			text = text[len(prefix):]
			break
		}
	}
	bit := 0
	hasBit := false
	if dot := strings.IndexByte(text, '.'); dot >= 0 {
		b, err := strconv.Atoi(text[dot+1:])
		if err != nil || b < 0 || b > 15 {
			return nil, fmt.Errorf("invalid memobus bit index in %q", raw)
		}
		bit = b
		hasBit = true
		text = text[:dot]
	}
	num, err := strconv.ParseUint(text, 10, 32)
	if err != nil || num > 0xFFFFF {
		return nil, fmt.Errorf("invalid memobus register offset in %q", raw)
	}
	kind := strings.ToLower(strings.TrimSpace(dataType))
	addr := &pointAddress{
		Reg:      uint32(num),
		Bit:      bit,
		HasBit:   hasBit,
		DataType: kind,
	}
	switch kind {
	case "bool", "bit":
		addr.Words = 1
	case "int16", "uint16":
		addr.Words = 1
	case "int32", "uint32", "float32":
		addr.Words = 2
	case "int64", "uint64", "float64":
		addr.Words = 4
	case "string":
		addr.Words = 8
	default:
		return nil, fmt.Errorf("unsupported memobus data type %q", dataType)
	}
	return addr, nil
}

type readBatch struct {
	start uint32
	count int
	items []*batchItem
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
		return items[i].addr.Reg < items[j].addr.Reg
	})
	var batches []*readBatch
	for _, item := range items {
		a := item.addr
		if len(batches) > 0 {
			last := batches[len(batches)-1]
			lastEnd := last.start + uint32(last.count)
			newEnd := a.Reg + uint32(a.Words)
			if a.Reg <= lastEnd && int(newEnd-last.start) <= maxBatchRegs {
				item.offset = int(a.Reg - last.start)
				if newEnd > lastEnd {
					last.count = int(newEnd - last.start)
				}
				last.items = append(last.items, item)
				continue
			}
		}
		item.offset = 0
		batches = append(batches, &readBatch{
			start: a.Reg,
			count: a.Words,
			items: []*batchItem{item},
		})
	}
	return batches, nil
}

func wrap218IF(serial byte, pdu []byte) []byte {
	total := headerSize + len(pdu)
	frame := make([]byte, total)
	frame[0] = msgTypeCmd
	frame[1] = serial
	binary.LittleEndian.PutUint16(frame[6:8], uint16(total))
	copy(frame[headerSize:], pdu)
	return frame
}

func buildReadRequest(serial byte, cpu int, start uint32, count int) []byte {
	if start > 0xFFFF {
		// Extended 32-bit address SFC=0x0D: len(2) + MFC(1) + SFC(1) + CPU(2) + addr(4) + count(2) = 12B
		pdu := make([]byte, 12)
		binary.LittleEndian.PutUint16(pdu[0:2], 10)
		pdu[2] = mfcExtended
		pdu[3] = sfcReadHoldRegExt
		binary.LittleEndian.PutUint16(pdu[4:6], uint16(cpu))
		binary.LittleEndian.PutUint32(pdu[6:10], start)
		binary.LittleEndian.PutUint16(pdu[10:12], uint16(count))
		return wrap218IF(serial, pdu)
	}
	// Standard 16-bit address SFC=0x09: len(2) + MFC(1) + SFC(1) + CPU(2) + addr(2) + count(2) = 10B
	pdu := make([]byte, 10)
	binary.LittleEndian.PutUint16(pdu[0:2], 8)
	pdu[2] = mfcExtended
	pdu[3] = sfcReadHoldReg
	binary.LittleEndian.PutUint16(pdu[4:6], uint16(cpu))
	binary.LittleEndian.PutUint16(pdu[6:8], uint16(start))
	binary.LittleEndian.PutUint16(pdu[8:10], uint16(count))
	return wrap218IF(serial, pdu)
}

func buildWriteRequest(serial byte, cpu int, start uint32, count int, data []byte) []byte {
	if start > 0xFFFF {
		pdu := make([]byte, 12+len(data))
		binary.LittleEndian.PutUint16(pdu[0:2], uint16(10+len(data)))
		pdu[2] = mfcExtended
		pdu[3] = sfcWriteHoldRegEx
		binary.LittleEndian.PutUint16(pdu[4:6], uint16(cpu))
		binary.LittleEndian.PutUint32(pdu[6:10], start)
		binary.LittleEndian.PutUint16(pdu[10:12], uint16(count))
		copy(pdu[12:], data)
		return wrap218IF(serial, pdu)
	}
	pdu := make([]byte, 10+len(data))
	binary.LittleEndian.PutUint16(pdu[0:2], uint16(8+len(data)))
	pdu[2] = mfcExtended
	pdu[3] = sfcWriteHoldReg
	binary.LittleEndian.PutUint16(pdu[4:6], uint16(cpu))
	binary.LittleEndian.PutUint16(pdu[6:8], uint16(start))
	binary.LittleEndian.PutUint16(pdu[8:10], uint16(count))
	copy(pdu[10:], data)
	return wrap218IF(serial, pdu)
}

func parseReadResponse(resp []byte, expectedCount int) ([]byte, error) {
	if len(resp) < headerSize+6 {
		return nil, fmt.Errorf("memobus response is too short (%d bytes)", len(resp))
	}
	if resp[0] != msgTypeCmd && resp[0] != msgTypeResp {
		return nil, fmt.Errorf("unexpected memobus msgType 0x%02X", resp[0])
	}
	pdu := resp[headerSize:]
	if pdu[2]&0x80 != 0 || pdu[3]&0x80 != 0 {
		return nil, fmt.Errorf("memobus exception MFC=0x%02X SFC=0x%02X", pdu[2], pdu[3])
	}
	// Response PDU: len(2) + MFC(1) + SFC(1) + CPU(2) + count(2) + regData[count*2]
	if len(pdu) < 8+expectedCount*2 {
		return nil, fmt.Errorf("memobus read data truncated: got %d, want %d", len(pdu)-8, expectedCount*2)
	}
	return pdu[8 : 8+expectedCount*2], nil
}

func parseWriteResponse(resp []byte) error {
	if len(resp) < headerSize+4 {
		return fmt.Errorf("memobus write response is too short (%d bytes)", len(resp))
	}
	if resp[0] != msgTypeCmd && resp[0] != msgTypeResp {
		return fmt.Errorf("unexpected memobus write msgType 0x%02X", resp[0])
	}
	pdu := resp[headerSize:]
	if pdu[2]&0x80 != 0 || pdu[3]&0x80 != 0 {
		return fmt.Errorf("memobus write exception MFC=0x%02X SFC=0x%02X", pdu[2], pdu[3])
	}
	return nil
}

func decodeValue(addr *pointAddress, payload []byte, offset int) (any, error) {
	byteOff := offset * 2
	need := addr.Words * 2
	if byteOff+need > len(payload) {
		return nil, fmt.Errorf("memobus payload too short")
	}
	data := payload[byteOff : byteOff+need]
	switch addr.DataType {
	case "bool", "bit":
		w := binary.LittleEndian.Uint16(data)
		return (w & (1 << uint(addr.Bit))) != 0, nil
	case "int16":
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "uint16":
		return binary.LittleEndian.Uint16(data), nil
	case "int32":
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "uint32":
		return binary.LittleEndian.Uint32(data), nil
	case "int64":
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "uint64":
		return binary.LittleEndian.Uint64(data), nil
	case "float32":
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "float64":
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	case "string":
		return strings.TrimRight(string(data), "\x00"), nil
	default:
		return nil, fmt.Errorf("unsupported memobus data type %q", addr.DataType)
	}
}

func encodeValue(addr *pointAddress, value any) ([]byte, error) {
	switch addr.DataType {
	case "bool", "bit":
		b, ok := toBool(value)
		if !ok {
			return nil, fmt.Errorf("invalid bool %v", value)
		}
		buf := make([]byte, 2)
		if b {
			binary.LittleEndian.PutUint16(buf, 1<<uint(addr.Bit))
		}
		return buf, nil
	case "int16", "uint16":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(n))
		return buf, nil
	case "int32", "uint32":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(n))
		return buf, nil
	case "int64", "uint64":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(n))
		return buf, nil
	case "float32":
		f, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("invalid float %v", value)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, math.Float32bits(float32(f)))
		return buf, nil
	case "float64":
		f, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("invalid float %v", value)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, math.Float64bits(f))
		return buf, nil
	case "string":
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string %v", value)
		}
		buf := make([]byte, addr.Words*2)
		copy(buf, []byte(s))
		return buf, nil
	default:
		return nil, fmt.Errorf("unsupported memobus write data type %q", addr.DataType)
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
