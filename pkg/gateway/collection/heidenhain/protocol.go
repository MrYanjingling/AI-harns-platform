package heidenhain

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"time"

	"harns-platform/pkg/gateway/collection"
)

type systemParams struct {
	markersStart     uint32
	markersCount     uint32
	inputsStart      uint32
	inputsCount      uint32
	outputsStart     uint32
	outputsCount     uint32
	countersStart    uint32
	countersCount    uint32
	timersStart      uint32
	timersCount      uint32
	wordsStart       uint32
	wordsCount       uint32
	stringsStart     uint32
	stringsCount     uint32
	maxStringLength  uint32
	inputWordsStart  uint32
	inputWordsCount  uint32
	outputWordsStart uint32
	outputWordsCount uint32
}

type lsv2Client struct {
	conn    net.Conn
	timeout time.Duration
	params  systemParams
	closed  bool
}

func openLSV2Client(host string, port int, timeout time.Duration) (*lsv2Client, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), timeout)
	if err != nil {
		return nil, err
	}
	client := &lsv2Client{conn: conn, timeout: timeout}
	if err := client.loginAll(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.readSystemParams(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *lsv2Client) Available() bool { return c != nil && c.conn != nil && !c.closed }
func (c *lsv2Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	c.closed = true
	return c.conn.Close()
}

func (c *lsv2Client) loginAll() error {
	for _, login := range []string{"INSPECT", "DNC", "PLCDEBUG", "MONITOR"} {
		if err := c.request("A_LG", []byte(login), "T_OK"); err != nil {
			return err
		}
	}
	return nil
}

func (c *lsv2Client) readSystemParams() error {
	body, err := c.requestBody("R_PR", nil, "S_PR")
	if err != nil {
		return err
	}
	if len(body) != 0x78 && len(body) != 0x7c {
		return fmt.Errorf("unexpected LSV2 system parameter length %d", len(body))
	}
	read := func(offset int) uint32 { return binary.BigEndian.Uint32(body[offset : offset+4]) }
	c.params = systemParams{
		markersStart: read(0), markersCount: read(4),
		inputsStart: read(8), inputsCount: read(12),
		outputsStart: read(16), outputsCount: read(20),
		countersStart: read(24), countersCount: read(28),
		timersStart: read(32), timersCount: read(36),
		wordsStart: read(40), wordsCount: read(44),
		stringsStart: read(48), stringsCount: read(52),
		maxStringLength: uint32(body[0x38]),
		inputWordsStart: read(0x40), inputWordsCount: read(0x44),
		outputWordsStart: read(0x48), outputWordsCount: read(0x4c),
	}
	return nil
}

func (c *lsv2Client) request(cmd string, body []byte, expected string) error {
	_, err := c.requestBody(cmd, body, expected)
	return err
}

func (c *lsv2Client) requestBody(cmd string, body []byte, expected string) ([]byte, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("LSV2 client is not connected")
	}
	if len(cmd) != 4 {
		return nil, fmt.Errorf("invalid LSV2 command %q", cmd)
	}
	packet := make([]byte, 8+len(body))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(body)))
	copy(packet[4:8], cmd)
	copy(packet[8:], body)
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(packet); err != nil {
		return nil, err
	}
	return c.readResponse(expected)
}

func (c *lsv2Client) readResponse(expected string) ([]byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header[:4])
	if length > 1<<20 {
		return nil, fmt.Errorf("LSV2 response is too large")
	}
	cmd := string(header[4:8])
	body := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(c.conn, body); err != nil {
			return nil, err
		}
	}
	if cmd == "T_ER" {
		return nil, fmt.Errorf("LSV2 returned error")
	}
	if expected != "" && cmd != expected {
		return nil, fmt.Errorf("unexpected LSV2 response %q, want %q", cmd, expected)
	}
	return body, nil
}

func (c *lsv2Client) Read(ctx context.Context, address Address, variable *collection.Variable) (any, error) {
	_ = ctx
	if address.Kind == kindPLC {
		return c.readPLC(address, variable.DataType)
	}
	return c.readCNC(address, variable.DataType)
}

func (c *lsv2Client) readPLC(address Address, dataType string) (any, error) {
	start, ok := c.plcStart(address)
	if !ok {
		return nil, fmt.Errorf("unsupported Heidenhain PLC area %c", address.Area)
	}
	count := address.Size
	if address.Area == 'S' {
		start = c.params.stringsStart + uint32(address.Number)*c.params.maxStringLength
	}
	if count <= 0 || count > 255 {
		return nil, fmt.Errorf("invalid Heidenhain PLC read length")
	}
	body := make([]byte, 5)
	binary.BigEndian.PutUint32(body[:4], start)
	body[4] = byte(count)
	data, err := c.requestBody("R_MB", body, "S_MB")
	if err != nil {
		return nil, err
	}
	if len(data) < count {
		return nil, fmt.Errorf("short Heidenhain PLC response")
	}
	return decodePLC(data[:count], address, dataType)
}

func (c *lsv2Client) plcStart(address Address) (uint32, bool) {
	switch address.Area {
	case 'B', 'W', 'D':
		return c.params.wordsStart + uint32(address.Number), true
	case 'N':
		return c.params.inputWordsStart + uint32(address.Number), true
	case 'U':
		return c.params.outputWordsStart + uint32(address.Number), true
	case 'M':
		return c.params.markersStart + uint32(address.Number), true
	case 'I':
		return c.params.inputsStart + uint32(address.Number), true
	case 'O':
		return c.params.outputsStart + uint32(address.Number), true
	case 'T':
		return c.params.timersStart + uint32(address.Number), true
	case 'C':
		return c.params.countersStart + uint32(address.Number), true
	case 'S':
		return c.params.stringsStart + uint32(address.Number)*c.params.maxStringLength, true
	default:
		return 0, false
	}
}

func decodePLC(data []byte, address Address, dataType string) (any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty Heidenhain PLC data")
	}
	if address.Bit > 0 || strings.EqualFold(dataType, "bit") || strings.EqualFold(dataType, "bool") {
		bit := data[0]
		if address.Bit > 0 {
			bit = (bit >> address.Bit) & 1
		} else {
			bit &= 1
		}
		if strings.EqualFold(dataType, "bool") {
			return bit != 0, nil
		}
		return int(bit), nil
	}
	switch strings.ToLower(dataType) {
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
	case "float":
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "int64":
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "uint64":
		return binary.LittleEndian.Uint64(data), nil
	case "double":
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	case "string":
		return strings.TrimRight(string(data), "\x00"), nil
	default:
		return nil, fmt.Errorf("unsupported Heidenhain data type %q", dataType)
	}
}

func (c *lsv2Client) readCNC(address Address, dataType string) (any, error) {
	switch address.Kind {
	case kindRunState:
		body, err := c.readRI(26)
		if err != nil {
			return nil, err
		}
		if len(body) < 2 {
			return nil, fmt.Errorf("short LSV2 run state")
		}
		return binary.BigEndian.Uint16(body), nil
	case kindControlMode:
		body, err := c.readRI(23)
		if err != nil {
			return nil, err
		}
		if len(body) < 2 {
			return nil, fmt.Errorf("short LSV2 control mode")
		}
		return binary.BigEndian.Uint16(body), nil
	case kindProgramMain, kindProgramCurrent, kindProgramLine:
		body, err := c.readRI(24)
		if err != nil {
			return nil, err
		}
		return decodeProgramState(body, address.Kind)
	case kindFeedOverride, kindSpindleOverride, kindRapidOverride:
		body, err := c.readRI(25)
		if err != nil {
			return nil, err
		}
		return decodeOverride(body, address.Kind)
	case kindToolNumber, kindToolIndex, kindToolAxis:
		body, err := c.readRI(51)
		if err != nil {
			return nil, err
		}
		return decodeTool(body, address.Kind)
	case kindToolLength, kindToolRadius:
		body, err := c.readRI(51)
		if err != nil {
			return nil, err
		}
		return decodeToolDouble(body, address.Kind)
	case kindMachinePosition:
		body, err := c.readRI(22)
		if err != nil {
			return nil, err
		}
		return decodeMachinePosition(body, address.Key)
	case kindParameter:
		body, err := c.requestBody("R_MC", append([]byte(address.Key), 0), "S_MC")
		if err != nil {
			return nil, err
		}
		return convertParameter(body, dataType)
	default:
		return nil, fmt.Errorf("unsupported Heidenhain CNC address")
	}
}

func (c *lsv2Client) readRI(param uint16) ([]byte, error) {
	body := make([]byte, 2)
	binary.BigEndian.PutUint16(body, param)
	return c.requestBody("R_RI", body, "S_RI")
}

func decodeProgramState(body []byte, kind addressKind) (any, error) {
	if len(body) < 4 {
		return nil, fmt.Errorf("short LSV2 program state")
	}
	if kind == kindProgramLine {
		return binary.BigEndian.Uint32(body[:4]), nil
	}
	text := body[4:]
	main := text
	if index := strings.IndexByte(string(text), 0); index >= 0 {
		main = text[:index]
		if kind == kindProgramCurrent {
			rest := text[index+1:]
			if end := strings.IndexByte(string(rest), 0); end >= 0 {
				return string(rest[:end]), nil
			}
			return string(rest), nil
		}
	}
	return string(main), nil
}

func decodeOverride(body []byte, kind addressKind) (any, error) {
	if len(body) < 12 {
		return nil, fmt.Errorf("short LSV2 override response")
	}
	index := 0
	if kind == kindSpindleOverride {
		index = 4
	}
	if kind == kindRapidOverride {
		index = 8
	}
	return float64(binary.BigEndian.Uint32(body[index:index+4])) / 100, nil
}

func decodeTool(body []byte, kind addressKind) (any, error) {
	if len(body) < 8 {
		return nil, fmt.Errorf("short LSV2 tool response")
	}
	switch kind {
	case kindToolNumber:
		return binary.BigEndian.Uint32(body[:4]), nil
	case kindToolIndex:
		return binary.BigEndian.Uint16(body[4:6]), nil
	default:
		return binary.BigEndian.Uint16(body[6:8]), nil
	}
}

func decodeToolDouble(body []byte, kind addressKind) (any, error) {
	if len(body) < 24 {
		return nil, fmt.Errorf("short LSV2 tool response")
	}
	offset := 8
	if kind == kindToolRadius {
		offset = 16
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(body[offset : offset+8])), nil
}

func decodeMachinePosition(body []byte, axis string) (any, error) {
	if len(body) < 2 {
		return nil, fmt.Errorf("short LSV2 axis location response")
	}
	count := int(body[1])
	values := strings.Split(string(body[2:]), "\x00")
	values = values[:len(values)-1]
	for index, value := range values {
		if !strings.EqualFold(strings.TrimSpace(value), axis) {
			continue
		}
		candidates := make([]int, 0, 2)
		if index > 0 {
			candidates = append(candidates, index-1)
		}
		if index+1 < len(values) {
			candidates = append(candidates, index+1)
		}
		if index-count >= 0 {
			candidates = append(candidates, index-count)
		}
		for _, candidate := range candidates {
			var number float64
			if _, err := fmt.Sscanf(strings.TrimSpace(values[candidate]), "%f", &number); err == nil {
				return number, nil
			}
		}
	}
	return nil, fmt.Errorf("LSV2 axis %q not found", axis)
}

func convertParameter(body []byte, dataType string) (any, error) {
	if strings.EqualFold(dataType, "string") {
		return strings.TrimRight(string(body), "\x00"), nil
	}
	return decodePLC(body, Address{Size: len(body)}, dataType)
}
