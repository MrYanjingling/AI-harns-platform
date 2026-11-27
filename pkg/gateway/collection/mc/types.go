package mc

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

type Family string

const (
	Family1E Family = "1e"
	Family3E Family = "3e"
	Family4E Family = "4e"
)

type Info struct {
	DriverType  string
	DriverModel string
	Family      Family
	ASCII       bool
}

var (
	InfoA1E            = Info{DriverType: "Mitsubishi", DriverModel: "MC-1E", Family: Family1E}
	Info3E             = Info{DriverType: "Mitsubishi", DriverModel: "MC-3E", Family: Family3E}
	Info3EAscii        = Info{DriverType: "Mitsubishi", DriverModel: "MC-3E ASCII", Family: Family3E, ASCII: true}
	Info4E             = Info{DriverType: "Mitsubishi", DriverModel: "MC-4E", Family: Family4E}
	InfoPanasonicMC3E  = Info{DriverType: "Panasonic", DriverModel: "Panasonic MC-3E", Family: Family3E}
	InfoKeyenceMC3EAsc = Info{DriverType: "Keyence", DriverModel: "KV MC-3E ASCII", Family: Family3E, ASCII: true}
)

const (
	DefaultPort       = 2000
	DefaultSrcPort    = 52001
	DefaultTimeoutMs  = 15000
	DefaultIntervalMs = 20
	DefaultRetry      = 0
)

type ProtocolConfig struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	TransportMode int    `json:"transport_mode"`
	SrcPort       int    `json:"src-port"`
	TimeoutMs     int    `json:"timeout"`
	IntervalMs    int    `json:"interval"`
	Retry         int    `json:"retry"`
}

func (c *ProtocolConfig) Normalize() {
	c.Host = strings.TrimSpace(c.Host)
	if c.Port <= 0 {
		c.Port = DefaultPort
	}
	if c.TransportMode < 0 || c.TransportMode > 1 {
		c.TransportMode = 0
	}
	if c.SrcPort <= 0 {
		c.SrcPort = DefaultSrcPort
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = DefaultTimeoutMs
	}
	if c.IntervalMs <= 0 {
		c.IntervalMs = DefaultIntervalMs
	}
	if c.Retry < 0 {
		c.Retry = 0
	}
	if c.Retry > 10 {
		c.Retry = 10
	}
}

type CreateReq struct {
	Name             string                 `json:"name"`
	CollectorCycleMs uint64                 `json:"collectorCycleMs"`
	Variables        []*collection.Variable `json:"variables"`
	ProtocolConfig   ProtocolConfig         `json:"protocolConfig"`
}

type UpdateReq struct {
	Name             string         `json:"name"`
	CollectorCycleMs uint64         `json:"collectorCycleMs"`
	ProtocolConfig   ProtocolConfig `json:"protocolConfig"`
}

type areaDef struct {
	code byte
	word bool
	hex  bool
}

var areaDefs = map[string]areaDef{
	"DX":  {0xA2, false, true},
	"DY":  {0xA3, false, true},
	"STN": {0xFF, true, false},
	"STS": {0xFF, false, false},
	"STC": {0xFF, false, false},
	"ZR":  {0xB0, true, false},
	"SB":  {0xA1, false, true},
	"SW":  {0xB5, true, true},
	"SM":  {0x91, false, false},
	"SD":  {0xA9, true, false},
	"TS":  {0xC1, false, false},
	"TC":  {0xC0, false, false},
	"TN":  {0xC2, true, false},
	"SS":  {0xC7, false, false},
	"SC":  {0xC6, false, false},
	"CS":  {0xC4, false, false},
	"CC":  {0xC3, false, false},
	"CN":  {0xC5, true, false},
	"SN":  {0xC8, true, false},
	"X":   {0x9C, false, true},
	"Y":   {0x9D, false, true},
	"M":   {0x90, false, false},
	"L":   {0x92, false, false},
	"F":   {0x93, false, false},
	"V":   {0x94, false, false},
	"B":   {0xA0, false, true},
	"S":   {0x98, false, false},
	"D":   {0xA8, true, false},
	"W":   {0xB4, true, true},
	"R":   {0xAF, true, false},
	"Z":   {0xCC, true, false},
}

var areaPrefixes = func() []string {
	prefixes := make([]string, 0, len(areaDefs))
	for prefix := range areaDefs {
		prefixes = append(prefixes, prefix)
	}
	sort.Slice(prefixes, func(i, j int) bool { return len(prefixes[i]) > len(prefixes[j]) })
	return prefixes
}()

type point struct {
	variable *collection.Variable
	area     byte
	device   string
	number   uint32
	bit      int
	word     bool
	quantity int
	dataType string
}

func parseAddress(variable *collection.Variable) (*point, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	address := strings.ToUpper(strings.TrimSpace(variable.Address))
	prefix := ""
	for _, candidate := range areaPrefixes {
		if strings.HasPrefix(address, candidate) {
			prefix = candidate
			break
		}
	}
	if prefix == "" {
		return nil, fmt.Errorf("MC address %q has an unknown device prefix", variable.Address)
	}
	def := areaDefs[prefix]
	if def.code == 0xFF {
		return nil, fmt.Errorf("MC device %q is not supported", prefix)
	}
	rest := address[len(prefix):]
	numberText := rest
	bit := -1
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		numberText = rest[:dot]
		bitText := rest[dot+1:]
		value, err := strconv.Atoi(bitText)
		if err != nil || value < 0 || value > 15 {
			return nil, fmt.Errorf("MC bit %q must be between 0 and 15", bitText)
		}
		bit = value
	}
	base := 10
	if def.hex {
		base = 16
	}
	number, err := strconv.ParseUint(numberText, base, 32)
	if err != nil {
		return nil, fmt.Errorf("MC device number %q is invalid", numberText)
	}
	if number > 0xFFFFFF {
		return nil, fmt.Errorf("MC device number %d exceeds 0xFFFFFF", number)
	}
	dataType := strings.ToUpper(strings.TrimSpace(variable.DataType))
	item := &point{
		variable: variable,
		area:     def.code,
		device:   prefix,
		number:   uint32(number),
		bit:      bit,
		word:     def.word,
		dataType: dataType,
	}
	if def.word {
		if bit >= 0 {
			if dataType != "BIT" && dataType != "BOOL" {
				return nil, fmt.Errorf("MC bit suffix requires BIT/BOOL, got %s", variable.DataType)
			}
			item.quantity = 1
			return item, nil
		}
		words, err := wordsForType(dataType)
		if err != nil {
			return nil, err
		}
		item.quantity = words
		return item, nil
	}
	if bit >= 0 {
		return nil, fmt.Errorf("MC bit device %q must not include a bit suffix", variable.Address)
	}
	if dataType != "BIT" && dataType != "BOOL" {
		return nil, fmt.Errorf("MC bit device %q only supports BIT/BOOL, got %s", variable.Address, variable.DataType)
	}
	item.quantity = 1
	return item, nil
}

func wordsForType(dataType string) (int, error) {
	switch dataType {
	case "INT16", "UINT16":
		return 1, nil
	case "INT32", "UINT32", "FLOAT32", "FLOAT":
		return 2, nil
	case "INT64", "UINT64", "FLOAT64", "DOUBLE":
		return 4, nil
	default:
		return 0, fmt.Errorf("MC data type %s is not supported", dataType)
	}
}

type command struct {
	area   byte
	word   bool
	start  uint32
	count  int
	points []*point
}

func buildCommands(variables []*collection.Variable) []*command {
	return buildCommandsForFamily(Family3E, variables)
}

func buildCommandsForFamily(family Family, variables []*collection.Variable) []*command {
	points := make([]*point, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || strings.TrimSpace(variable.Address) == "" {
			continue
		}
		item, err := parseAddress(variable)
		if err != nil {
			continue
		}
		points = append(points, item)
	}
	sort.SliceStable(points, func(i, j int) bool {
		if points[i].area != points[j].area {
			return points[i].area < points[j].area
		}
		if points[i].word != points[j].word {
			return points[i].word
		}
		return points[i].number < points[j].number
	})
	maxWords := 960
	maxBits := 7168
	if family == Family1E {
		maxWords = 64
		maxBits = 200
	}
	commands := make([]*command, 0)
	for _, item := range points {
		if len(commands) > 0 {
			current := commands[len(commands)-1]
			if current.area == item.area && current.word == item.word && item.number <= current.start+uint32(current.count) {
				end := item.number + uint32(item.quantity)
				currentEnd := current.start + uint32(current.count)
				if end < currentEnd {
					end = currentEnd
				}
				max := maxWords
				if !current.word {
					max = maxBits
				}
				if int(end-current.start) <= max {
					current.count = int(end - current.start)
					current.points = append(current.points, item)
					continue
				}
			}
		}
		commands = append(commands, &command{
			area:   item.area,
			word:   item.word,
			start:  item.number,
			count:  item.quantity,
			points: []*point{item},
		})
	}
	return commands
}

func decodePoint(item *point, cmd *command, data []byte) (any, error) {
	if !cmd.word {
		index := int(item.number - cmd.start)
		byteIndex := index / 2
		if byteIndex < 0 || byteIndex >= len(data) {
			return nil, fmt.Errorf("MC bit data index out of range")
		}
		var nibble byte
		if index%2 == 0 {
			nibble = (data[byteIndex] >> 4) & 0x0F
		} else {
			nibble = data[byteIndex] & 0x0F
		}
		return nibble != 0, nil
	}
	wordIndex := int(item.number - cmd.start)
	offset := wordIndex * 2
	if offset < 0 || offset+2 > len(data) {
		return nil, fmt.Errorf("MC word data index out of range")
	}
	if item.dataType == "BIT" || item.dataType == "BOOL" {
		value := uint16(data[offset]) | uint16(data[offset+1])<<8
		return value&(1<<item.bit) != 0, nil
	}
	raw := data[offset : offset+item.quantity*2]
	return decodeWords(raw, item.dataType)
}

func encodePoint(item *point, value any) ([]byte, error) {
	if item.dataType == "BIT" || item.dataType == "BOOL" {
		if !item.word {
			bit, err := valueAsBool(value)
			if err != nil {
				return nil, err
			}
			if bit {
				return []byte{0x10}, nil
			}
			return []byte{0x00}, nil
		}
		return nil, fmt.Errorf("MC word bit writes require read-modify-write")
	}
	data := make([]byte, item.quantity*2)
	switch item.dataType {
	case "INT16":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		put16(data, uint16(int16(number)))
	case "UINT16":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		put16(data, uint16(number))
	case "INT32":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		put32(data, uint32(int32(number)))
	case "UINT32":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		put32(data, uint32(number))
	case "INT64":
		number, err := valueAsInt64(value)
		if err != nil {
			return nil, err
		}
		put64(data, uint64(number))
	case "UINT64":
		number, err := valueAsUint64(value)
		if err != nil {
			return nil, err
		}
		put64(data, number)
	case "FLOAT32", "FLOAT":
		number, err := valueAsFloat64(value)
		if err != nil {
			return nil, err
		}
		put32(data, math.Float32bits(float32(number)))
	case "FLOAT64", "DOUBLE":
		number, err := valueAsFloat64(value)
		if err != nil {
			return nil, err
		}
		put64(data, math.Float64bits(number))
	default:
		return nil, fmt.Errorf("MC data type %s is not writable", item.dataType)
	}
	return data, nil
}

func decodeWords(data []byte, dataType string) (any, error) {
	switch dataType {
	case "INT16":
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "UINT16":
		return binary.LittleEndian.Uint16(data), nil
	case "INT32":
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "UINT32":
		return binary.LittleEndian.Uint32(data), nil
	case "INT64":
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "UINT64":
		return binary.LittleEndian.Uint64(data), nil
	case "FLOAT32", "FLOAT":
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "FLOAT64", "DOUBLE":
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	default:
		return nil, fmt.Errorf("MC data type %s is not supported", dataType)
	}
}

func valueAsBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case int16:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int64:
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
	default:
		return false, fmt.Errorf("MC value %v is not a bit", value)
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
		return 0, fmt.Errorf("MC value %v is not numeric", value)
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
		return 0, fmt.Errorf("MC value %v is not numeric", value)
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
		return 0, fmt.Errorf("MC value %v is not numeric", value)
	}
}

func put16(data []byte, value uint16) {
	binary.LittleEndian.PutUint16(data, value)
}

func put32(data []byte, value uint32) {
	binary.LittleEndian.PutUint32(data, value)
}

func put64(data []byte, value uint64) {
	binary.LittleEndian.PutUint64(data, value)
}
