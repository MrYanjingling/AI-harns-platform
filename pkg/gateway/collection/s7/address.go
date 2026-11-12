package s7

import (
	"fmt"
	"strconv"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

type storeArea int8

const (
	areaI storeArea = iota
	areaQ
	areaM
	areaDB
)

func (a storeArea) code() uint8 {
	switch a {
	case areaI:
		return 129
	case areaQ:
		return 130
	case areaM:
		return 131
	default:
		return 132
	}
}

func (a storeArea) transportSize() uint8 {
	return 2
}

type parsedAddress struct {
	area      storeArea
	blockSize uint
	address   uint32
	bit       uint8
	length    uint16
}

func parseAddress(value string) (parsedAddress, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return parsedAddress{}, fmt.Errorf("s7 address is empty")
	}
	var result parsedAddress
	switch {
	case strings.HasPrefix(value, "I"):
		result.area = areaI
		return parseAreaAddress(value, result)
	case strings.HasPrefix(value, "Q"):
		result.area = areaQ
		return parseAreaAddress(value, result)
	case strings.HasPrefix(value, "M"):
		result.area = areaM
		return parseAreaAddress(value, result)
	case strings.HasPrefix(value, "V"):
		return parseVAddress(value)
	case strings.HasPrefix(value, "DB"):
		return parseDBAddress(value)
	default:
		return parsedAddress{}, fmt.Errorf("unsupported s7 address %q", value)
	}
}

func parseAreaAddress(value string, result parsedAddress) (parsedAddress, error) {
	addressPart := value[1:]
	if !shortening(addressPart) {
		addressPart = addressPart[1:]
	}
	return parseByteAndBit(addressPart, result)
}

func parseVAddress(value string) (parsedAddress, error) {
	rest := strings.TrimPrefix(value, "V")
	if rest == "" {
		return parsedAddress{}, fmt.Errorf("invalid s7 V address %q", value)
	}
	result := parsedAddress{area: areaDB, blockSize: 1}
	if strings.Contains(rest, ".") {
		return parseByteAndBit(rest, result)
	}
	switch rest[0] {
	case 'B', 'W', 'D':
		return parseByteLength(rest[1:], result)
	default:
		return parsedAddress{}, fmt.Errorf("invalid s7 V address %q", value)
	}
}

func parseDBAddress(value string) (parsedAddress, error) {
	index := strings.Index(value, ".")
	if index < 3 {
		return parsedAddress{}, fmt.Errorf("invalid s7 DB address %q", value)
	}
	block, err := strconv.Atoi(value[2:index])
	if err != nil || block < 0 {
		return parsedAddress{}, fmt.Errorf("invalid s7 DB block in %q", value)
	}
	addressPart := value[index+1:]
	result := parsedAddress{area: areaDB, blockSize: uint(block)}
	switch {
	case strings.HasPrefix(addressPart, "DBX"):
		return parseByteAndBit(strings.TrimPrefix(addressPart, "DBX"), result)
	case strings.HasPrefix(addressPart, "DBB"):
		return parseByteLength(strings.TrimPrefix(addressPart, "DBB"), result)
	case strings.HasPrefix(addressPart, "DBW"):
		return parseByteLength(strings.TrimPrefix(addressPart, "DBW"), result)
	case strings.HasPrefix(addressPart, "DBD"):
		return parseByteLength(strings.TrimPrefix(addressPart, "DBD"), result)
	default:
		return parseByteAndBit(addressPart, result)
	}
}

func parseByteLength(value string, result parsedAddress) (parsedAddress, error) {
	index := strings.LastIndex(value, ".")
	addressPart := value
	if index >= 0 {
		addressPart = value[:index]
		length, err := strconv.ParseUint(value[index+1:], 10, 16)
		if err != nil || length == 0 {
			return parsedAddress{}, fmt.Errorf("invalid s7 byte length %q", value)
		}
		result.length = uint16(length)
	}
	address, err := strconv.ParseUint(addressPart, 10, 32)
	if err != nil {
		return parsedAddress{}, fmt.Errorf("invalid s7 byte address %q", value)
	}
	result.address = uint32(address)
	return result, nil
}
func parseByteAndBit(value string, result parsedAddress) (parsedAddress, error) {
	index := strings.LastIndex(value, ".")
	addressPart := value
	if index >= 0 {
		addressPart = value[:index]
		bit, err := strconv.Atoi(value[index+1:])
		if err != nil || bit < 0 || bit > 7 {
			return parsedAddress{}, fmt.Errorf("invalid s7 bit address %q", value)
		}
		result.bit = uint8(bit)
	}
	address, err := strconv.ParseUint(addressPart, 10, 32)
	if err != nil {
		return parsedAddress{}, fmt.Errorf("invalid s7 byte address %q", value)
	}
	result.address = uint32(address)
	return result, nil
}

func shortening(value string) bool {
	if strings.HasPrefix(value, "DBX") || strings.HasPrefix(value, "DBB") || strings.HasPrefix(value, "DBW") || strings.HasPrefix(value, "DBD") || strings.HasPrefix(value, "DBS") {
		return false
	}
	return !(strings.HasPrefix(value, "X") || strings.HasPrefix(value, "B") || strings.HasPrefix(value, "W") || strings.HasPrefix(value, "D") || strings.HasPrefix(value, "S"))
}

func dataRequestLength(dataType string, byteLength uint16) uint16 {
	switch strings.ToLower(dataType) {
	case "string", "byte_array":
		return (byteLength + 1) / 2
	case "bool", "bit":
		return 1
	case "uint16", "int16":
		return 2
	case "int32", "float32":
		return 4
	case "int64", "float64":
		return 8
	default:
		return 1
	}
}

func dataResponseLength(area storeArea, dataType string, byteLength uint16) uint16 {
	switch strings.ToLower(dataType) {
	case "string", "byte_array":
		return byteLength
	case "bool":
		if area == areaDB {
			return 2
		}
		return 1
	case "uint16", "int16":
		return 2
	case "int32", "float32":
		return 4
	case "int64", "float64":
		return 8
	default:
		return 2
	}
}

func dataTypeBitLength(dataType string) uint16 {
	switch strings.ToLower(dataType) {
	case "bool":
		return 1
	case "uint16", "int16":
		return 16
	case "int32", "float32":
		return 32
	case "int64", "float64":
		return 64
	default:
		return 16
	}
}

func sortVariables(values []*collection.Variable) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0; j-- {
			left, _ := parseAddress(values[j-1].Address)
			right, _ := parseAddress(values[j].Address)
			if compareParsed(left, right) <= 0 {
				break
			}
			values[j-1], values[j] = values[j], values[j-1]
		}
	}
}

func compareParsed(left, right parsedAddress) int {
	if left.area != right.area {
		if left.area < right.area {
			return -1
		}
		return 1
	}
	if left.blockSize != right.blockSize {
		if left.blockSize < right.blockSize {
			return -1
		}
		return 1
	}
	if left.address != right.address {
		if left.address < right.address {
			return -1
		}
		return 1
	}
	if left.bit < right.bit {
		return -1
	}
	if left.bit > right.bit {
		return 1
	}
	return 0
}
