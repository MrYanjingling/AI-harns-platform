package comli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

const (
	stx byte = 0x02
	etx byte = 0x03

	messageReadRegisters  byte = '2'
	messageWriteRegisters byte = '0'
	messageWriteBit       byte = '4'
	messageAck            byte = '1'

	areaBits      byte = 0
	areaRegisters byte = 1

	maxBitAddress      = 0x3fff
	maxRegisterAddress = 0xbff
	maxReadSpan        = 0x3f
)

const hexDigits = "0123456789ABCDEF"

// comliPoint is one parsed COMLI tag.
type comliPoint struct {
	variable *collection.Variable
	identity byte
	area     byte
	offset   uint16
	bit      uint8
	quantity int
	dataType string
}

func (p *comliPoint) startByte() int {
	if p.area == areaRegisters {
		return int(p.offset) * 2
	}
	return int(p.offset) / 8
}

// readCommand groups contiguous points that can share one COMLI read request.
type readCommand struct {
	identity byte
	area     byte
	start    int
	end      int
	points   []*comliPoint
}

func (c *readCommand) quantity() int { return c.end - c.start }

func (c *readCommand) wireAddress() uint16 {
	if c.area == areaBits {
		return uint16(c.start)
	}
	return uint16((c.start/2 + 0x400) * 0x10)
}

func parseAddress(variable *collection.Variable) (*comliPoint, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	address := strings.TrimSpace(variable.Address)
	identityText, rest, ok := strings.Cut(address, "!")
	if !ok {
		return nil, fmt.Errorf("COMLI address %q must be <identity>!<area>.<offset>", address)
	}
	identity, err := strconv.Atoi(identityText)
	if err != nil || identity < 1 || identity > 247 {
		return nil, fmt.Errorf("COMLI identity %q must be between 1 and 247", identityText)
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 2 && len(parts) != 3 {
		return nil, fmt.Errorf("COMLI address %q must be <identity>!<area>.<offset>[.<bit>]", address)
	}
	area, err := strconv.Atoi(parts[0])
	if err != nil || area < 0 || area > 1 {
		return nil, fmt.Errorf("COMLI area %q must be 0 or 1", parts[0])
	}
	offset, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("COMLI offset %q is invalid", parts[1])
	}
	dataType := strings.ToUpper(strings.TrimSpace(variable.DataType))
	point := &comliPoint{
		variable: variable,
		identity: byte(identity),
		area:     byte(area),
		offset:   uint16(offset),
		dataType: dataType,
	}
	switch point.area {
	case areaBits:
		if len(parts) != 2 {
			return nil, fmt.Errorf("COMLI bit area address %q must not include a bit suffix", address)
		}
		if dataType != "BIT" && dataType != "BOOL" {
			return nil, fmt.Errorf("COMLI bit area only supports BIT/BOOL, got %s", variable.DataType)
		}
		if point.offset > maxBitAddress {
			return nil, fmt.Errorf("COMLI bit offset %d exceeds %d", point.offset, maxBitAddress)
		}
		point.bit = uint8(point.offset & 7)
		point.quantity = 1
	case areaRegisters:
		if point.offset > maxRegisterAddress {
			return nil, fmt.Errorf("COMLI register offset %d exceeds %d", point.offset, maxRegisterAddress)
		}
		if len(parts) == 3 {
			bit, err := strconv.Atoi(parts[2])
			if err != nil || bit < 0 || bit > 15 {
				return nil, fmt.Errorf("COMLI register bit %q must be between 0 and 15", parts[2])
			}
			if dataType != "BIT" && dataType != "BOOL" {
				return nil, fmt.Errorf("COMLI bit suffix requires BIT/BOOL, got %s", variable.DataType)
			}
			point.bit = uint8(bit)
			point.quantity = 2
			break
		}
		if dataType == "BIT" || dataType == "BOOL" {
			return nil, fmt.Errorf("COMLI register bit address %q requires a bit suffix", address)
		}
		quantity, err := quantityForDataType(dataType)
		if err != nil {
			return nil, err
		}
		point.quantity = quantity
	default:
		return nil, fmt.Errorf("COMLI area %d is not supported", point.area)
	}
	return point, nil
}

func quantityForDataType(dataType string) (int, error) {
	switch dataType {
	case "INT16", "UINT16":
		return 2, nil
	case "INT32", "UINT32", "FLOAT32", "FLOAT":
		return 4, nil
	case "INT64", "UINT64", "FLOAT64", "DOUBLE":
		return 8, nil
	default:
		return 0, fmt.Errorf("COMLI data type %s is not supported", dataType)
	}
}

func buildReadCommands(variables []*collection.Variable) []*readCommand {
	points := make([]*comliPoint, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || strings.TrimSpace(variable.Address) == "" {
			continue
		}
		point, err := parseAddress(variable)
		if err != nil {
			continue
		}
		points = append(points, point)
	}
	sort.SliceStable(points, func(i, j int) bool {
		if points[i].identity != points[j].identity {
			return points[i].identity < points[j].identity
		}
		if points[i].area != points[j].area {
			return points[i].area < points[j].area
		}
		return points[i].startByte() < points[j].startByte()
	})

	commands := make([]*readCommand, 0)
	for _, point := range points {
		start := point.startByte()
		end := start + point.quantity
		if len(commands) > 0 {
			current := commands[len(commands)-1]
			if current.identity == point.identity && current.area == point.area && start <= current.end {
				mergedEnd := current.end
				if end > mergedEnd {
					mergedEnd = end
				}
				if mergedEnd-current.start <= maxReadSpan {
					current.end = mergedEnd
					current.points = append(current.points, point)
					continue
				}
			}
		}
		commands = append(commands, &readCommand{
			identity: point.identity,
			area:     point.area,
			start:    start,
			end:      end,
			points:   []*comliPoint{point},
		})
	}
	return commands
}

func buildReadRequest(identity byte, address uint16, quantity, stamp byte) []byte {
	frame := make([]byte, 13)
	frame[0] = stx
	putHex8(frame[1:3], identity)
	frame[3] = stamp
	frame[4] = messageReadRegisters
	putHex16(frame[5:9], address)
	putHex8(frame[9:11], quantity)
	frame[11] = etx
	frame[12] = checksum(frame[:12])
	return frame
}

func buildWriteRequest(identity, messageType byte, address uint16, stamp byte, data []byte) []byte {
	frame := make([]byte, 11+len(data)+2)
	frame[0] = stx
	putHex8(frame[1:3], identity)
	frame[3] = stamp
	frame[4] = messageType
	putHex16(frame[5:9], address)
	putHex8(frame[9:11], byte(len(data)))
	copy(frame[11:], data)
	for i := 0; i < len(data); i++ {
		frame[11+i] = mirrorByte(frame[11+i])
	}
	frame[11+len(data)] = etx
	frame[12+len(data)] = checksum(frame[:12+len(data)])
	return frame
}

func parseReadResponse(response []byte, address uint16, quantity, stamp, messageType byte) ([]byte, error) {
	expectedLen := 13 + int(quantity)
	if len(response) != expectedLen {
		return nil, fmt.Errorf("COMLI read response length %d, want %d", len(response), expectedLen)
	}
	if err := validateHeader(response, stamp, messageType, address, quantity); err != nil {
		return nil, err
	}
	if response[11+int(quantity)] != etx {
		return nil, fmt.Errorf("COMLI read response ETX mismatch")
	}
	if checksum(response[:12+int(quantity)]) != response[12+int(quantity)] {
		return nil, fmt.Errorf("COMLI read response BCC mismatch")
	}
	data := append([]byte(nil), response[11:11+int(quantity)]...)
	for i := range data {
		data[i] = mirrorByte(data[i])
	}
	return data, nil
}

func parseWriteAck(response []byte, stamp byte) error {
	if len(response) != 8 {
		return fmt.Errorf("COMLI write ACK length %d, want 8", len(response))
	}
	if response[0] != stx || response[1] != '0' || response[2] != '0' || response[3] != stamp ||
		response[4] != messageAck || response[5] != 0x06 || response[6] != etx {
		return fmt.Errorf("COMLI write ACK header mismatch")
	}
	if checksum(response[:7]) != response[7] {
		return fmt.Errorf("COMLI write ACK BCC mismatch")
	}
	return nil
}

func validateHeader(response []byte, stamp, messageType byte, address uint16, quantity byte) error {
	if response[0] != stx {
		return fmt.Errorf("COMLI response STX mismatch")
	}
	if response[1] != '0' || response[2] != '0' {
		return fmt.Errorf("COMLI response identity %q is not 00", string(response[1:3]))
	}
	if response[3] != stamp {
		return fmt.Errorf("COMLI response stamp %q does not match request %q", string(response[3]), string(stamp))
	}
	if response[4] != messageType {
		return fmt.Errorf("COMLI response message type %q does not match request %q", string(response[4]), string(messageType))
	}
	if !bytes.EqualFold(response[5:9], hex16(address)) {
		return fmt.Errorf("COMLI response address %q does not match %q", string(response[5:9]), string(hex16(address)))
	}
	if !bytes.EqualFold(response[9:11], hex8(quantity)) {
		return fmt.Errorf("COMLI response quantity %q does not match %q", string(response[9:11]), string(hex8(quantity)))
	}
	return nil
}

func checksum(data []byte) byte {
	var value byte
	for _, b := range data {
		value ^= b
	}
	return value
}

func mirrorByte(value byte) byte {
	var mirrored byte
	for i := 0; i < 8; i++ {
		mirrored = (mirrored << 1) | (value & 1)
		value >>= 1
	}
	return mirrored
}

func putHex8(dst []byte, value byte) {
	dst[0] = hexDigits[value>>4]
	dst[1] = hexDigits[value&0x0f]
}

func putHex16(dst []byte, value uint16) {
	putHex8(dst[0:2], byte(value>>8))
	putHex8(dst[2:4], byte(value))
}

func hex8(value byte) []byte {
	out := make([]byte, 2)
	putHex8(out, value)
	return out
}

func hex16(value uint16) []byte {
	out := make([]byte, 4)
	putHex16(out, value)
	return out
}

func decodePointValue(point *comliPoint, command *readCommand, data []byte) (any, error) {
	if point.area == areaBits {
		index := point.startByte() - command.start
		if index < 0 || index >= len(data) {
			return nil, fmt.Errorf("COMLI bit data index out of range")
		}
		return data[index]&(1<<point.bit) != 0, nil
	}
	index := point.startByte() - command.start
	if point.dataType == "BIT" || point.dataType == "BOOL" {
		if index < 0 || index+2 > len(data) {
			return nil, fmt.Errorf("COMLI register bit data is too short")
		}
		value := binary.LittleEndian.Uint16(data[index : index+2])
		return value&(1<<point.bit) != 0, nil
	}
	if index < 0 || index+point.quantity > len(data) {
		return nil, fmt.Errorf("COMLI register data is too short")
	}
	raw := data[index : index+point.quantity]
	switch point.dataType {
	case "INT16":
		return int16(binary.LittleEndian.Uint16(raw)), nil
	case "UINT16":
		return binary.LittleEndian.Uint16(raw), nil
	case "INT32":
		return int32(binary.LittleEndian.Uint32(raw)), nil
	case "UINT32":
		return binary.LittleEndian.Uint32(raw), nil
	case "INT64":
		return int64(binary.LittleEndian.Uint64(raw)), nil
	case "UINT64":
		return binary.LittleEndian.Uint64(raw), nil
	case "FLOAT32", "FLOAT":
		return math.Float32frombits(binary.LittleEndian.Uint32(raw)), nil
	case "FLOAT64", "DOUBLE":
		return math.Float64frombits(binary.LittleEndian.Uint64(raw)), nil
	default:
		return nil, fmt.Errorf("COMLI data type %s is not supported", point.dataType)
	}
}

func encodeWriteData(point *comliPoint, value any) ([]byte, error) {
	if point.dataType == "BIT" || point.dataType == "BOOL" {
		bit, err := valueAsBool(value)
		if err != nil {
			return nil, err
		}
		if point.area == areaBits {
			if bit {
				return []byte{'1'}, nil
			}
			return []byte{'0'}, nil
		}
		return nil, fmt.Errorf("COMLI register bits are read-only")
	}
	data := make([]byte, point.quantity)
	switch point.dataType {
	case "INT16":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint16(data, uint16(int16(number)))
	case "UINT16":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint16(data, uint16(number))
	case "INT32":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(data, uint32(int32(number)))
	case "UINT32":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(data, uint32(number))
	case "INT64":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint64(data, uint64(number))
	case "UINT64":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint64(data, number)
	case "FLOAT32", "FLOAT":
		number, err := valueAsFloat64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(data, math.Float32bits(float32(number)))
	case "FLOAT64", "DOUBLE":
		number, err := valueAsFloat64(value)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint64(data, math.Float64bits(number))
	default:
		return nil, fmt.Errorf("COMLI data type %s is not writable", point.dataType)
	}
	return data, nil
}

func valueAsBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case int16:
		return v != 0, nil
	case uint16:
		return v != 0, nil
	case float32:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("COMLI value %v is not a bit", value)
	}
}

func valueAsInt64(value any) (int64, error) {
	switch v := value.(type) {
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
	default:
		return 0, fmt.Errorf("COMLI value %v is not numeric", value)
	}
}

func valueAsUint64(value any) (uint64, error) {
	switch v := value.(type) {
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
	default:
		return 0, fmt.Errorf("COMLI value %v is not numeric", value)
	}
}

func valueAsFloat64(value any) (float64, error) {
	switch v := value.(type) {
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
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("COMLI value %v is not numeric", value)
	}
}
