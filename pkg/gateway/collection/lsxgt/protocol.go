package lsxgt

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	fenetMagic        = "LSIS-XGT\x00\x00"
	fenetHeaderSize   = 20
	fenetCmdReadReq   = 0x0054
	fenetCmdReadResp  = 0x0055
	fenetCmdWriteReq  = 0x0058
	fenetCmdWriteResp = 0x0059
	fenetTypeBit      = 0x0000
	fenetTypeCont     = 0x0014

	maxFEnetBatchBytes = 200
	maxCnetBatchBytes  = 60
)

type pointAddress struct {
	Area       byte
	SizeCode   byte // 'X', 'B', 'W', 'D', 'L'
	ByteOffset int
	BitIndex   int // 0..7 within ByteOffset
	BitOffset  int // absolute bit index for %<area>X<bit>
	ByteCount  int
	IsBit      bool
	DataType   string
}

func isValidArea(area byte) bool {
	switch area {
	case 'P', 'M', 'L', 'K', 'F', 'T', 'C', 'D', 'U', 'Z', 'N', 'R':
		return true
	default:
		return false
	}
}

// parseAddress parses canonical XGT direct variables (%MW100, %MX16, %DB20, %MD10)
// as well as shorthand forms (MW100, D100, M10).
func parseAddress(raw, dataType string) (*pointAddress, error) {
	text := strings.ToUpper(strings.TrimSpace(raw))
	text = strings.TrimPrefix(text, "%")
	if len(text) < 2 {
		return nil, fmt.Errorf("invalid XGT address %q", raw)
	}
	area := text[0]
	if !isValidArea(area) {
		return nil, fmt.Errorf("unsupported XGT area in %q", raw)
	}
	rest := text[1:]
	sizeCode := byte('W')
	switch rest[0] {
	case 'X', 'B', 'W', 'D', 'L':
		sizeCode = rest[0]
		rest = rest[1:]
	default:
		if strings.EqualFold(strings.TrimSpace(dataType), "bool") || strings.EqualFold(strings.TrimSpace(dataType), "bit") {
			sizeCode = 'X'
		}
	}
	offsetPart := rest
	dotBit := -1
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		offsetPart = rest[:dot]
		b, err := strconv.Atoi(rest[dot+1:])
		if err != nil || b < 0 || b > 15 {
			return nil, fmt.Errorf("invalid XGT bit index in %q", raw)
		}
		dotBit = b
	}
	num, err := strconv.Atoi(offsetPart)
	if err != nil || num < 0 || num > 999999 {
		return nil, fmt.Errorf("invalid XGT offset in %q", raw)
	}
	kind := strings.ToLower(strings.TrimSpace(dataType))
	addr := &pointAddress{
		Area:     area,
		SizeCode: sizeCode,
		DataType: kind,
	}
	switch sizeCode {
	case 'X':
		addr.BitOffset = num
		addr.ByteOffset = num / 8
		addr.BitIndex = num % 8
	case 'B':
		addr.ByteOffset = num
		if dotBit >= 0 {
			addr.BitIndex = dotBit % 8
			addr.BitOffset = num*8 + dotBit
		}
	case 'W':
		addr.ByteOffset = num * 2
		if dotBit >= 0 {
			addr.ByteOffset += dotBit / 8
			addr.BitIndex = dotBit % 8
			addr.BitOffset = num*16 + dotBit
		}
	case 'D':
		addr.ByteOffset = num * 4
		if dotBit >= 0 {
			addr.ByteOffset += dotBit / 8
			addr.BitIndex = dotBit % 8
			addr.BitOffset = num*32 + dotBit
		}
	case 'L':
		addr.ByteOffset = num * 8
	}
	switch kind {
	case "bool", "bit":
		addr.IsBit = true
		addr.ByteCount = 1
		if sizeCode != 'X' && dotBit < 0 {
			addr.BitOffset = addr.ByteOffset * 8
		}
	case "int8", "uint8":
		addr.ByteCount = 1
	case "int16", "uint16":
		addr.ByteCount = 2
	case "int32", "uint32", "float32":
		addr.ByteCount = 4
	case "int64", "uint64", "float64":
		addr.ByteCount = 8
	case "string":
		addr.ByteCount = 16
	default:
		return nil, fmt.Errorf("unsupported XGT data type %q", dataType)
	}
	return addr, nil
}

func (a *pointAddress) byteVarName() string {
	return fmt.Sprintf("%%%cB%d", a.Area, a.ByteOffset)
}

func (a *pointAddress) bitVarName() string {
	return fmt.Sprintf("%%%cX%d", a.Area, a.BitOffset)
}

type readBatch struct {
	area       byte
	byteOffset int
	byteCount  int
	items      []*batchItem
}

type batchItem struct {
	variable *collection.Variable
	addr     *pointAddress
	offset   int
}

func buildReadBatches(variables []*collection.Variable, maxBytes int) ([]*readBatch, error) {
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
		return items[i].addr.ByteOffset < items[j].addr.ByteOffset
	})
	var batches []*readBatch
	for _, item := range items {
		a := item.addr
		if len(batches) > 0 {
			last := batches[len(batches)-1]
			lastEnd := last.byteOffset + last.byteCount
			newEnd := a.ByteOffset + a.ByteCount
			if last.area == a.Area && a.ByteOffset <= lastEnd && newEnd-last.byteOffset <= maxBytes {
				item.offset = a.ByteOffset - last.byteOffset
				if newEnd > lastEnd {
					last.byteCount = newEnd - last.byteOffset
				}
				last.items = append(last.items, item)
				continue
			}
		}
		item.offset = 0
		batches = append(batches, &readBatch{
			area:       a.Area,
			byteOffset: a.ByteOffset,
			byteCount:  a.ByteCount,
			items:      []*batchItem{item},
		})
	}
	return batches, nil
}

// --- FEnet Binary Protocol (Official LS XGT FEnet I/F Manual) ---

func buildFEnetHeader(invokeID uint16, bodyLen int) []byte {
	hdr := make([]byte, fenetHeaderSize)
	copy(hdr[0:10], fenetMagic)
	hdr[12] = 0xA0 // CPU info
	hdr[13] = 0x33 // Client -> PLC
	binary.LittleEndian.PutUint16(hdr[14:16], invokeID)
	binary.LittleEndian.PutUint16(hdr[16:18], uint16(bodyLen))
	var bcc byte
	for i := 0; i < 19; i++ {
		bcc += hdr[i]
	}
	hdr[19] = bcc
	return hdr
}

func buildFEnetReadRequest(invokeID uint16, area byte, byteOffset, byteCount int) []byte {
	varName := fmt.Sprintf("%%%cB%d", area, byteOffset)
	body := make([]byte, 0, 12+len(varName))
	body = binary.LittleEndian.AppendUint16(body, fenetCmdReadReq)
	body = binary.LittleEndian.AppendUint16(body, fenetTypeCont)
	body = binary.LittleEndian.AppendUint16(body, 0) // reserved
	body = binary.LittleEndian.AppendUint16(body, 1) // block count
	body = binary.LittleEndian.AppendUint16(body, uint16(len(varName)))
	body = append(body, varName...)
	body = binary.LittleEndian.AppendUint16(body, uint16(byteCount))
	return append(buildFEnetHeader(invokeID, len(body)), body...)
}

func buildFEnetWriteRequest(invokeID uint16, addr *pointAddress, data []byte) []byte {
	varName := addr.byteVarName()
	dataType := uint16(fenetTypeCont)
	if addr.IsBit {
		varName = addr.bitVarName()
		dataType = fenetTypeBit
	}
	body := make([]byte, 0, 12+len(varName)+len(data))
	body = binary.LittleEndian.AppendUint16(body, fenetCmdWriteReq)
	body = binary.LittleEndian.AppendUint16(body, dataType)
	body = binary.LittleEndian.AppendUint16(body, 0) // reserved
	body = binary.LittleEndian.AppendUint16(body, 1) // block count
	body = binary.LittleEndian.AppendUint16(body, uint16(len(varName)))
	body = append(body, varName...)
	body = binary.LittleEndian.AppendUint16(body, uint16(len(data)))
	body = append(body, data...)
	return append(buildFEnetHeader(invokeID, len(body)), body...)
}

func parseFEnetReadResponse(frame []byte) ([]byte, error) {
	if len(frame) < fenetHeaderSize+12 {
		return nil, fmt.Errorf("FEnet read response is too short (%d bytes)", len(frame))
	}
	if string(frame[0:8]) != "LSIS-XGT" {
		return nil, fmt.Errorf("invalid FEnet header magic")
	}
	body := frame[fenetHeaderSize:]
	cmd := binary.LittleEndian.Uint16(body[0:2])
	if cmd != fenetCmdReadResp {
		return nil, fmt.Errorf("unexpected FEnet read response cmd 0x%04X", cmd)
	}
	errStatus := binary.LittleEndian.Uint16(body[6:8])
	if errStatus != 0 {
		return nil, fmt.Errorf("FEnet PLC error status 0x%04X", errStatus)
	}
	dataLen := int(binary.LittleEndian.Uint16(body[10:12]))
	if len(body) < 12+dataLen {
		return nil, fmt.Errorf("FEnet read payload truncated: got %d, want %d", len(body)-12, dataLen)
	}
	return body[12 : 12+dataLen], nil
}

func parseFEnetWriteResponse(frame []byte) error {
	if len(frame) < fenetHeaderSize+8 {
		return fmt.Errorf("FEnet write response is too short (%d bytes)", len(frame))
	}
	if string(frame[0:8]) != "LSIS-XGT" {
		return fmt.Errorf("invalid FEnet header magic")
	}
	body := frame[fenetHeaderSize:]
	cmd := binary.LittleEndian.Uint16(body[0:2])
	if cmd != fenetCmdWriteResp {
		return fmt.Errorf("unexpected FEnet write response cmd 0x%04X", cmd)
	}
	errStatus := binary.LittleEndian.Uint16(body[6:8])
	if errStatus != 0 {
		return fmt.Errorf("FEnet PLC write error 0x%04X", errStatus)
	}
	return nil
}

// --- Cnet ASCII Protocol (Official LS XGT Cnet I/F Manual) ---

func cnetBCC(frame []byte) byte {
	var sum byte
	for _, b := range frame {
		sum += b
	}
	return sum
}

func buildCnetReadRequest(station int, area byte, byteOffset, byteCount int) []byte {
	varName := fmt.Sprintf("%%%cB%d", area, byteOffset)
	// Per LS XGT Cnet I/F Manual, uppercase commands (RSB/WSB/WSS) terminate at EOT (0x04) without trailing BCC.
	return []byte(fmt.Sprintf("\x05%02XRSB%02X%s%02X\x04", station, len(varName), varName, byteCount))
}

func buildCnetWriteRequest(station int, addr *pointAddress, data []byte) []byte {
	if addr.IsBit {
		varName := addr.bitVarName()
		valHex := "00"
		if len(data) > 0 && data[0] != 0 {
			valHex = "01"
		}
		return []byte(fmt.Sprintf("\x05%02XWSS01%02X%s%s\x04", station, len(varName), varName, valHex))
	}
	varName := addr.byteVarName()
	return []byte(fmt.Sprintf("\x05%02XWSB%02X%s%02X%s\x04", station, len(varName), varName, len(data), strings.ToUpper(hex.EncodeToString(data))))
}

func parseCnetReadResponse(frame []byte) ([]byte, error) {
	if len(frame) < 11 {
		return nil, fmt.Errorf("Cnet read response is too short")
	}
	if frame[0] == 0x15 {
		return nil, fmt.Errorf("Cnet PLC NAK response %q", string(frame[1:]))
	}
	if frame[0] != 0x06 {
		return nil, fmt.Errorf("invalid Cnet response start 0x%02X", frame[0])
	}
	etx := strings.IndexByte(string(frame), 0x03)
	if etx < 10 {
		return nil, fmt.Errorf("Cnet response missing ETX")
	}
	// Format: ACK(1) + station(2) + RSB(3) + blockCount(2) + byteCount(2) + hexData + ETX
	dataLen, err := strconv.ParseUint(string(frame[8:10]), 16, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid Cnet byte count %q", string(frame[8:10]))
	}
	hexPayload := string(frame[10:etx])
	if len(hexPayload) < int(dataLen)*2 {
		return nil, fmt.Errorf("Cnet payload truncated: got %d hex chars, want %d", len(hexPayload), dataLen*2)
	}
	return hex.DecodeString(hexPayload[:int(dataLen)*2])
}

func parseCnetWriteResponse(frame []byte) error {
	if len(frame) < 7 {
		return fmt.Errorf("Cnet write response is too short")
	}
	if frame[0] == 0x15 {
		return fmt.Errorf("Cnet PLC write NAK %q", string(frame[1:]))
	}
	if frame[0] != 0x06 {
		return fmt.Errorf("invalid Cnet write response start 0x%02X", frame[0])
	}
	return nil
}

// --- Shared Little-Endian Value Codec ---

func decodeValue(addr *pointAddress, data []byte) (any, error) {
	if len(data) < addr.ByteCount {
		return nil, fmt.Errorf("XGT data too short: %d < %d", len(data), addr.ByteCount)
	}
	if addr.IsBit {
		return (data[0] & (1 << uint(addr.BitIndex))) != 0, nil
	}
	switch addr.DataType {
	case "int8":
		return int8(data[0]), nil
	case "uint8":
		return data[0], nil
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
		return strings.TrimRight(string(data[:addr.ByteCount]), "\x00"), nil
	default:
		return nil, fmt.Errorf("unsupported XGT data type %q", addr.DataType)
	}
}

func encodeValue(addr *pointAddress, value any) ([]byte, error) {
	if addr.IsBit {
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
	case "int8", "uint8":
		n, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid integer %v", value)
		}
		return []byte{byte(n)}, nil
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
		buf := make([]byte, addr.ByteCount)
		copy(buf, []byte(s))
		return buf, nil
	default:
		return nil, fmt.Errorf("unsupported XGT write data type %q", addr.DataType)
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
