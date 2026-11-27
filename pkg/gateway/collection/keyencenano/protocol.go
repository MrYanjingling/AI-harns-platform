package keyencenano

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const maxBatchWords = 100

type deviceSpec struct {
	prefix string
	isBit  bool
	isHex  bool
}

var deviceSpecs = []deviceSpec{
	{"DM", false, false},
	{"EM", false, false},
	{"FM", false, false},
	{"ZF", false, false},
	{"CM", false, false},
	{"TM", false, false},
	{"MR", true, false},
	{"LR", true, false},
	{"CR", true, false},
	{"VB", true, true},
	{"W", false, true},
	{"R", true, false},
	{"B", true, true},
	{"T", false, false},
	{"C", false, false},
	{"Z", false, false},
}

type pointAddress struct {
	Area      string
	Offset    int
	Bit       int
	HasBit    bool
	IsBitArea bool
	IsHex     bool
	Words     int
	DataType  string
}

func parseAddress(raw, dataType string) (*pointAddress, error) {
	text := strings.ToUpper(strings.TrimSpace(raw))
	if text == "" {
		return nil, fmt.Errorf("keyence address is empty")
	}
	var matched *deviceSpec
	for i := range deviceSpecs {
		if strings.HasPrefix(text, deviceSpecs[i].prefix) {
			matched = &deviceSpecs[i]
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("unsupported keyence device area in %q", raw)
	}
	rest := text[len(matched.prefix):]
	addrPart := rest
	bitPart := ""
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		addrPart = rest[:dot]
		bitPart = rest[dot+1:]
	}
	base := 10
	if matched.isHex {
		base = 16
	}
	offset, err := strconv.ParseInt(addrPart, base, 32)
	if err != nil || offset < 0 || offset > 999999 {
		return nil, fmt.Errorf("invalid keyence address offset %q", raw)
	}
	kind := strings.ToLower(strings.TrimSpace(dataType))
	addr := &pointAddress{
		Area:      matched.prefix,
		Offset:    int(offset),
		IsBitArea: matched.isBit,
		IsHex:     matched.isHex,
		DataType:  kind,
	}
	if bitPart != "" {
		bit, err := strconv.Atoi(bitPart)
		if err != nil || bit < 0 || bit > 15 {
			return nil, fmt.Errorf("invalid keyence bit index %q", raw)
		}
		addr.Bit = bit
		addr.HasBit = true
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
		return nil, fmt.Errorf("unsupported keyence data type %q", dataType)
	}
	return addr, nil
}

func (a *pointAddress) formatDevice(offset int) string {
	if a.IsHex {
		return fmt.Sprintf("%s%X", a.Area, offset)
	}
	return fmt.Sprintf("%s%d", a.Area, offset)
}

type readBatch struct {
	area      string
	isBitArea bool
	isHex     bool
	start     int
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
		a, b := items[i].addr, items[j].addr
		if a.Area != b.Area {
			return a.Area < b.Area
		}
		if a.IsBitArea != b.IsBitArea {
			return !a.IsBitArea
		}
		return a.Offset < b.Offset
	})
	var batches []*readBatch
	for _, item := range items {
		a := item.addr
		if len(batches) > 0 {
			last := batches[len(batches)-1]
			lastEnd := last.start + last.count
			newEnd := a.Offset + a.Words
			if last.area == a.Area && last.isBitArea == a.IsBitArea && a.Offset <= lastEnd && newEnd-last.start <= maxBatchWords {
				item.offset = a.Offset - last.start
				if newEnd > lastEnd {
					last.count = newEnd - last.start
				}
				last.items = append(last.items, item)
				continue
			}
		}
		item.offset = 0
		batches = append(batches, &readBatch{
			area:      a.Area,
			isBitArea: a.IsBitArea,
			isHex:     a.IsHex,
			start:     a.Offset,
			count:     a.Words,
			items:     []*batchItem{item},
		})
	}
	return batches, nil
}

// buildReadCommand formats an Upper Link RDS command per Keyence KV manual:
// RDS <device><addr>.U <count>\r for word devices, RDS <device><addr> <count>\r for bit devices.
func buildReadCommand(b *readBatch) []byte {
	dev := fmt.Sprintf("%s%d", b.area, b.start)
	if b.isHex {
		dev = fmt.Sprintf("%s%X", b.area, b.start)
	}
	if b.isBitArea {
		return []byte(fmt.Sprintf("RDS %s %d\r", dev, b.count))
	}
	return []byte(fmt.Sprintf("RDS %s.U %d\r", dev, b.count))
}

func buildWriteCommand(addr *pointAddress, value any) ([]byte, error) {
	dev := addr.formatDevice(addr.Offset)
	if addr.IsBitArea && (addr.DataType == "bool" || addr.DataType == "bit") {
		b, ok := toBool(value)
		if !ok {
			return nil, fmt.Errorf("invalid bool value %v", value)
		}
		v := 0
		if b {
			v = 1
		}
		return []byte(fmt.Sprintf("WR %s %d\r", dev, v)), nil
	}
	words, err := encodeWords(addr.DataType, value)
	if err != nil {
		return nil, err
	}
	if len(words) == 1 {
		return []byte(fmt.Sprintf("WR %s.U %d\r", dev, words[0])), nil
	}
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = strconv.FormatUint(uint64(w), 10)
	}
	return []byte(fmt.Sprintf("WRS %s.U %d %s\r", dev, len(words), strings.Join(parts, " "))), nil
}

func parseReadResponse(raw string, count int) ([]uint16, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, fmt.Errorf("empty keyence response")
	}
	if strings.HasPrefix(text, "E") && len(text) == 2 {
		return nil, fmt.Errorf("keyence PLC error %s", text)
	}
	fields := strings.Fields(text)
	if len(fields) < count {
		return nil, fmt.Errorf("keyence response field count %d < expected %d (%q)", len(fields), count, text)
	}
	out := make([]uint16, count)
	for i := 0; i < count; i++ {
		v, err := strconv.ParseInt(fields[i], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid keyence value %q: %w", fields[i], err)
		}
		out[i] = uint16(v)
	}
	return out, nil
}

func parseWriteResponse(raw string) error {
	text := strings.TrimSpace(raw)
	if text == "OK" {
		return nil
	}
	return fmt.Errorf("keyence write failed: %q", text)
}

func decodeWords(addr *pointAddress, words []uint16) (any, error) {
	if len(words) < addr.Words {
		return nil, fmt.Errorf("insufficient words %d < %d", len(words), addr.Words)
	}
	if addr.DataType == "bool" || addr.DataType == "bit" {
		if addr.IsBitArea {
			return words[0] != 0, nil
		}
		return (words[0] & (1 << uint(addr.Bit))) != 0, nil
	}
	buf := make([]byte, addr.Words*2)
	for i := 0; i < addr.Words; i++ {
		binary.LittleEndian.PutUint16(buf[i*2:], words[i])
	}
	switch addr.DataType {
	case "int16":
		return int16(binary.LittleEndian.Uint16(buf)), nil
	case "uint16":
		return binary.LittleEndian.Uint16(buf), nil
	case "int32":
		return int32(binary.LittleEndian.Uint32(buf)), nil
	case "uint32":
		return binary.LittleEndian.Uint32(buf), nil
	case "int64":
		return int64(binary.LittleEndian.Uint64(buf)), nil
	case "uint64":
		return binary.LittleEndian.Uint64(buf), nil
	case "float32":
		return math.Float32frombits(binary.LittleEndian.Uint32(buf)), nil
	case "float64":
		return math.Float64frombits(binary.LittleEndian.Uint64(buf)), nil
	case "string":
		return strings.TrimRight(string(buf), "\x00"), nil
	default:
		return nil, fmt.Errorf("unsupported keyence data type %q", addr.DataType)
	}
}

func encodeWords(kind string, value any) ([]uint16, error) {
	switch kind {
	case "int16", "uint16":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		return []uint16{uint16(n)}, nil
	case "int32", "uint32":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		u := uint32(n)
		return []uint16{uint16(u), uint16(u >> 16)}, nil
	case "int64", "uint64":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		u := uint64(n)
		return []uint16{uint16(u), uint16(u >> 16), uint16(u >> 32), uint16(u >> 48)}, nil
	case "float32":
		f, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("invalid float %v", value)
		}
		u := math.Float32bits(float32(f))
		return []uint16{uint16(u), uint16(u >> 16)}, nil
	case "float64":
		f, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("invalid float %v", value)
		}
		u := math.Float64bits(f)
		return []uint16{uint16(u), uint16(u >> 16), uint16(u >> 32), uint16(u >> 48)}, nil
	case "string":
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string %v", value)
		}
		b := []byte(s)
		if len(b)%2 != 0 {
			b = append(b, 0)
		}
		out := make([]uint16, len(b)/2)
		for i := range out {
			out[i] = binary.LittleEndian.Uint16(b[i*2:])
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported keyence write data type %q", kind)
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
