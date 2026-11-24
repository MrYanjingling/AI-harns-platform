package srtp

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	memoryIByte  = 0x10
	memoryQByte  = 0x12
	memoryMByte  = 0x16
	memoryTByte  = 0x14
	memorySAByte = 0x18
	memorySBByte = 0x1A
	memorySCByte = 0x1C
	memorySByte  = 0x1E
	memoryGByte  = 0x38
	memoryIBit   = 0x46
	memoryQBit   = 0x48
	memoryMBit   = 0x4C
	memoryTBit   = 0x4A
	memorySABit  = 0x4E
	memorySBBit  = 0x50
	memorySCBit  = 0x52
	memorySBit   = 0x54
	memoryGBit   = 0x56
	memoryAIWord = 0x0A
	memoryAQWord = 0x0C
	memoryRWord  = 0x08
)

type Address struct {
	Memory     uint8
	Start      uint16
	Quantity   uint16
	ByteLength uint16
	Bit        int
	StringLen  int
	IsWord     bool
	DataType   string
}

type prefixInfo struct {
	byteMemory uint8
	bitMemory  uint8
	word       bool
}

var addressPrefixes = []struct {
	prefix string
	info   prefixInfo
}{
	{"SA", prefixInfo{memorySAByte, memorySABit, false}},
	{"SB", prefixInfo{memorySBByte, memorySBBit, false}},
	{"SC", prefixInfo{memorySCByte, memorySCBit, false}},
	{"AI", prefixInfo{memoryAIWord, 0, true}},
	{"AQ", prefixInfo{memoryAQWord, 0, true}},
	{"I", prefixInfo{memoryIByte, memoryIBit, false}},
	{"Q", prefixInfo{memoryQByte, memoryQBit, false}},
	{"M", prefixInfo{memoryMByte, memoryMBit, false}},
	{"T", prefixInfo{memoryTByte, memoryTBit, false}},
	{"S", prefixInfo{memorySByte, memorySBit, false}},
	{"G", prefixInfo{memoryGByte, memoryGBit, false}},
	{"R", prefixInfo{memoryRWord, 0, true}},
}

func ParseAddress(value, dataType string) (Address, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "%") {
		return Address{}, fmt.Errorf("SRTP address must start with %%")
	}
	body := strings.TrimPrefix(value, "%")
	var prefix string
	var info prefixInfo
	for _, candidate := range addressPrefixes {
		if strings.HasPrefix(strings.ToUpper(body), candidate.prefix) {
			prefix = candidate.prefix
			info = candidate.info
			break
		}
	}
	if prefix == "" {
		return Address{}, fmt.Errorf("unsupported SRTP memory prefix in %q", value)
	}
	rest := body[len(prefix):]
	stringLength := 0
	if hash := strings.IndexByte(rest, '#'); hash >= 0 {
		lengthText := strings.TrimSpace(rest[hash+1:])
		parsed, err := strconv.Atoi(lengthText)
		if err != nil || parsed < 1 || parsed > 128 {
			return Address{}, fmt.Errorf("invalid SRTP string length in %q", value)
		}
		stringLength = parsed
		rest = strings.TrimSpace(rest[:hash])
	}
	bit := 0
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		bitText := strings.TrimSpace(rest[dot+1:])
		parsed, err := strconv.Atoi(bitText)
		if err != nil || parsed < 0 || parsed > 15 {
			return Address{}, fmt.Errorf("invalid SRTP bit offset in %q", value)
		}
		bit = parsed
		rest = strings.TrimSpace(rest[:dot])
	}
	base, err := strconv.Atoi(strings.TrimSpace(rest))
	if err != nil || base <= 0 || base > 65535 {
		return Address{}, fmt.Errorf("invalid SRTP address %q", value)
	}
	normalized := normalizeDataType(dataType)
	if normalized == "string" {
		if stringLength == 0 {
			return Address{}, fmt.Errorf("SRTP STRING address requires #length")
		}
	} else if stringLength != 0 {
		return Address{}, fmt.Errorf("SRTP #length is only valid for STRING")
	}
	if bit != 0 && normalized != "bit" {
		return Address{}, fmt.Errorf("SRTP bit offset is only valid for BIT")
	}
	if normalized == "bit" && info.word && bit > 15 {
		return Address{}, fmt.Errorf("SRTP word bit offset must be 0..15")
	}
	if info.word {
		if normalized == "bit" {
			return Address{Memory: info.byteMemory, Start: uint16(base - 1), Quantity: 1, ByteLength: 2, Bit: bit, IsWord: true, DataType: normalized}, nil
		}
		return wordAddress(info.byteMemory, base, normalized, stringLength)
	}
	if normalized == "bit" {
		return Address{Memory: info.bitMemory, Start: uint16(base - 1), Quantity: 1, ByteLength: 1, DataType: normalized}, nil
	}
	return byteAddress(info.byteMemory, base, normalized, stringLength)
}

func wordAddress(memory uint8, base int, dataType string, stringLength int) (Address, error) {
	quantity := 0
	switch dataType {
	case "bool", "bit":
		quantity = 1
	case "int16", "uint16":
		quantity = 1
	case "int32", "uint32", "float32":
		quantity = 2
	case "int64", "uint64", "float64":
		quantity = 4
	case "string":
		quantity = (stringLength + 1) / 2
	default:
		return Address{}, fmt.Errorf("SRTP word memory does not support %s", dataType)
	}
	return Address{Memory: memory, Start: uint16(base - 1), Quantity: uint16(quantity), ByteLength: uint16(quantity * 2), StringLen: stringLength, IsWord: true, DataType: dataType}, nil
}

func byteAddress(memory uint8, base int, dataType string, stringLength int) (Address, error) {
	quantity := 0
	switch dataType {
	case "bool", "bit", "int8", "uint8":
		quantity = 1
	case "int16", "uint16":
		quantity = 2
	case "int32", "uint32", "float32":
		quantity = 4
	case "int64", "uint64", "float64":
		quantity = 8
	case "string":
		quantity = stringLength
	default:
		return Address{}, fmt.Errorf("SRTP byte memory does not support %s", dataType)
	}
	return Address{Memory: memory, Start: uint16(base - 1), Quantity: uint16(quantity), ByteLength: uint16(quantity), StringLen: stringLength, DataType: dataType}, nil
}

func normalizeDataType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bool", "boolean":
		return "bool"
	case "bit":
		return "bit"
	case "int8", "sint":
		return "int8"
	case "uint8", "usint", "byte":
		return "uint8"
	case "int16", "int":
		return "int16"
	case "uint16", "uint":
		return "uint16"
	case "int32", "dint":
		return "int32"
	case "uint32", "udint":
		return "uint32"
	case "int64", "lint":
		return "int64"
	case "uint64", "ulint":
		return "uint64"
	case "float", "float32", "real":
		return "float32"
	case "double", "float64", "lreal":
		return "float64"
	case "string":
		return "string"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}
