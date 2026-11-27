package heidenhain

import (
	"fmt"
	"strconv"
	"strings"
)

type addressKind int

const (
	kindPLC addressKind = iota
	kindRunState
	kindControlMode
	kindProgramMain
	kindProgramCurrent
	kindProgramLine
	kindFeedOverride
	kindSpindleOverride
	kindRapidOverride
	kindToolNumber
	kindToolIndex
	kindToolAxis
	kindToolLength
	kindToolRadius
	kindMachinePosition
	kindParameter
)

type Address struct {
	Kind   addressKind
	Area   byte
	Number int
	Size   int
	Bit    uint8
	Key    string
	Axis   int
}

func ParseAddress(value, dataType string) (Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Address{}, fmt.Errorf("Heidenhain address is empty")
	}
	upper := strings.ToUpper(value)
	if len(upper) >= 2 && upper[1] >= '0' && upper[1] <= '9' && strings.ContainsRune("BWDNUMIOTC", rune(upper[0])) {
		return parsePLC(value, dataType)
	}
	if len(upper) >= 2 && upper[0] == 'S' && upper[1] >= '0' && upper[1] <= '9' {
		return parseStringPLC(value)
	}
	switch strings.ToLower(value) {
	case "runstate":
		return Address{Kind: kindRunState}, nil
	case "controlmode":
		return Address{Kind: kindControlMode}, nil
	case "programmain":
		return Address{Kind: kindProgramMain}, nil
	case "programcurrent":
		return Address{Kind: kindProgramCurrent}, nil
	case "programlineno":
		return Address{Kind: kindProgramLine}, nil
	case "feedoverride":
		return Address{Kind: kindFeedOverride}, nil
	case "spindleoverride":
		return Address{Kind: kindSpindleOverride}, nil
	case "rapidoverride":
		return Address{Kind: kindRapidOverride}, nil
	case "spindletoolnumber":
		return Address{Kind: kindToolNumber}, nil
	case "spindletoolindex":
		return Address{Kind: kindToolIndex}, nil
	case "spindletoolaxis":
		return Address{Kind: kindToolAxis}, nil
	case "spindletoollength":
		return Address{Kind: kindToolLength}, nil
	case "spindletoolradius":
		return Address{Kind: kindToolRadius}, nil
	}
	if strings.HasPrefix(strings.ToLower(value), "machineposition.") {
		axis := strings.TrimSpace(value[len("machinePosition."):])
		index := axisIndex(axis)
		if index < 0 {
			return Address{}, fmt.Errorf("invalid Heidenhain axis %q", axis)
		}
		return Address{Kind: kindMachinePosition, Axis: index, Key: axis}, nil
	}
	if strings.HasPrefix(strings.ToLower(value), "parameter.") {
		return Address{Kind: kindParameter, Key: value[len("parameter."):]}, nil
	}
	return Address{}, fmt.Errorf("unsupported Heidenhain address %q", value)
}

func parsePLC(value, dataType string) (Address, error) {
	upper := strings.ToUpper(value)
	letter := upper[0]
	rest := upper[1:]
	numberPart := rest
	suffix := ""
	if dot := strings.Index(rest, "."); dot >= 0 {
		numberPart = rest[:dot]
		suffix = rest[dot+1:]
	}
	number, err := strconv.Atoi(numberPart)
	if err != nil {
		return Address{}, fmt.Errorf("invalid Heidenhain address %q", value)
	}
	size, err := valueSize(dataType)
	if err != nil {
		return Address{}, err
	}
	address := Address{Kind: kindPLC, Area: letter, Number: number, Size: size}
	if suffix != "" {
		if !strings.EqualFold(dataType, "bit") && !strings.EqualFold(dataType, "bool") {
			return Address{}, fmt.Errorf("Heidenhain bit suffix requires BIT/BOOL")
		}
		bit, err := strconv.Atoi(suffix)
		if err != nil || bit < 0 || bit > 7 {
			return Address{}, fmt.Errorf("invalid Heidenhain bit address %q", value)
		}
		address.Bit = uint8(bit)
	}
	switch letter {
	case 'D':
		if number%4 != 0 {
			return Address{}, fmt.Errorf("Heidenhain DWORD address must be aligned to 4 bytes")
		}
	case 'W', 'N', 'U':
		if number%2 != 0 {
			return Address{}, fmt.Errorf("Heidenhain WORD address must be aligned to 2 bytes")
		}
	}
	return address, nil
}

func parseStringPLC(value string) (Address, error) {
	upper := strings.ToUpper(value)
	parts := strings.SplitN(upper[1:], ".", 2)
	if len(parts) != 2 {
		return Address{}, fmt.Errorf("Heidenhain string address requires S<n>.<len>")
	}
	number, err := strconv.Atoi(parts[0])
	if err != nil {
		return Address{}, fmt.Errorf("invalid Heidenhain string address %q", value)
	}
	length, err := strconv.Atoi(parts[1])
	if err != nil || length < 1 || length > 128 {
		return Address{}, fmt.Errorf("Heidenhain string length must be 1..128")
	}
	return Address{Kind: kindPLC, Area: 'S', Number: number, Size: length}, nil
}

func valueSize(dataType string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bit", "bool", "int8", "uint8":
		return 1, nil
	case "int16", "uint16":
		return 2, nil
	case "int32", "uint32", "float":
		return 4, nil
	case "int64", "uint64", "double":
		return 8, nil
	default:
		return 0, fmt.Errorf("unsupported Heidenhain data type %q", dataType)
	}
}

func axisIndex(value string) int {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "X":
		return 0
	case "Y":
		return 1
	case "Z":
		return 2
	case "A":
		return 3
	case "B":
		return 4
	case "C":
		return 5
	default:
		return -1
	}
}
