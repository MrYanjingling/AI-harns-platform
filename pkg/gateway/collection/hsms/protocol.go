package hsms

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

// HSMS SType values per SEMI E37 Table 7.
const (
	STypeData        uint8 = 0
	STypeSelectReq   uint8 = 1
	STypeSelectRsp   uint8 = 2
	STypeDeselectReq uint8 = 3
	STypeDeselectRsp uint8 = 4
	STypeLinktestReq uint8 = 5
	STypeLinktestRsp uint8 = 6
	STypeRejectReq   uint8 = 7
	STypeSeparateReq uint8 = 9

	maxHSMSMessageLen = 16 * 1024 * 1024
)

// Header is the 10-byte HSMS message header (big-endian on wire).
type Header struct {
	SessionID   uint16
	WBit        bool
	Stream      uint8 // 0..127
	Function    uint8
	PType       uint8
	SType       uint8
	SystemBytes uint32
}

// Message is a complete HSMS frame (10-byte header + optional SECS-II body).
type Message struct {
	Header Header
	Body   []byte
}

// Encode serializes the HSMS message with its 4-byte big-endian length prefix
// (length = 10 + len(Body), excluding the 4-byte length prefix itself).
func (m *Message) Encode() []byte {
	total := 10 + len(m.Body)
	buf := make([]byte, 4+total)
	binary.BigEndian.PutUint32(buf[0:4], uint32(total))
	binary.BigEndian.PutUint16(buf[4:6], m.Header.SessionID)
	b2 := m.Header.Stream & 0x7F
	if m.Header.WBit {
		b2 |= 0x80
	}
	buf[6] = b2
	buf[7] = m.Header.Function
	buf[8] = m.Header.PType
	buf[9] = m.Header.SType
	binary.BigEndian.PutUint32(buf[10:14], m.Header.SystemBytes)
	copy(buf[14:], m.Body)
	return buf
}

// ReadMessage reads a single framed HSMS message from r.
func ReadMessage(r io.Reader) (*Message, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint32(lenBuf[:])
	if msgLen < 10 || msgLen > maxHSMSMessageLen {
		return nil, fmt.Errorf("invalid HSMS message length %d", msgLen)
	}
	payload := make([]byte, msgLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	msg := &Message{
		Header: Header{
			SessionID:   binary.BigEndian.Uint16(payload[0:2]),
			WBit:        (payload[2] & 0x80) != 0,
			Stream:      payload[2] & 0x7F,
			Function:    payload[3],
			PType:       payload[4],
			SType:       payload[5],
			SystemBytes: binary.BigEndian.Uint32(payload[6:10]),
		},
	}
	if msgLen > 10 {
		msg.Body = payload[10:]
	}
	return msg, nil
}

func buildControlMessage(sType uint8, statusByte uint8, systemBytes uint32) *Message {
	return &Message{
		Header: Header{
			SessionID:   0xFFFF,
			Function:    statusByte,
			PType:       0,
			SType:       sType,
			SystemBytes: systemBytes,
		},
	}
}

func buildDataMessage(deviceID uint16, stream, function uint8, wBit bool, systemBytes uint32, body []byte) *Message {
	return &Message{
		Header: Header{
			SessionID:   deviceID,
			WBit:        wBit,
			Stream:      stream & 0x7F,
			Function:    function,
			PType:       0,
			SType:       STypeData,
			SystemBytes: systemBytes,
		},
		Body: body,
	}
}

// indexKind identifies how a tag address indexes into a SECS-II message body.
type indexKind uint8

const (
	indexNone indexKind = iota
	indexSVID
	indexList
)

type point struct {
	variable  *collection.Variable
	stream    uint8
	function  uint8
	kind      indexKind
	svid      string
	svidNum   uint32
	svidIsNum bool
	listPath  []int // 1-based list indices ([1]..[N], max 10 levels)
	dataType  string
}

// isReportOnly reports whether (stream, function) is an equipment-initiated report/event message.
func isReportOnly(stream, function uint8) bool {
	switch {
	case stream == 2 && (function == 33 || function == 35 || function == 37):
		return true
	case stream == 5 && function == 1:
		return true
	case stream == 6 && (function == 11 || function == 13 || function == 21 || function == 25 || function == 27 || function == 29):
		return true
	case stream == 12 && function == 19:
		return true
	default:
		return false
	}
}

// parseAddress parses SECS-II / HSMS tag addresses:
//   - S<stream>F<function>               (e.g. S1F1, S6F11)
//   - S<stream>F<function>.<svid>        (e.g. S1F3.1001, S1F12.MEASURE1)
//   - S<stream>F<function>[i1][i2]..[i10] (1-based list path, e.g. S1F1[1], S6F11[3][1][2])
func parseAddress(variable *collection.Variable) (*point, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	raw := strings.TrimSpace(variable.Address)
	upper := strings.ToUpper(raw)
	if len(upper) < 4 || upper[0] != 'S' {
		return nil, fmt.Errorf("HSMS address %q must start with S<stream>F<function>", variable.Address)
	}
	fIdx := strings.IndexByte(upper[1:], 'F')
	if fIdx <= 0 {
		return nil, fmt.Errorf("HSMS address %q missing 'F'", variable.Address)
	}
	fIdx++ // adjust for slice offset 1
	streamVal, err := strconv.ParseUint(upper[1:fIdx], 10, 8)
	if err != nil || streamVal > 127 {
		return nil, fmt.Errorf("HSMS stream in %q must be 0..127", variable.Address)
	}
	rest := raw[fIdx+1:]
	funcEnd := 0
	for funcEnd < len(rest) && rest[funcEnd] >= '0' && rest[funcEnd] <= '9' {
		funcEnd++
	}
	if funcEnd == 0 {
		return nil, fmt.Errorf("HSMS function missing in %q", variable.Address)
	}
	funcVal, err := strconv.ParseUint(rest[:funcEnd], 10, 8)
	if err != nil || funcVal > 255 {
		return nil, fmt.Errorf("HSMS function in %q must be 0..255", variable.Address)
	}

	dt := strings.ToUpper(strings.TrimSpace(variable.DataType))
	if !isSupportedDataType(dt) {
		return nil, fmt.Errorf("HSMS data type %q is not supported", variable.DataType)
	}

	p := &point{
		variable: variable,
		stream:   uint8(streamVal),
		function: uint8(funcVal),
		kind:     indexNone,
		dataType: dt,
	}

	tail := rest[funcEnd:]
	if tail == "" {
		return p, nil
	}
	if tail[0] == '.' {
		svid := strings.TrimSpace(tail[1:])
		if svid == "" || len(svid) > 64 {
			return nil, fmt.Errorf("HSMS SVID in %q must be 1..64 chars", variable.Address)
		}
		p.kind = indexSVID
		p.svid = svid
		if n, err := strconv.ParseUint(svid, 10, 32); err == nil {
			p.svidNum = uint32(n)
			p.svidIsNum = true
		}
		return p, nil
	}
	if tail[0] == '[' {
		var indices []int
		cur := tail
		for len(cur) > 0 {
			if cur[0] != '[' {
				return nil, fmt.Errorf("HSMS list index syntax invalid in %q", variable.Address)
			}
			closeIdx := strings.IndexByte(cur, ']')
			if closeIdx <= 1 {
				return nil, fmt.Errorf("HSMS list index unclosed in %q", variable.Address)
			}
			idxVal, err := strconv.Atoi(cur[1:closeIdx])
			if err != nil || idxVal < 1 || idxVal > 255 {
				return nil, fmt.Errorf("HSMS list index %q must be 1..255", cur[1:closeIdx])
			}
			indices = append(indices, idxVal)
			if len(indices) > 10 {
				return nil, fmt.Errorf("HSMS list index depth exceeds 10 in %q", variable.Address)
			}
			cur = cur[closeIdx+1:]
		}
		p.kind = indexList
		p.listPath = indices
		return p, nil
	}
	return nil, fmt.Errorf("HSMS address %q has unexpected suffix %q", variable.Address, tail)
}

func isSupportedDataType(dt string) bool {
	switch dt {
	case "BOOL", "BIT",
		"INT8", "INT16", "INT32", "INT64",
		"UINT8", "UINT16", "UINT32", "UINT64",
		"FLOAT32", "FLOAT", "FLOAT64", "DOUBLE",
		"STRING":
		return true
	default:
		return false
	}
}

// extractItem locates the target SECS-II item inside root according to point's index.
func extractItem(root *Item, stream, function uint8, p *point) (*Item, error) {
	if root == nil {
		return nil, fmt.Errorf("empty SECS-II body")
	}
	switch p.kind {
	case indexNone:
		return root, nil
	case indexList:
		cur := root
		for _, oneBased := range p.listPath {
			if cur.Type != ItemList {
				return nil, fmt.Errorf("expected SECS-II List at index [%d]", oneBased)
			}
			zeroBased := oneBased - 1
			if zeroBased < 0 || zeroBased >= len(cur.Children) {
				return nil, fmt.Errorf("SECS-II List index [%d] out of range (len=%d)", oneBased, len(cur.Children))
			}
			cur = cur.Children[zeroBased]
		}
		return cur, nil
	case indexSVID:
		// S1F12 (Status Variable Namelist Reply): <L[n] <L[3] <SVID> <SVNAME> <UNITS>> ...>
		// Matches elem[0] == svid and returns elem[1].
		if stream == 1 && function == 12 && root.Type == ItemList {
			for _, entry := range root.Children {
				if entry.Type == ItemList && len(entry.Children) >= 2 {
					if matchesSVIDItem(entry.Children[0], p) {
						return entry.Children[1], nil
					}
				}
			}
			return nil, fmt.Errorf("SVID %q not found in S1F12", p.svid)
		}
		// S1F4 (Selected Equipment Status Data): <L[1] <val>> for single-SVID S1F3 request.
		if root.Type == ItemList && len(root.Children) == 1 {
			return root.Children[0], nil
		}
		return root, nil
	default:
		return root, nil
	}
}

func matchesSVIDItem(it *Item, p *point) bool {
	if it == nil {
		return false
	}
	if it.Type == ItemASCII || it.Type == ItemJIS8 {
		return string(it.Raw) == p.svid
	}
	if p.svidIsNum {
		if u, err := itemToUint64(it); err == nil && u == uint64(p.svidNum) {
			return true
		}
	}
	return false
}
