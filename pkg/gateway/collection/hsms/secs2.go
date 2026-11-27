// Package hsms implements SEMI E37 (HSMS-SS), SEMI E5 (SECS-II), and SEMI E30 (GEM)
// aligned with Neuron's hsms plugin (plugins/secs_gem/).
package hsms

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// SECS-II item format codes (6-bit code in bits 7..2 of item header byte).
type ItemType uint8

const (
	ItemList    ItemType = 0x00 // L
	ItemBinary  ItemType = 0x08 // B
	ItemBoolean ItemType = 0x09 // BOOLEAN
	ItemASCII   ItemType = 0x10 // A
	ItemJIS8    ItemType = 0x11 // J
	ItemInt64   ItemType = 0x18 // I8
	ItemInt8    ItemType = 0x19 // I1
	ItemInt16   ItemType = 0x1A // I2
	ItemInt32   ItemType = 0x1C // I4
	ItemFloat64 ItemType = 0x20 // F8
	ItemFloat32 ItemType = 0x24 // F4
	ItemUint64  ItemType = 0x28 // U8
	ItemUint8   ItemType = 0x29 // U1
	ItemUint16  ItemType = 0x2A // U2
	ItemUint32  ItemType = 0x2C // U4
)

// Item represents a decoded SECS-II data item.
type Item struct {
	Type     ItemType
	Raw      []byte  // Raw big-endian payload for non-list items
	Children []*Item // Child items for ItemList
}

// Encode serializes the SECS-II item tree into wire bytes.
func (it *Item) Encode() ([]byte, error) {
	if it == nil {
		return nil, fmt.Errorf("secs2 item is nil")
	}
	if it.Type == ItemList {
		length := len(it.Children)
		hdr, err := encodeItemHeader(ItemList, length)
		if err != nil {
			return nil, err
		}
		out := make([]byte, 0, len(hdr)+length*4)
		out = append(out, hdr...)
		for _, child := range it.Children {
			b, err := child.Encode()
			if err != nil {
				return nil, err
			}
			out = append(out, b...)
		}
		return out, nil
	}
	hdr, err := encodeItemHeader(it.Type, len(it.Raw))
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(hdr)+len(it.Raw))
	out = append(out, hdr...)
	out = append(out, it.Raw...)
	return out, nil
}

func encodeItemHeader(typ ItemType, length int) ([]byte, error) {
	if length < 0 || length > 0xFFFFFF {
		return nil, fmt.Errorf("secs2 item length %d out of 24-bit range", length)
	}
	switch {
	case length <= 0xFF:
		return []byte{byte(typ)<<2 | 1, byte(length)}, nil
	case length <= 0xFFFF:
		return []byte{byte(typ)<<2 | 2, byte(length >> 8), byte(length)}, nil
	default:
		return []byte{byte(typ)<<2 | 3, byte(length >> 16), byte(length >> 8), byte(length)}, nil
	}
}

// DecodeItem decodes a single SECS-II item from data, returning the item and bytes consumed.
func DecodeItem(data []byte) (*Item, int, error) {
	if len(data) < 2 {
		return nil, 0, fmt.Errorf("secs2 item truncated (%d bytes)", len(data))
	}
	hdr := data[0]
	typ := ItemType(hdr >> 2)
	numLen := int(hdr & 0x03)
	if numLen < 1 || numLen > 3 {
		return nil, 0, fmt.Errorf("secs2 invalid length byte count %d", numLen)
	}
	if len(data) < 1+numLen {
		return nil, 0, fmt.Errorf("secs2 length header truncated")
	}
	length := 0
	for i := 0; i < numLen; i++ {
		length = (length << 8) | int(data[1+i])
	}
	pos := 1 + numLen
	if typ == ItemList {
		children := make([]*Item, 0, length)
		for i := 0; i < length; i++ {
			child, n, err := DecodeItem(data[pos:])
			if err != nil {
				return nil, 0, err
			}
			children = append(children, child)
			pos += n
		}
		return &Item{Type: ItemList, Children: children}, pos, nil
	}
	if pos+length > len(data) {
		return nil, 0, fmt.Errorf("secs2 payload truncated: need %d have %d", length, len(data)-pos)
	}
	raw := make([]byte, length)
	copy(raw, data[pos:pos+length])
	return &Item{Type: typ, Raw: raw}, pos + length, nil
}

// FormatSML renders a SECS-II item in Neuron-compatible SML layout text.
func (it *Item) FormatSML() string {
	var sb strings.Builder
	it.formatSML(&sb, true)
	return strings.TrimRight(sb.String(), " ")
}

func (it *Item) formatSML(sb *strings.Builder, root bool) {
	if it == nil {
		return
	}
	switch it.Type {
	case ItemList:
		sb.WriteString(fmt.Sprintf("<L[%d] ", len(it.Children)))
		for _, ch := range it.Children {
			ch.formatSML(sb, false)
		}
		if root {
			sb.WriteString(">. ")
		} else {
			sb.WriteString("> ")
		}
	case ItemBinary:
		sb.WriteString(fmt.Sprintf("<B[%d] %s> ", len(it.Raw), strings.ToUpper(hex.EncodeToString(it.Raw))))
	case ItemBoolean:
		v := 0
		if len(it.Raw) > 0 && it.Raw[0] != 0 {
			v = 1
		}
		sb.WriteString(fmt.Sprintf("<Boolean %d> ", v))
	case ItemASCII, ItemJIS8:
		sb.WriteString(fmt.Sprintf("<A[%d] %s> ", len(it.Raw), string(it.Raw)))
	case ItemInt8:
		if len(it.Raw) >= 1 {
			sb.WriteString(fmt.Sprintf("<I1 %d> ", int8(it.Raw[0])))
		}
	case ItemInt16:
		if len(it.Raw) >= 2 {
			sb.WriteString(fmt.Sprintf("<I2 %d> ", int16(binary.BigEndian.Uint16(it.Raw))))
		}
	case ItemInt32:
		if len(it.Raw) >= 4 {
			sb.WriteString(fmt.Sprintf("<I4 %d> ", int32(binary.BigEndian.Uint32(it.Raw))))
		}
	case ItemInt64:
		if len(it.Raw) >= 8 {
			sb.WriteString(fmt.Sprintf("<I8 %d> ", int64(binary.BigEndian.Uint64(it.Raw))))
		}
	case ItemUint8:
		if len(it.Raw) >= 1 {
			sb.WriteString(fmt.Sprintf("<U1 %d> ", it.Raw[0]))
		}
	case ItemUint16:
		if len(it.Raw) >= 2 {
			sb.WriteString(fmt.Sprintf("<U2 %d> ", binary.BigEndian.Uint16(it.Raw)))
		}
	case ItemUint32:
		if len(it.Raw) >= 4 {
			sb.WriteString(fmt.Sprintf("<U4 %d> ", binary.BigEndian.Uint32(it.Raw)))
		}
	case ItemUint64:
		if len(it.Raw) >= 8 {
			sb.WriteString(fmt.Sprintf("<U8 %d> ", binary.BigEndian.Uint64(it.Raw)))
		}
	case ItemFloat32:
		if len(it.Raw) >= 4 {
			sb.WriteString(fmt.Sprintf("<F4 %g> ", math.Float32frombits(binary.BigEndian.Uint32(it.Raw))))
		}
	case ItemFloat64:
		if len(it.Raw) >= 8 {
			sb.WriteString(fmt.Sprintf("<F8 %g> ", math.Float64frombits(binary.BigEndian.Uint64(it.Raw))))
		}
	}
}

// ParseSML parses Neuron-style SML text (e.g. "<L[2] <A[4] TEST> <U2 12> >.") into an Item.
func ParseSML(text string) (*Item, error) {
	p := &smlParser{s: text}
	it, err := p.parseItem()
	if err != nil {
		return nil, err
	}
	return it, nil
}

type smlParser struct {
	s   string
	pos int
}

func (p *smlParser) skipWS() {
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			p.pos++
		} else {
			break
		}
	}
}

func (p *smlParser) parseItem() (*Item, error) {
	p.skipWS()
	if p.pos >= len(p.s) || p.s[p.pos] != '<' {
		return nil, fmt.Errorf("expected '<' at pos %d", p.pos)
	}
	p.pos++
	tagStart := p.pos
	for p.pos < len(p.s) && p.s[p.pos] != '[' && p.s[p.pos] != ' ' && p.s[p.pos] != '\t' && p.s[p.pos] != '\r' && p.s[p.pos] != '\n' && p.s[p.pos] != '>' {
		p.pos++
	}
	tag := p.s[tagStart:p.pos]
	hasLen := false
	declaredLen := 0
	if p.pos < len(p.s) && p.s[p.pos] == '[' {
		p.pos++
		end := strings.IndexByte(p.s[p.pos:], ']')
		if end < 0 {
			return nil, fmt.Errorf("unclosed '[' in SML")
		}
		n, err := strconv.Atoi(strings.TrimSpace(p.s[p.pos : p.pos+end]))
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid SML length %q", p.s[p.pos:p.pos+end])
		}
		declaredLen = n
		hasLen = true
		p.pos += end + 1
	}
	p.skipWS()
	switch strings.ToUpper(tag) {
	case "L":
		children := make([]*Item, 0, declaredLen)
		for {
			p.skipWS()
			if p.pos >= len(p.s) {
				return nil, fmt.Errorf("unclosed SML list")
			}
			if p.s[p.pos] == '>' {
				p.pos++
				if p.pos < len(p.s) && p.s[p.pos] == '.' {
					p.pos++
				}
				break
			}
			ch, err := p.parseItem()
			if err != nil {
				return nil, err
			}
			children = append(children, ch)
		}
		return &Item{Type: ItemList, Children: children}, nil
	case "A", "JIS-8":
		typ := ItemASCII
		if strings.ToUpper(tag) == "JIS-8" {
			typ = ItemJIS8
		}
		if hasLen {
			if p.pos+declaredLen > len(p.s) {
				return nil, fmt.Errorf("SML ASCII length %d exceeds input", declaredLen)
			}
			raw := []byte(p.s[p.pos : p.pos+declaredLen])
			p.pos += declaredLen
			p.skipWS()
			if p.pos >= len(p.s) || p.s[p.pos] != '>' {
				return nil, fmt.Errorf("expected '>' after SML ASCII")
			}
			p.pos++
			return &Item{Type: typ, Raw: raw}, nil
		}
		val, err := p.readUntilGT()
		if err != nil {
			return nil, err
		}
		val = strings.Trim(val, "\"'")
		return &Item{Type: typ, Raw: []byte(val)}, nil
	case "B":
		val, err := p.readUntilGT()
		if err != nil {
			return nil, err
		}
		cleaned := strings.ReplaceAll(strings.TrimSpace(val), " ", "")
		cleaned = strings.TrimPrefix(strings.TrimPrefix(cleaned, "0x"), "0X")
		if cleaned == "" {
			return &Item{Type: ItemBinary, Raw: nil}, nil
		}
		raw, err := hex.DecodeString(cleaned)
		if err != nil {
			return nil, fmt.Errorf("invalid SML binary hex %q: %w", val, err)
		}
		return &Item{Type: ItemBinary, Raw: raw}, nil
	case "BOOLEAN", "BOOL":
		val, err := p.readUntilGT()
		if err != nil {
			return nil, err
		}
		b, err := toBool(val)
		if err != nil {
			return nil, err
		}
		v := byte(0)
		if b {
			v = 1
		}
		return &Item{Type: ItemBoolean, Raw: []byte{v}}, nil
	case "I1":
		return p.parseIntScalar(ItemInt8, 1, true)
	case "I2":
		return p.parseIntScalar(ItemInt16, 2, true)
	case "I4":
		return p.parseIntScalar(ItemInt32, 4, true)
	case "I8":
		return p.parseIntScalar(ItemInt64, 8, true)
	case "U1":
		return p.parseIntScalar(ItemUint8, 1, false)
	case "U2":
		return p.parseIntScalar(ItemUint16, 2, false)
	case "U4":
		return p.parseIntScalar(ItemUint32, 4, false)
	case "U8":
		return p.parseIntScalar(ItemUint64, 8, false)
	case "F4":
		val, err := p.readUntilGT()
		if err != nil {
			return nil, err
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 32)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, math.Float32bits(float32(f)))
		return &Item{Type: ItemFloat32, Raw: buf}, nil
	case "F8":
		val, err := p.readUntilGT()
		if err != nil {
			return nil, err
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, math.Float64bits(f))
		return &Item{Type: ItemFloat64, Raw: buf}, nil
	default:
		return nil, fmt.Errorf("unsupported SML tag %q", tag)
	}
}

func (p *smlParser) readUntilGT() (string, error) {
	end := strings.IndexByte(p.s[p.pos:], '>')
	if end < 0 {
		return "", fmt.Errorf("missing '>' in SML")
	}
	val := p.s[p.pos : p.pos+end]
	p.pos += end + 1
	return val, nil
}

func (p *smlParser) parseIntScalar(typ ItemType, size int, signed bool) (*Item, error) {
	val, err := p.readUntilGT()
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(val)
	buf := make([]byte, size)
	if signed {
		n, err := strconv.ParseInt(s, 10, size*8)
		if err != nil {
			return nil, err
		}
		switch size {
		case 1:
			buf[0] = byte(int8(n))
		case 2:
			binary.BigEndian.PutUint16(buf, uint16(int16(n)))
		case 4:
			binary.BigEndian.PutUint32(buf, uint32(int32(n)))
		case 8:
			binary.BigEndian.PutUint64(buf, uint64(n))
		}
	} else {
		u, err := strconv.ParseUint(s, 10, size*8)
		if err != nil {
			return nil, err
		}
		switch size {
		case 1:
			buf[0] = byte(u)
		case 2:
			binary.BigEndian.PutUint16(buf, uint16(u))
		case 4:
			binary.BigEndian.PutUint32(buf, uint32(u))
		case 8:
			binary.BigEndian.PutUint64(buf, u)
		}
	}
	return &Item{Type: typ, Raw: buf}, nil
}

// DecodeItemValue converts a SECS-II item into a Go value matching the target DataType.
func DecodeItemValue(it *Item, dataType string) (any, error) {
	if it == nil {
		return nil, fmt.Errorf("secs2 item is nil")
	}
	dt := strings.ToUpper(strings.TrimSpace(dataType))
	if it.Type == ItemList {
		if dt == "STRING" {
			return it.FormatSML(), nil
		}
		return nil, fmt.Errorf("cannot decode SECS-II List into %s", dt)
	}
	switch dt {
	case "STRING":
		if it.Type == ItemASCII || it.Type == ItemJIS8 {
			return string(it.Raw), nil
		}
		if it.Type == ItemBinary {
			return strings.ToUpper(hex.EncodeToString(it.Raw)), nil
		}
		return it.FormatSML(), nil
	case "BOOL", "BIT":
		v, err := itemToBool(it)
		if err != nil {
			return nil, err
		}
		if dt == "BIT" {
			if v {
				return 1, nil
			}
			return 0, nil
		}
		return v, nil
	case "INT8":
		n, err := itemToInt64(it)
		return int8(n), err
	case "INT16":
		n, err := itemToInt64(it)
		return int16(n), err
	case "INT32":
		n, err := itemToInt64(it)
		return int32(n), err
	case "INT64":
		return itemToInt64(it)
	case "UINT8":
		u, err := itemToUint64(it)
		return uint8(u), err
	case "UINT16":
		u, err := itemToUint64(it)
		return uint16(u), err
	case "UINT32":
		u, err := itemToUint64(it)
		return uint32(u), err
	case "UINT64":
		return itemToUint64(it)
	case "FLOAT32", "FLOAT":
		f, err := itemToFloat64(it)
		return float32(f), err
	case "FLOAT64", "DOUBLE":
		return itemToFloat64(it)
	default:
		return nil, fmt.Errorf("unsupported HSMS data type %q", dataType)
	}
}

func itemToBool(it *Item) (bool, error) {
	switch it.Type {
	case ItemBoolean, ItemBinary, ItemUint8, ItemInt8:
		if len(it.Raw) < 1 {
			return false, fmt.Errorf("empty boolean item")
		}
		return it.Raw[0] != 0, nil
	default:
		n, err := itemToInt64(it)
		return n != 0, err
	}
}

func itemToInt64(it *Item) (int64, error) {
	switch it.Type {
	case ItemInt8:
		if len(it.Raw) < 1 {
			return 0, fmt.Errorf("short I1")
		}
		return int64(int8(it.Raw[0])), nil
	case ItemInt16:
		if len(it.Raw) < 2 {
			return 0, fmt.Errorf("short I2")
		}
		return int64(int16(binary.BigEndian.Uint16(it.Raw))), nil
	case ItemInt32:
		if len(it.Raw) < 4 {
			return 0, fmt.Errorf("short I4")
		}
		return int64(int32(binary.BigEndian.Uint32(it.Raw))), nil
	case ItemInt64:
		if len(it.Raw) < 8 {
			return 0, fmt.Errorf("short I8")
		}
		return int64(binary.BigEndian.Uint64(it.Raw)), nil
	case ItemUint8, ItemBinary, ItemBoolean:
		if len(it.Raw) < 1 {
			return 0, fmt.Errorf("short U1")
		}
		return int64(it.Raw[0]), nil
	case ItemUint16:
		if len(it.Raw) < 2 {
			return 0, fmt.Errorf("short U2")
		}
		return int64(binary.BigEndian.Uint16(it.Raw)), nil
	case ItemUint32:
		if len(it.Raw) < 4 {
			return 0, fmt.Errorf("short U4")
		}
		return int64(binary.BigEndian.Uint32(it.Raw)), nil
	case ItemUint64:
		if len(it.Raw) < 8 {
			return 0, fmt.Errorf("short U8")
		}
		return int64(binary.BigEndian.Uint64(it.Raw)), nil
	case ItemFloat32:
		if len(it.Raw) < 4 {
			return 0, fmt.Errorf("short F4")
		}
		return int64(math.Float32frombits(binary.BigEndian.Uint32(it.Raw))), nil
	case ItemFloat64:
		if len(it.Raw) < 8 {
			return 0, fmt.Errorf("short F8")
		}
		return int64(math.Float64frombits(binary.BigEndian.Uint64(it.Raw))), nil
	case ItemASCII:
		return strconv.ParseInt(strings.TrimSpace(string(it.Raw)), 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert SECS-II type 0x%02X to int", it.Type)
	}
}

func itemToUint64(it *Item) (uint64, error) {
	switch it.Type {
	case ItemUint8, ItemBinary, ItemBoolean:
		if len(it.Raw) < 1 {
			return 0, fmt.Errorf("short U1")
		}
		return uint64(it.Raw[0]), nil
	case ItemUint16:
		if len(it.Raw) < 2 {
			return 0, fmt.Errorf("short U2")
		}
		return uint64(binary.BigEndian.Uint16(it.Raw)), nil
	case ItemUint32:
		if len(it.Raw) < 4 {
			return 0, fmt.Errorf("short U4")
		}
		return uint64(binary.BigEndian.Uint32(it.Raw)), nil
	case ItemUint64:
		if len(it.Raw) < 8 {
			return 0, fmt.Errorf("short U8")
		}
		return binary.BigEndian.Uint64(it.Raw), nil
	default:
		n, err := itemToInt64(it)
		return uint64(n), err
	}
}

func itemToFloat64(it *Item) (float64, error) {
	switch it.Type {
	case ItemFloat32:
		if len(it.Raw) < 4 {
			return 0, fmt.Errorf("short F4")
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(it.Raw))), nil
	case ItemFloat64:
		if len(it.Raw) < 8 {
			return 0, fmt.Errorf("short F8")
		}
		return math.Float64frombits(binary.BigEndian.Uint64(it.Raw)), nil
	case ItemASCII:
		return strconv.ParseFloat(strings.TrimSpace(string(it.Raw)), 64)
	default:
		n, err := itemToInt64(it)
		return float64(n), err
	}
}

// EncodeGoValue builds a SECS-II item for writing a Go value with the given DataType.
// If dataType is STRING and value starts with "<L", it parses Neuron SML syntax.
func EncodeGoValue(dataType string, value any) (*Item, error) {
	dt := strings.ToUpper(strings.TrimSpace(dataType))
	switch dt {
	case "STRING":
		s := fmt.Sprint(value)
		trimmed := strings.TrimSpace(s)
		if strings.HasPrefix(trimmed, "<L") || strings.HasPrefix(trimmed, "<l") {
			return ParseSML(trimmed)
		}
		return &Item{Type: ItemASCII, Raw: []byte(s)}, nil
	case "BOOL", "BIT":
		b, err := toBool(value)
		if err != nil {
			return nil, err
		}
		v := byte(0)
		if b {
			v = 1
		}
		return &Item{Type: ItemBoolean, Raw: []byte{v}}, nil
	case "INT8":
		n, err := toInt64(value)
		return &Item{Type: ItemInt8, Raw: []byte{byte(int8(n))}}, err
	case "INT16":
		n, err := toInt64(value)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(int16(n)))
		return &Item{Type: ItemInt16, Raw: buf}, err
	case "INT32":
		n, err := toInt64(value)
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(int32(n)))
		return &Item{Type: ItemInt32, Raw: buf}, err
	case "INT64":
		n, err := toInt64(value)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(n))
		return &Item{Type: ItemInt64, Raw: buf}, err
	case "UINT8":
		u, err := toUint64(value)
		return &Item{Type: ItemUint8, Raw: []byte{byte(u)}}, err
	case "UINT16":
		u, err := toUint64(value)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(u))
		return &Item{Type: ItemUint16, Raw: buf}, err
	case "UINT32":
		u, err := toUint64(value)
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(u))
		return &Item{Type: ItemUint32, Raw: buf}, err
	case "UINT64":
		u, err := toUint64(value)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, u)
		return &Item{Type: ItemUint64, Raw: buf}, err
	case "FLOAT32", "FLOAT":
		f, err := toFloat64(value)
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, math.Float32bits(float32(f)))
		return &Item{Type: ItemFloat32, Raw: buf}, err
	case "FLOAT64", "DOUBLE":
		f, err := toFloat64(value)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, math.Float64bits(f))
		return &Item{Type: ItemFloat64, Raw: buf}, err
	default:
		return nil, fmt.Errorf("unsupported HSMS write data type %q", dataType)
	}
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case int8:
		return v != 0, nil
	case int16:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case uint:
		return v != 0, nil
	case uint8:
		return v != 0, nil
	case uint16:
		return v != 0, nil
	case uint32:
		return v != 0, nil
	case uint64:
		return v != 0, nil
	case float32:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		if s == "true" || s == "1" {
			return true, nil
		}
		if s == "false" || s == "0" {
			return false, nil
		}
	}
	return false, fmt.Errorf("invalid boolean value %v", value)
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	}
	return 0, fmt.Errorf("invalid integer value %v", value)
}

func toUint64(value any) (uint64, error) {
	switch v := value.(type) {
	case int:
		return uint64(v), nil
	case int8:
		return uint64(v), nil
	case int16:
		return uint64(v), nil
	case int32:
		return uint64(v), nil
	case int64:
		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case float32:
		return uint64(v), nil
	case float64:
		return uint64(v), nil
	case string:
		return strconv.ParseUint(strings.TrimSpace(v), 10, 64)
	}
	return 0, fmt.Errorf("invalid unsigned value %v", value)
}

func toFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	}
	return 0, fmt.Errorf("invalid float value %v", value)
}
