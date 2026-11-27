// Package fxlinks implements Mitsubishi FX-Links (Computer Link Format 1/4)
// per Mitsubishi FX Series User's Manual - Data Communication Edition (JY997D16901).
package fxlinks

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
	ctrlENQ byte = 0x05
	ctrlSTX byte = 0x02
	ctrlETX byte = 0x03
	ctrlACK byte = 0x06
	ctrlNAK byte = 0x15

	maxWordBatch = 64
	maxBitBatch  = 64
)

type point struct {
	variable *collection.Variable
	device   string // D, T, C, R, X, Y, M, S
	number   uint32
	isBit    bool
	wordBit  int // >=0 for D0.3
	words    int
	dataType string
}

type readBatch struct {
	device string
	isBit  bool
	start  uint32
	count  int
	points []*point
}

func parseAddress(variable *collection.Variable) (*point, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	raw := strings.ToUpper(strings.TrimSpace(variable.Address))
	if len(raw) < 2 {
		return nil, fmt.Errorf("FX-Links address %q is invalid", variable.Address)
	}
	prefix := raw[:1]
	rest := raw[1:]
	isBit := false
	switch prefix {
	case "X", "Y", "M", "S":
		isBit = true
	case "D", "T", "C", "R":
		isBit = false
	default:
		return nil, fmt.Errorf("FX-Links device %q is not supported", prefix)
	}
	numText := rest
	wordBit := -1
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		numText = rest[:dot]
		b, err := strconv.Atoi(rest[dot+1:])
		if err != nil || b < 0 || b > 15 {
			return nil, fmt.Errorf("FX-Links bit suffix %q must be 0..15", rest[dot+1:])
		}
		wordBit = b
	}
	num, err := strconv.ParseUint(numText, 10, 32)
	if err != nil || num > 9999 {
		return nil, fmt.Errorf("FX-Links device number %q must be 0..9999", numText)
	}
	dt := strings.ToUpper(strings.TrimSpace(variable.DataType))
	p := &point{
		variable: variable,
		device:   prefix,
		number:   uint32(num),
		isBit:    isBit,
		wordBit:  wordBit,
		dataType: dt,
	}
	if isBit {
		if wordBit >= 0 {
			return nil, fmt.Errorf("FX-Links bit device %q must not have bit suffix", variable.Address)
		}
		if dt != "BIT" && dt != "BOOL" {
			return nil, fmt.Errorf("FX-Links bit device %q only supports BIT/BOOL", variable.Address)
		}
		p.words = 1
		return p, nil
	}
	if wordBit >= 0 {
		if dt != "BIT" && dt != "BOOL" {
			return nil, fmt.Errorf("FX-Links word bit %q requires BIT/BOOL", variable.Address)
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
		return 0, fmt.Errorf("FX-Links data type %q is not supported", dt)
	}
}

func formatDevice(device string, number uint32) string {
	return fmt.Sprintf("%s%04d", device, number)
}

func sumCheck(data []byte) string {
	var sum byte
	for _, b := range data {
		sum += b
	}
	return fmt.Sprintf("%02X", sum)
}

func buildReadRequest(station, waitTime, format int, useSum bool, isBit bool, device string, start uint32, count int) []byte {
	cmd := "WR"
	if isBit {
		cmd = "BR"
	}
	body := fmt.Sprintf("%02XFF%s%1X%s%02X", station&0xFF, cmd, waitTime&0x0F, formatDevice(device, start), count&0xFF)
	return finalizeFrame(ctrlENQ, body, useSum, format)
}

func buildWordWriteRequest(station, waitTime, format int, useSum bool, device string, start uint32, words []uint16) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%02XFFWW%1X%s%02X", station&0xFF, waitTime&0x0F, formatDevice(device, start), len(words)&0xFF))
	for _, w := range words {
		sb.WriteString(fmt.Sprintf("%04X", w))
	}
	return finalizeFrame(ctrlENQ, sb.String(), useSum, format)
}

func buildBitWriteRequest(station, waitTime, format int, useSum bool, device string, start uint32, bits []bool) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%02XFFBW%1X%s%02X", station&0xFF, waitTime&0x0F, formatDevice(device, start), len(bits)&0xFF))
	for _, b := range bits {
		if b {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	return finalizeFrame(ctrlENQ, sb.String(), useSum, format)
}

func finalizeFrame(lead byte, body string, useSum bool, format int) []byte {
	frame := make([]byte, 0, 1+len(body)+4)
	frame = append(frame, lead)
	frame = append(frame, body...)
	if useSum {
		frame = append(frame, sumCheck([]byte(body))...)
	}
	if format == 4 {
		frame = append(frame, '\r', '\n')
	}
	return frame
}

func parseReadResponse(resp []byte, useSum bool, isBit bool, count int) ([]byte, error) {
	if len(resp) < 6 {
		return nil, fmt.Errorf("FX-Links read response too short (%d bytes)", len(resp))
	}
	if resp[0] == ctrlNAK {
		return nil, fmt.Errorf("FX-Links NAK response: %q", string(resp))
	}
	if resp[0] != ctrlSTX {
		return nil, fmt.Errorf("FX-Links unexpected response lead 0x%02X", resp[0])
	}
	etxIdx := -1
	for i := 1; i < len(resp); i++ {
		if resp[i] == ctrlETX {
			etxIdx = i
			break
		}
	}
	if etxIdx < 5 {
		return nil, fmt.Errorf("FX-Links missing ETX in read response")
	}
	if useSum {
		if etxIdx+3 > len(resp) {
			return nil, fmt.Errorf("FX-Links missing checksum in read response")
		}
		// Standard JY997D16901 sums Station..ETX (excluding STX); some simulators
		// (e.g. plcsim) include STX (0x02) in the response sum.
		want := sumCheck(resp[1 : etxIdx+1])
		wantInclSTX := sumCheck(resp[0 : etxIdx+1])
		got := string(resp[etxIdx+1 : etxIdx+3])
		if !strings.EqualFold(want, got) && !strings.EqualFold(wantInclSTX, got) {
			return nil, fmt.Errorf("FX-Links checksum mismatch: got %s want %s", got, want)
		}
	}
	payload := resp[5:etxIdx] // skip STX + station(2) + PC(2)
	if isBit {
		if len(payload) < count {
			return nil, fmt.Errorf("FX-Links bit payload too short: got %d want %d", len(payload), count)
		}
		out := make([]byte, count)
		for i := 0; i < count; i++ {
			if payload[i] == '1' {
				out[i] = 1
			}
		}
		return out, nil
	}
	if len(payload) < count*4 {
		return nil, fmt.Errorf("FX-Links word payload too short: got %d want %d", len(payload), count*4)
	}
	out := make([]byte, count*2)
	for i := 0; i < count; i++ {
		v, err := strconv.ParseUint(string(payload[i*4:i*4+4]), 16, 16)
		if err != nil {
			return nil, fmt.Errorf("FX-Links invalid hex word %q: %w", string(payload[i*4:i*4+4]), err)
		}
		binary.LittleEndian.PutUint16(out[i*2:], uint16(v))
	}
	return out, nil
}

func parseWriteResponse(resp []byte) error {
	if len(resp) < 5 {
		return fmt.Errorf("FX-Links write response too short (%d bytes)", len(resp))
	}
	if resp[0] == ctrlACK {
		return nil
	}
	if resp[0] == ctrlNAK {
		return fmt.Errorf("FX-Links write NAK: %q", string(resp))
	}
	return fmt.Errorf("FX-Links unexpected write response lead 0x%02X", resp[0])
}

func buildReadBatches(variables []*collection.Variable) ([]*readBatch, error) {
	pts := make([]*point, 0, len(variables))
	for _, v := range variables {
		if v == nil || v.IsCalculated || strings.TrimSpace(v.Address) == "" {
			continue
		}
		p, err := parseAddress(v)
		if err != nil {
			return nil, err
		}
		pts = append(pts, p)
	}
	if len(pts) == 0 {
		return nil, nil
	}
	sort.Slice(pts, func(i, j int) bool {
		if pts[i].device != pts[j].device {
			return pts[i].device < pts[j].device
		}
		return pts[i].number < pts[j].number
	})
	batches := make([]*readBatch, 0)
	for _, p := range pts {
		maxSpan := maxWordBatch
		if p.isBit {
			maxSpan = maxBitBatch
		}
		if len(batches) > 0 {
			last := batches[len(batches)-1]
			if last.device == p.device && last.isBit == p.isBit {
				newEnd := p.number + uint32(p.words)
				curEnd := last.start + uint32(last.count)
				if newEnd < curEnd {
					newEnd = curEnd
				}
				if int(newEnd-last.start) <= maxSpan {
					last.count = int(newEnd - last.start)
					last.points = append(last.points, p)
					continue
				}
			}
		}
		batches = append(batches, &readBatch{
			device: p.device,
			isBit:  p.isBit,
			start:  p.number,
			count:  p.words,
			points: []*point{p},
		})
	}
	return batches, nil
}

func decodePoint(p *point, batchData []byte, batchStart uint32) (any, error) {
	offset := int(p.number - batchStart)
	if p.isBit {
		if offset >= len(batchData) {
			return nil, fmt.Errorf("FX-Links bit offset out of range")
		}
		on := batchData[offset] != 0
		if p.dataType == "BIT" {
			if on {
				return 1, nil
			}
			return 0, nil
		}
		return on, nil
	}
	byteOff := offset * 2
	if byteOff+p.words*2 > len(batchData) {
		return nil, fmt.Errorf("FX-Links word offset out of range")
	}
	raw := batchData[byteOff : byteOff+p.words*2]
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
		return nil, fmt.Errorf("unsupported FX-Links data type %s", p.dataType)
	}
}

func encodeWordValue(dt string, value any) ([]uint16, error) {
	switch dt {
	case "INT16":
		n, err := toInt64(value)
		if err != nil {
			return nil, err
		}
		return []uint16{uint16(int16(n))}, nil
	case "UINT16":
		n, err := toUint64(value)
		if err != nil {
			return nil, err
		}
		return []uint16{uint16(n)}, nil
	case "INT32":
		n, err := toInt64(value)
		if err != nil {
			return nil, err
		}
		u := uint32(int32(n))
		return []uint16{uint16(u), uint16(u >> 16)}, nil
	case "UINT32":
		n, err := toUint64(value)
		if err != nil {
			return nil, err
		}
		u := uint32(n)
		return []uint16{uint16(u), uint16(u >> 16)}, nil
	case "FLOAT32", "FLOAT":
		f, err := toFloat64(value)
		if err != nil {
			return nil, err
		}
		u := math.Float32bits(float32(f))
		return []uint16{uint16(u), uint16(u >> 16)}, nil
	case "INT64":
		n, err := toInt64(value)
		if err != nil {
			return nil, err
		}
		u := uint64(n)
		return []uint16{uint16(u), uint16(u >> 16), uint16(u >> 32), uint16(u >> 48)}, nil
	case "UINT64":
		u, err := toUint64(value)
		if err != nil {
			return nil, err
		}
		return []uint16{uint16(u), uint16(u >> 16), uint16(u >> 32), uint16(u >> 48)}, nil
	case "FLOAT64", "DOUBLE":
		f, err := toFloat64(value)
		if err != nil {
			return nil, err
		}
		u := math.Float64bits(f)
		return []uint16{uint16(u), uint16(u >> 16), uint16(u >> 32), uint16(u >> 48)}, nil
	default:
		return nil, fmt.Errorf("unsupported FX-Links write type %s", dt)
	}
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
