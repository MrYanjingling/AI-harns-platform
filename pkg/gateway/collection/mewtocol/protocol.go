package mewtocol

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	cmdReadWord  = "RD"
	cmdReadBit   = "RCC"
	cmdWriteWord = "WD"
	cmdWriteBits = "WCC"
	cmdWriteBit  = "WCS"
)

type pointAddress struct {
	Area        string
	CommandArea string
	Word        int
	Bit         int
	Words       int
	StringLen   int
	IsBit       bool
	IsBitArea   bool
	DataType    string
}

func parseAddress(raw, dataType string, bits uint8) (*pointAddress, error) {
	text := strings.ToUpper(strings.TrimSpace(raw))
	if text == "" {
		return nil, fmt.Errorf("mewtocol address is empty")
	}
	area := ""
	for _, candidate := range []string{"LD", "DT", "FL", "IX", "IY", "ID", "X", "Y", "R", "L", "T", "C"} {
		if strings.HasPrefix(text, candidate) {
			area = candidate
			break
		}
	}
	if area == "" {
		return nil, fmt.Errorf("invalid mewtocol area in %q", raw)
	}
	rest := strings.TrimSpace(text[len(area):])
	wordPart := rest
	suffix := ""
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		wordPart = rest[:dot]
		suffix = rest[dot+1:]
	}
	word, err := strconv.Atoi(wordPart)
	if err != nil || word < 0 || word > 99999 {
		return nil, fmt.Errorf("invalid mewtocol address %q", raw)
	}
	kind := strings.ToLower(strings.TrimSpace(dataType))
	addr := &pointAddress{
		Area:      area,
		Word:      word,
		Bit:       int(bits),
		DataType:  kind,
		IsBitArea: area == "X" || area == "Y" || area == "R" || area == "L" || area == "T" || area == "C",
	}
	addr.CommandArea = area
	switch area {
	case "DT":
		addr.CommandArea = "D"
	case "LD":
		addr.CommandArea = "L"
	case "FL":
		addr.CommandArea = "F"
	}
	switch kind {
	case "bool", "bit":
		addr.IsBit = true
		addr.Words = 1
		if suffix != "" {
			bit, err := strconv.Atoi(suffix)
			if err != nil || bit < 0 || bit > 15 {
				return nil, fmt.Errorf("invalid mewtocol bit address %q", raw)
			}
			addr.Bit = bit
		}
	case "string":
		if suffix == "" {
			return nil, fmt.Errorf("mewtocol string address requires length: %q", raw)
		}
		lengthText := suffix
		if strings.HasSuffix(lengthText, "H") || strings.HasSuffix(lengthText, "L") {
			lengthText = lengthText[:len(lengthText)-1]
		}
		length, err := strconv.Atoi(lengthText)
		if err != nil || length <= 0 || length > 126 || length%2 != 0 {
			return nil, fmt.Errorf("invalid mewtocol string length %q", raw)
		}
		addr.StringLen = length
		addr.Words = (length + 1) / 2
	case "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64":
		if suffix != "" {
			return nil, fmt.Errorf("mewtocol numeric address must not contain a bit suffix: %q", raw)
		}
		words, ok := dataTypeWords(kind)
		if !ok {
			return nil, fmt.Errorf("unsupported mewtocol data type %q", dataType)
		}
		addr.Words = words
	default:
		return nil, fmt.Errorf("unsupported mewtocol data type %q", dataType)
	}
	if addr.IsBit && addr.IsBitArea {
		addr.Word = word*10 + addr.Bit
	}
	if addr.Word > 99999 {
		return nil, fmt.Errorf("mewtocol address is out of range: %q", raw)
	}
	return addr, nil
}

func dataTypeWords(kind string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "int16", "uint16":
		return 1, true
	case "int32", "uint32", "float32":
		return 2, true
	case "int64", "uint64", "float64":
		return 4, true
	default:
		return 0, false
	}
}

func buildReadRequest(unit int, addr *pointAddress) ([]byte, error) {
	if addr == nil {
		return nil, fmt.Errorf("mewtocol address is nil")
	}
	if addr.IsBit {
		if addr.IsBitArea {
			return buildCommand(unit, cmdReadBit, addr.Area, addr.Word, addr.Word, 4, "")
		}
		return buildCommand(unit, cmdReadWord, addr.CommandArea, addr.Word, addr.Word, 5, "")
	}
	if addr.IsBitArea {
		return buildCommand(unit, cmdReadBit, addr.Area, addr.Word, addr.Word+addr.Words-1, 4, "")
	}
	endWidth := 5
	if addr.Area == "IX" || addr.Area == "IY" || addr.Area == "ID" {
		endWidth = 4
	}
	return buildCommand(unit, cmdReadWord, addr.CommandArea, addr.Word, addr.Word+addr.Words-1, endWidth, "")
}

func buildWriteRequest(unit int, addr *pointAddress, data []byte) ([]byte, error) {
	if addr == nil {
		return nil, fmt.Errorf("mewtocol address is nil")
	}
	if addr.IsBit {
		if !addr.IsBitArea {
			return nil, fmt.Errorf("mewtocol bit write is only supported for X/Y/R/L/T/C areas")
		}
		value := "0"
		if len(data) > 0 && data[0] != 0 {
			value = "1"
		}
		return buildCommand(unit, cmdWriteBit, addr.Area, addr.Word, 0, 4, value)
	}
	hexData := strings.ToUpper(hex.EncodeToString(data))
	if addr.IsBitArea {
		return buildCommand(unit, cmdWriteBits, addr.Area, addr.Word, addr.Word+addr.Words-1, 4, hexData)
	}
	endWidth := 5
	if addr.Area == "IX" || addr.Area == "IY" || addr.Area == "ID" {
		endWidth = 4
	}
	return buildCommand(unit, cmdWriteWord, addr.CommandArea, addr.Word, addr.Word+addr.Words-1, endWidth, hexData)
}

func buildCommand(unit int, command, area string, start, end, endWidth int, data string) ([]byte, error) {
	if unit < 1 || unit > 99 {
		return nil, fmt.Errorf("mewtocol unit must be between 1 and 99")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%%%02d#%s%s", unit, command, area)
	switch command {
	case cmdReadWord, cmdWriteWord:
		fmt.Fprintf(&b, "%05d", start)
		if command == cmdWriteWord || endWidth == 5 {
			fmt.Fprintf(&b, "%05d", end)
		} else {
			fmt.Fprintf(&b, "%04d", end)
		}
	case cmdReadBit, cmdWriteBits:
		fmt.Fprintf(&b, "%04d%04d", start, end)
	case cmdWriteBit:
		fmt.Fprintf(&b, "%04d", start)
	default:
		return nil, fmt.Errorf("unsupported mewtocol command %q", command)
	}
	b.WriteString(data)
	content := b.String()
	return []byte(fmt.Sprintf("%s%02X\r", content, checksum([]byte(content)))), nil
}

func checksum(data []byte) byte {
	var value byte
	for _, item := range data {
		value ^= item
	}
	return value
}

func parseResponse(raw []byte) (string, []byte, error) {
	if len(raw) < 8 {
		return "", nil, fmt.Errorf("mewtocol response is too short")
	}
	if raw[len(raw)-1] == '\r' {
		raw = raw[:len(raw)-1]
	}
	if len(raw) < 7 {
		return "", nil, fmt.Errorf("mewtocol response is too short")
	}
	if raw[0] != '%' && raw[0] != '<' {
		return "", nil, fmt.Errorf("invalid mewtocol response start")
	}
	expected := checksum(raw[:len(raw)-2])
	actual, err := strconv.ParseUint(string(raw[len(raw)-2:]), 16, 8)
	if err != nil || byte(actual) != expected {
		return "", nil, fmt.Errorf("mewtocol BCC mismatch")
	}
	if raw[3] == '!' {
		return "", nil, fmt.Errorf("mewtocol device error %s", string(raw[4:6]))
	}
	command := string(raw[3:5])
	if command != "RD" && command != "RC" {
		return "", nil, fmt.Errorf("unexpected mewtocol response command %q", command)
	}
	payload, err := hex.DecodeString(string(raw[5 : len(raw)-2]))
	if err != nil {
		return "", nil, fmt.Errorf("invalid mewtocol response data: %w", err)
	}
	return command, payload, nil
}

func decodeValue(addr *pointAddress, payload []byte) (any, error) {
	if addr == nil {
		return nil, fmt.Errorf("mewtocol address is nil")
	}
	if addr.IsBit {
		if len(payload) < 2 {
			return nil, fmt.Errorf("mewtocol bit response is too short")
		}
		value := binary.BigEndian.Uint16(payload)
		return value&(1<<uint(addr.Bit)) != 0, nil
	}
	need := addr.Words * 2
	if len(payload) < need {
		return nil, fmt.Errorf("mewtocol word response is too short")
	}
	data := payload[:need]
	if addr.DataType == "string" {
		return strings.TrimRight(string(data), "\x00"), nil
	}
	switch addr.DataType {
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
	default:
		return nil, fmt.Errorf("unsupported mewtocol data type %q", addr.DataType)
	}
}

func encodeValue(addr *pointAddress, value any) ([]byte, error) {
	if addr == nil {
		return nil, fmt.Errorf("mewtocol address is nil")
	}
	if addr.IsBit {
		switch v := value.(type) {
		case bool:
			if v {
				return []byte{1}, nil
			}
			return []byte{0}, nil
		default:
			return nil, fmt.Errorf("mewtocol bit value must be bool")
		}
	}
	switch addr.DataType {
	case "string":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("mewtocol string value is invalid")
		}
		data := []byte(text)
		if len(data) != addr.StringLen {
			if len(data) > addr.StringLen {
				data = data[:addr.StringLen]
			} else {
				data = append(data, make([]byte, addr.StringLen-len(data))...)
			}
		}
		return data, nil
	case "int16", "uint16", "int32", "uint32", "int64", "uint64":
		number, ok := numericInt(value)
		if !ok {
			return nil, fmt.Errorf("mewtocol integer value is invalid")
		}
		return encodeInt(addr.DataType, number), nil
	case "float32":
		number, ok := numericFloat(value)
		if !ok {
			return nil, fmt.Errorf("mewtocol float value is invalid")
		}
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, math.Float32bits(float32(number)))
		return data, nil
	case "float64":
		number, ok := numericFloat(value)
		if !ok {
			return nil, fmt.Errorf("mewtocol float value is invalid")
		}
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, math.Float64bits(number))
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported mewtocol data type %q", addr.DataType)
	}
}

func encodeInt(kind string, value int64) []byte {
	switch kind {
	case "int16", "uint16":
		data := make([]byte, 2)
		binary.BigEndian.PutUint16(data, uint16(value))
		return data
	case "int32", "uint32":
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, uint32(value))
		return data
	default:
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(value))
		return data
	}
}

func numericInt(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func numericFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}
