package fetchwrite

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
)

const (
	opWrite byte = 0x03
	opFetch byte = 0x05

	headerSize = 16
)

type fwClient struct {
	mu      sync.Mutex
	conn    net.Conn
	timeout time.Duration
	closed  bool
}

func dialClient(ctx context.Context, endpoint string, timeout time.Duration) (*fwClient, error) {
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	return &fwClient{conn: conn, timeout: timeout}, nil
}

func (c *fwClient) Available() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.closed && c.conn != nil
}

func (c *fwClient) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func buildRequestHeader(op, org, db byte, start, length uint16) [headerSize]byte {
	var hdr [headerSize]byte
	hdr[0] = 'S'
	hdr[1] = '5'
	hdr[2] = 0x10
	hdr[3] = 0x01
	hdr[4] = 0x03
	hdr[5] = op
	hdr[6] = 0x03
	hdr[7] = 0x08
	hdr[8] = org
	hdr[9] = db
	binary.BigEndian.PutUint16(hdr[10:12], start)
	binary.BigEndian.PutUint16(hdr[12:14], length)
	hdr[14] = 0xFF
	hdr[15] = 0x02
	return hdr
}

func parseResponseHeader(hdr []byte) error {
	if len(hdr) != headerSize || hdr[0] != 'S' || hdr[1] != '5' {
		return fmt.Errorf("invalid fetch/write response header")
	}
	if hdr[8] != 0 {
		return fmt.Errorf("fetch/write error code 0x%02X", hdr[8])
	}
	return nil
}

func (c *fwClient) Fetch(org, db byte, start, unitCount uint16) ([]byte, []byte, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil {
		return nil, nil, nil, fmt.Errorf("fetch/write connection is closed")
	}
	hdr := buildRequestHeader(opFetch, org, db, start, unitCount)
	req := hdr[:]
	if err := c.writeAll(req); err != nil {
		return nil, req, nil, err
	}
	respHdr := make([]byte, headerSize)
	if err := c.readFull(respHdr); err != nil {
		return nil, req, nil, err
	}
	if err := parseResponseHeader(respHdr); err != nil {
		return nil, req, respHdr, err
	}
	wantBytes := int(unitCount) * unitBytes(org)
	data := make([]byte, wantBytes)
	if wantBytes > 0 {
		if err := c.readFull(data); err != nil {
			return nil, req, respHdr, err
		}
	}
	rawResp := make([]byte, 0, headerSize+len(data))
	rawResp = append(rawResp, respHdr...)
	rawResp = append(rawResp, data...)
	return data, req, rawResp, nil
}

func (c *fwClient) Write(org, db byte, start, unitCount uint16, payload []byte) ([]byte, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil {
		return nil, nil, fmt.Errorf("fetch/write connection is closed")
	}
	wantBytes := int(unitCount) * unitBytes(org)
	if len(payload) != wantBytes {
		return nil, nil, fmt.Errorf("fetch/write payload length %d does not match expected %d", len(payload), wantBytes)
	}
	hdr := buildRequestHeader(opWrite, org, db, start, unitCount)
	req := make([]byte, 0, headerSize+len(payload))
	req = append(req, hdr[:]...)
	req = append(req, payload...)
	if err := c.writeAll(req); err != nil {
		return req, nil, err
	}
	respHdr := make([]byte, headerSize)
	if err := c.readFull(respHdr); err != nil {
		return req, nil, err
	}
	if err := parseResponseHeader(respHdr); err != nil {
		return req, respHdr, err
	}
	return req, respHdr, nil
}

func (c *fwClient) writeAll(data []byte) error {
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	for len(data) > 0 {
		n, err := c.conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (c *fwClient) readFull(dst []byte) error {
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := io.ReadFull(c.conn, dst)
	return err
}

func decodeValue(raw []byte, addr Address, variable *collection.Variable) (any, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	off := int(addr.ByteOffset)
	if off < 0 || off+int(addr.ByteLength) > len(raw) {
		return nil, fmt.Errorf("short fetch/write response for %s", variable.Name)
	}
	data := raw[off : off+int(addr.ByteLength)]
	switch strings.ToLower(strings.TrimSpace(variable.DataType)) {
	case "bit":
		if extractBit(data, addr) {
			return 1, nil
		}
		return 0, nil
	case "bool":
		return extractBit(data, addr), nil
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
	case "byte_array":
		out := make([]int, len(data))
		for i, b := range data {
			out[i] = int(b)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported fetch/write data type %q", variable.DataType)
	}
}

func extractBit(data []byte, addr Address) bool {
	if addr.WordBit && len(data) >= 2 {
		word := binary.BigEndian.Uint16(data[:2])
		return (word & (1 << addr.Bit)) != 0
	}
	return (data[0] & (1 << addr.Bit)) != 0
}

func applyBit(buf []byte, addr Address, set bool) error {
	off := int(addr.ByteOffset)
	if addr.WordBit {
		if off+2 > len(buf) {
			return fmt.Errorf("short buffer for word bit write")
		}
		word := binary.BigEndian.Uint16(buf[off : off+2])
		if set {
			word |= 1 << addr.Bit
		} else {
			word &^= 1 << addr.Bit
		}
		binary.BigEndian.PutUint16(buf[off:off+2], word)
		return nil
	}
	if off >= len(buf) {
		return fmt.Errorf("short buffer for byte bit write")
	}
	if set {
		buf[off] |= 1 << addr.Bit
	} else {
		buf[off] &^= 1 << addr.Bit
	}
	return nil
}

func encodeValue(addr Address, variable *collection.Variable, raw any) ([]byte, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	switch strings.ToLower(strings.TrimSpace(variable.DataType)) {
	case "int16":
		v, err := toInt64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(int16(v)))
		return buf, nil
	case "uint16":
		v, err := toUint64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(v))
		return buf, nil
	case "int32":
		v, err := toInt64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(int32(v)))
		return buf, nil
	case "uint32":
		v, err := toUint64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(v))
		return buf, nil
	case "int64":
		v, err := toInt64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(v))
		return buf, nil
	case "uint64":
		v, err := toUint64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, v)
		return buf, nil
	case "float32":
		v, err := toFloat64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, math.Float32bits(float32(v)))
		return buf, nil
	case "float64":
		v, err := toFloat64(raw)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, math.Float64bits(v))
		return buf, nil
	case "string":
		s, ok := raw.(string)
		if !ok {
			s = fmt.Sprint(raw)
		}
		buf := make([]byte, addr.ByteLength)
		copy(buf, s)
		return buf, nil
	case "byte_array":
		buf := make([]byte, addr.ByteLength)
		switch v := raw.(type) {
		case []byte:
			copy(buf, v)
		case []int:
			for i := 0; i < len(v) && i < len(buf); i++ {
				buf[i] = byte(v[i])
			}
		default:
			return nil, fmt.Errorf("unsupported fetch/write byte_array value %T", raw)
		}
		return buf, nil
	default:
		return nil, fmt.Errorf("unsupported fetch/write data type %q", variable.DataType)
	}
}

func toBool(raw any) (bool, error) {
	switch v := raw.(type) {
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
		return strconv.ParseBool(strings.TrimSpace(v))
	default:
		return false, fmt.Errorf("invalid fetch/write bool value %T", raw)
	}
}

func toInt64(raw any) (int64, error) {
	switch v := raw.(type) {
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
	default:
		return 0, fmt.Errorf("invalid fetch/write integer value %T", raw)
	}
}

func toUint64(raw any) (uint64, error) {
	switch v := raw.(type) {
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
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %d", v)
		}
		return uint64(v), nil
	case float32:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %v", v)
		}
		return uint64(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("negative unsigned value %v", v)
		}
		return uint64(v), nil
	case string:
		return strconv.ParseUint(strings.TrimSpace(v), 10, 64)
	default:
		return 0, fmt.Errorf("invalid fetch/write unsigned value %T", raw)
	}
}

func toFloat64(raw any) (float64, error) {
	switch v := raw.(type) {
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
	default:
		return 0, fmt.Errorf("invalid fetch/write float value %T", raw)
	}
}
