// Package cclinkie implements Mitsubishi / CLPA CC-Link IE Field Basic
// cyclic communication (SLMP 0x0E70 over UDP port 61450).
package cclinkie

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	slmpSubheaderReq  uint16 = 0x5000
	slmpSubheaderResp uint16 = 0xD000
	slmpCmdCyclicData uint16 = 0x0E70
	protocolVersion   uint16 = 0x0001

	bitsPerStation  = 64
	bytesBitArea    = 8  // 64 bits = 8 bytes per occupied station
	wordsPerStation = 32 // 32 words = 64 bytes per occupied station
	bytesWordArea   = 64
	bytesPerStation = bytesBitArea + bytesWordArea // 72 bytes per occupied station
)

type areaKind int

const (
	areaRX  areaKind = iota // Slave -> Master bits (read-only)
	areaRY                  // Master -> Slave bits (read/write)
	areaRWr                 // Slave -> Master words (read-only)
	areaRWw                 // Master -> Slave words (read/write)
)

type point struct {
	variable *collection.Variable
	area     areaKind
	index    int
	wordBit  int // >=0 for RWr0.3 / RWw0.3
	words    int
	dataType string
}

func parseAddress(variable *collection.Variable) (*point, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	raw := strings.ToUpper(strings.TrimSpace(variable.Address))
	var area areaKind
	var rest string
	switch {
	case strings.HasPrefix(raw, "RWR"):
		area, rest = areaRWr, raw[3:]
	case strings.HasPrefix(raw, "RWW"):
		area, rest = areaRWw, raw[3:]
	case strings.HasPrefix(raw, "RX"):
		area, rest = areaRX, raw[2:]
	case strings.HasPrefix(raw, "RY"):
		area, rest = areaRY, raw[2:]
	default:
		return nil, fmt.Errorf("CC-Link IE address %q must start with RX/RY/RWr/RWw", variable.Address)
	}
	numText := rest
	wordBit := -1
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		numText = rest[:dot]
		b, err := strconv.Atoi(rest[dot+1:])
		if err != nil || b < 0 || b > 15 {
			return nil, fmt.Errorf("CC-Link IE bit suffix %q must be 0..15", rest[dot+1:])
		}
		wordBit = b
	}
	base := 10
	if strings.HasPrefix(numText, "0X") {
		numText = numText[2:]
		base = 16
	}
	idx, err := strconv.ParseUint(numText, base, 16)
	if err != nil {
		return nil, fmt.Errorf("CC-Link IE index %q is invalid", numText)
	}
	dt := strings.ToUpper(strings.TrimSpace(variable.DataType))
	p := &point{
		variable: variable,
		area:     area,
		index:    int(idx),
		wordBit:  wordBit,
		dataType: dt,
	}
	if area == areaRX || area == areaRY {
		if wordBit >= 0 {
			return nil, fmt.Errorf("CC-Link IE bit device %q must not have bit suffix", variable.Address)
		}
		if dt != "BIT" && dt != "BOOL" {
			return nil, fmt.Errorf("CC-Link IE bit device %q only supports BIT/BOOL", variable.Address)
		}
		p.words = 1
		return p, nil
	}
	if wordBit >= 0 {
		if dt != "BIT" && dt != "BOOL" {
			return nil, fmt.Errorf("CC-Link IE word bit %q requires BIT/BOOL", variable.Address)
		}
		p.words = 1
		return p, nil
	}
	words, err := wordsForType(dt)
	if err != nil {
		return nil, err
	}
	p.words = words
	return p, nil
}

func wordsForType(dt string) (int, error) {
	switch dt {
	case "INT16", "UINT16":
		return 1, nil
	case "INT32", "UINT32", "FLOAT32", "FLOAT":
		return 2, nil
	case "INT64", "UINT64", "FLOAT64", "DOUBLE":
		return 4, nil
	default:
		return 0, fmt.Errorf("CC-Link IE data type %q is not supported", dt)
	}
}

func ipv4ToUint32(ipStr string) uint32 {
	ip := net.ParseIP(strings.TrimSpace(ipStr)).To4()
	if ip == nil {
		return 0x0100007F // 127.0.0.1
	}
	return binary.LittleEndian.Uint32(ip)
}

// buildCyclicRequest builds a CLPA CC-Link IE Field Basic 0x0E70 CyclicData request.
func buildCyclicRequest(masterID, slaveID uint32, groupNo byte, seq uint16, occupied int, rww, ry []byte) []byte {
	if occupied < 1 {
		occupied = 1
	}
	rwwSize := occupied * bytesWordArea
	rySize := occupied * bytesBitArea
	reqDataLen := 2 + 2 + 16 + 12 + 20 + occupied*4 + rwwSize + rySize
	bodyLen := 6 + reqDataLen // timer(2) + cmd(2) + subcmd(2) + reqData

	pkt := make([]byte, 9+bodyLen)
	binary.BigEndian.PutUint16(pkt[0:2], slmpSubheaderReq)
	pkt[2] = 0x00 // Network No
	pkt[3] = 0xFF // Station No
	pkt[4] = 0xFF // Module I/O No (0x03FF LE)
	pkt[5] = 0x03
	pkt[6] = 0x00 // Multidrop
	binary.LittleEndian.PutUint16(pkt[7:9], uint16(bodyLen))
	binary.LittleEndian.PutUint16(pkt[9:11], 0x0000) // Monitoring timer / reserved
	binary.LittleEndian.PutUint16(pkt[11:13], slmpCmdCyclicData)
	binary.LittleEndian.PutUint16(pkt[13:15], 0x0000) // Subcommand

	off := 15
	binary.LittleEndian.PutUint16(pkt[off:off+2], protocolVersion)
	off += 4                                              // protocolVer(2) + reserved(2)
	binary.LittleEndian.PutUint16(pkt[off:off+2], 0x0010) // OffsetAddrInfoRQ (16 words = 32B)
	off += 16
	off += 12 // MasterNoticeInfoRQ (12B zeros)

	// CyclicInfoRQ (20B header)
	binary.LittleEndian.PutUint32(pkt[off:off+4], masterID)
	pkt[off+4] = groupNo
	pkt[off+5] = 0x00
	binary.LittleEndian.PutUint16(pkt[off+6:off+8], seq)
	binary.LittleEndian.PutUint16(pkt[off+8:off+10], 500) // timeoutValue
	binary.LittleEndian.PutUint16(pkt[off+10:off+12], 3)  // parallelOffTimeoutCount
	binary.LittleEndian.PutUint16(pkt[off+12:off+14], 0)  // parameterNo
	binary.LittleEndian.PutUint16(pkt[off+14:off+16], uint16(occupied))
	binary.LittleEndian.PutUint16(pkt[off+16:off+18], 0x0001) // cyclicTransmissionState = start
	binary.LittleEndian.PutUint16(pkt[off+18:off+20], 0x0000)
	off += 20

	for i := 0; i < occupied; i++ {
		binary.LittleEndian.PutUint32(pkt[off:off+4], slaveID)
		off += 4
	}
	copy(pkt[off:off+rwwSize], rww)
	off += rwwSize
	copy(pkt[off:off+rySize], ry)
	return pkt
}

// parseCyclicResponse parses a CLPA CC-Link IE Field Basic 0xD000 response and returns RWr and RX slices.
func parseCyclicResponse(resp []byte) (rwr []byte, rx []byte, err error) {
	if len(resp) < 9+50+bytesPerStation {
		return nil, nil, fmt.Errorf("CC-Link IE response too short (%d bytes)", len(resp))
	}
	if binary.BigEndian.Uint16(resp[0:2]) != slmpSubheaderResp {
		return nil, nil, fmt.Errorf("CC-Link IE invalid subheader 0x%04X", binary.BigEndian.Uint16(resp[0:2]))
	}
	dl := int(binary.LittleEndian.Uint16(resp[7:9]))
	if dl < 50+bytesPerStation || 9+dl > len(resp) {
		return nil, nil, fmt.Errorf("CC-Link IE invalid data length %d (packet %d)", dl, len(resp))
	}
	endCode := binary.LittleEndian.Uint16(resp[9:11])
	if endCode != 0 {
		return nil, nil, fmt.Errorf("CC-Link IE SLMP error code 0x%04X", endCode)
	}
	cyclicEndCode := binary.LittleEndian.Uint16(resp[13:15])
	if cyclicEndCode != 0 {
		return nil, nil, fmt.Errorf("CC-Link IE cyclic error code 0x%04X", cyclicEndCode)
	}
	occupied := (dl - 50) / bytesPerStation
	if occupied < 1 {
		return nil, nil, fmt.Errorf("CC-Link IE response has 0 occupied stations")
	}
	// Offset 9 + 2(EndCode) + 2(Ver) + 2(CyclicEnd) + 16(OffsetAddr) + 20(SlaveNotice) + 8(CyclicInfoRS hdr) = 59
	dataStart := 59
	rwrSize := occupied * bytesWordArea
	rxSize := occupied * bytesBitArea
	if dataStart+rwrSize+rxSize > len(resp) {
		return nil, nil, fmt.Errorf("CC-Link IE response cyclic payload truncated")
	}
	rwr = resp[dataStart : dataStart+rwrSize]
	rx = resp[dataStart+rwrSize : dataStart+rwrSize+rxSize]
	return rwr, rx, nil
}

func decodeBit(buf []byte, index int, dt string) (any, error) {
	byteIdx := index / 8
	bitIdx := uint(index % 8)
	if byteIdx < 0 || byteIdx >= len(buf) {
		return nil, fmt.Errorf("CC-Link IE bit index %d out of range", index)
	}
	on := (buf[byteIdx] & (1 << bitIdx)) != 0
	if dt == "BIT" {
		if on {
			return 1, nil
		}
		return 0, nil
	}
	return on, nil
}

func setBit(buf []byte, index int, on bool) error {
	byteIdx := index / 8
	bitIdx := uint(index % 8)
	if byteIdx < 0 || byteIdx >= len(buf) {
		return fmt.Errorf("CC-Link IE bit index %d out of range", index)
	}
	if on {
		buf[byteIdx] |= 1 << bitIdx
	} else {
		buf[byteIdx] &^= 1 << bitIdx
	}
	return nil
}

func decodeWordPoint(p *point, buf []byte) (any, error) {
	byteOff := p.index * 2
	if byteOff < 0 || byteOff+p.words*2 > len(buf) {
		return nil, fmt.Errorf("CC-Link IE word index %d out of range", p.index)
	}
	raw := buf[byteOff : byteOff+p.words*2]
	if p.wordBit >= 0 {
		word := binary.LittleEndian.Uint16(raw[:2])
		on := (word & (1 << uint(p.wordBit))) != 0
		if p.dataType == "BIT" {
			if on {
				return 1, nil
			}
			return 0, nil
		}
		return on, nil
	}
	switch p.dataType {
	case "INT16":
		return int16(binary.LittleEndian.Uint16(raw)), nil
	case "UINT16":
		return binary.LittleEndian.Uint16(raw), nil
	case "INT32":
		return int32(binary.LittleEndian.Uint32(raw)), nil
	case "UINT32":
		return binary.LittleEndian.Uint32(raw), nil
	case "FLOAT32", "FLOAT":
		return math.Float32frombits(binary.LittleEndian.Uint32(raw)), nil
	case "INT64":
		return int64(binary.LittleEndian.Uint64(raw)), nil
	case "UINT64":
		return binary.LittleEndian.Uint64(raw), nil
	case "FLOAT64", "DOUBLE":
		return math.Float64frombits(binary.LittleEndian.Uint64(raw)), nil
	default:
		return nil, fmt.Errorf("unsupported CC-Link IE data type %s", p.dataType)
	}
}

func encodeWordPoint(p *point, buf []byte, value any) error {
	byteOff := p.index * 2
	if byteOff < 0 || byteOff+p.words*2 > len(buf) {
		return fmt.Errorf("CC-Link IE word index %d out of range", p.index)
	}
	if p.wordBit >= 0 {
		b, err := toBool(value)
		if err != nil {
			return err
		}
		word := binary.LittleEndian.Uint16(buf[byteOff : byteOff+2])
		if b {
			word |= 1 << uint(p.wordBit)
		} else {
			word &^= 1 << uint(p.wordBit)
		}
		binary.LittleEndian.PutUint16(buf[byteOff:byteOff+2], word)
		return nil
	}
	switch p.dataType {
	case "INT16":
		n, err := toInt64(value)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint16(buf[byteOff:], uint16(int16(n)))
	case "UINT16":
		n, err := toUint64(value)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint16(buf[byteOff:], uint16(n))
	case "INT32":
		n, err := toInt64(value)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[byteOff:], uint32(int32(n)))
	case "UINT32":
		n, err := toUint64(value)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[byteOff:], uint32(n))
	case "FLOAT32", "FLOAT":
		f, err := toFloat64(value)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[byteOff:], math.Float32bits(float32(f)))
	case "INT64":
		n, err := toInt64(value)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(buf[byteOff:], uint64(n))
	case "UINT64":
		n, err := toUint64(value)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(buf[byteOff:], n)
	case "FLOAT64", "DOUBLE":
		f, err := toFloat64(value)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(buf[byteOff:], math.Float64bits(f))
	default:
		return fmt.Errorf("unsupported CC-Link IE write type %s", p.dataType)
	}
	return nil
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case uint64:
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
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
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
	case int16:
		return uint64(v), nil
	case int32:
		return uint64(v), nil
	case int64:
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
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
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
