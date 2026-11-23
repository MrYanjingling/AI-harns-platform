package mpi

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	areaS     = 4
	areaSM    = 5
	areaAI    = 6
	areaAQ    = 7
	areaC     = 28
	areaT     = 29
	areaIECC  = 30
	areaIECT  = 31
	areaIECHC = 32
	areaP     = 128
	areaI     = 129
	areaQ     = 130
	areaM     = 131
	areaDB    = 132
	areaDI    = 133

	typeBool  = 1
	typeByte  = 2
	typeWord  = 4
	typeDWord = 6
)

type parsedAddress struct {
	Area   byte
	Start  uint16
	DB     uint16
	Length uint16
	Bit    uint8
	Type   byte
}

// Address is the wire address of a Siemens S7/MPI/PPI item.
type Address = parsedAddress

// ParseAddress parses an S7-style address for MPI/PPI.
func ParseAddress(value string, dataType string) (Address, error) {
	return parseAddress(value, dataType)
}

func parseAddress(value string, dataType string) (parsedAddress, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return parsedAddress{}, fmt.Errorf("mpi address is empty")
	}
	var result parsedAddress
	upper := strings.ToUpper(value)
	switch {
	case strings.HasPrefix(upper, "DB"):
		if err := parseDBAddress(value, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "SM"):
		if err := parseNumberArea(value[2:], areaSM, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "IECHC"):
		if err := parseNumberArea(value[5:], areaIECHC, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "IECC"):
		if err := parseNumberArea(value[4:], areaIECC, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "IECT"):
		if err := parseNumberArea(value[4:], areaIECT, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "DI"):
		if err := parseNumberArea(value[2:], areaDI, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "AIW"):
		if err := parseNumberArea(value[3:], areaAI, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "AQW"):
		if err := parseNumberArea(value[3:], areaAQ, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "AI"):
		if err := parseNumberArea(value[2:], areaAI, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "AQ"):
		if err := parseNumberArea(value[2:], areaAQ, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "S"):
		if err := parseNumberArea(value[1:], areaS, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "C"):
		if err := parseNumberArea(value[1:], areaC, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "T"):
		if err := parseNumberArea(value[1:], areaT, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "P"):
		if err := parseNumberArea(value[1:], areaP, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "I"):
		if err := parseNumberArea(value[1:], areaI, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "Q"):
		if err := parseNumberArea(value[1:], areaQ, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "M"):
		if err := parseNumberArea(value[1:], areaM, &result); err != nil {
			return result, err
		}
	case strings.HasPrefix(upper, "V"):
		if err := parseVAddress(value, &result); err != nil {
			return result, err
		}
	default:
		return result, fmt.Errorf("unsupported mpi address %q", value)
	}
	length, err := valueLength(dataType, result.Length)
	if err != nil {
		return result, err
	}
	result.Length = length
	return result, nil
}

func parseDBAddress(value string, result *parsedAddress) error {
	index := strings.Index(value, ".")
	if index < 3 {
		return fmt.Errorf("invalid mpi DB address %q", value)
	}
	block, err := strconv.ParseUint(value[2:index], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid mpi DB block in %q", value)
	}
	result.Area = areaDB
	result.DB = uint16(block)
	part := strings.ToUpper(value[index+1:])
	switch {
	case strings.HasPrefix(part, "DBX"):
		return parseBitPart(value[index+4:], result)
	case strings.HasPrefix(part, "DBB"):
		return parseBytePart(value[index+4:], typeByte, result)
	case strings.HasPrefix(part, "DBW"):
		return parseBytePart(value[index+4:], typeWord, result)
	case strings.HasPrefix(part, "DBD"):
		return parseBytePart(value[index+4:], typeDWord, result)
	default:
		return fmt.Errorf("invalid mpi DB address %q", value)
	}
}

func parseNumberArea(value string, area byte, result *parsedAddress) error {
	result.Area = area
	if index := strings.LastIndex(value, "."); index >= 0 {
		return parseBitPart(value, result)
	}
	number, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid mpi address %q", value)
	}
	result.Start = uint16(number)
	result.Type = typeByte
	return nil
}

func parseBytePart(value string, typ byte, result *parsedAddress) error {
	addressPart := value
	if index := strings.LastIndex(value, "."); index >= 0 {
		addressPart = value[:index]
		length, err := strconv.ParseUint(value[index+1:], 10, 16)
		if err != nil || length == 0 {
			return fmt.Errorf("invalid mpi length %q", value)
		}
		result.Length = uint16(length)
	}
	number, err := strconv.ParseUint(addressPart, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid mpi address %q", value)
	}
	result.Start = uint16(number)
	result.Type = typ
	return nil
}

func parseBitPart(value string, result *parsedAddress) error {
	index := strings.LastIndex(value, ".")
	if index < 0 {
		return fmt.Errorf("invalid mpi bit address %q", value)
	}
	number, err := strconv.ParseUint(value[:index], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid mpi address %q", value)
	}
	bit, err := strconv.ParseUint(value[index+1:], 10, 8)
	if err != nil || bit > 7 {
		return fmt.Errorf("invalid mpi bit %q", value)
	}
	result.Start = uint16(number)
	result.Bit = uint8(bit)
	result.Type = typeBool
	return nil
}

func parseVAddress(value string, result *parsedAddress) error {
	result.Area = areaDB
	result.DB = 1
	rest := strings.ToUpper(value[1:])
	switch {
	case strings.HasPrefix(rest, "B"):
		return parseBytePart(value[2:], typeByte, result)
	case strings.HasPrefix(rest, "W"):
		return parseBytePart(value[2:], typeWord, result)
	case strings.HasPrefix(rest, "D"):
		return parseBytePart(value[2:], typeDWord, result)
	default:
		return parseBitPart(value[1:], result)
	}
}

func valueLength(dataType string, suffix uint16) (uint16, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "bit":
		return 1, nil
	case "int16", "uint16":
		return 2, nil
	case "int32", "uint32", "float32":
		return 4, nil
	case "int64", "uint64", "float64":
		return 8, nil
	case "string", "byte_array":
		if suffix == 0 {
			return 0, fmt.Errorf("string/byte_array requires a length suffix")
		}
		return suffix, nil
	default:
		return 0, fmt.Errorf("unsupported mpi data type %q", dataType)
	}
}
