package knd

import (
	"fmt"
	"strconv"
	"strings"
)

type addressKind int

const (
	kindPLC addressKind = iota
	kindSystemInfo
	kindSystemStatus
	kindAlarms
	kindAbsolute
	kindMachine
	kindRelative
	kindCycleTime
	kindWorkCounts
	kindWorkCountGoals
	kindFeedOverride
	kindJogOverride
	kindRapidOverride
	kindHandleOverride
	kindSpindleOverride
	kindSpSpeed
	kindFeedrate
	kindWorkCoors
	kindWorkCoorsCur
	kindVars
	kindProgCur
	kindProgExecStatus
)

type Address struct {
	Kind   addressKind
	Key    string
	Axis   int
	Index  int
	Area   string
	Number int
	Suffix string
	Size   int
	Bit    uint8
}

func ParseAddress(value string, dataType string) (Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Address{}, fmt.Errorf("KND address is empty")
	}
	if address, ok, err := parsePLCAddress(value, dataType); ok || err != nil {
		return address, err
	}
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "systeminfo."):
		return semantic(value, kindSystemInfo, "systemInfo.")
	case strings.HasPrefix(lower, "systemstatus."):
		return semantic(value, kindSystemStatus, "systemStatus.")
	case strings.HasPrefix(lower, "alarms."):
		return semantic(value, kindAlarms, "alarms.")
	case strings.HasPrefix(lower, "absolute."):
		return axisAddress(value, kindAbsolute, "absolute.")
	case strings.HasPrefix(lower, "machine."):
		return axisAddress(value, kindMachine, "machine.")
	case strings.HasPrefix(lower, "relative."):
		return axisAddress(value, kindRelative, "relative.")
	case strings.HasPrefix(lower, "cycletime."):
		return semantic(value, kindCycleTime, "cycleTime.")
	case strings.HasPrefix(lower, "workcounts."):
		return semantic(value, kindWorkCounts, "workCounts.")
	case strings.HasPrefix(lower, "workcountgoals."):
		return semantic(value, kindWorkCountGoals, "workCountGoals.")
	case lower == "feedoverride":
		return Address{Kind: kindFeedOverride}, nil
	case lower == "jogoverride":
		return Address{Kind: kindJogOverride}, nil
	case lower == "rapidoverride":
		return Address{Kind: kindRapidOverride}, nil
	case lower == "handleoverride":
		return Address{Kind: kindHandleOverride}, nil
	case strings.HasPrefix(lower, "spindleoverride."):
		return indexedAddress(value, kindSpindleOverride, "spindleOverride.", 0)
	case strings.HasPrefix(lower, "spspeed."):
		return indexedAddress(value, kindSpSpeed, "spSpeed.", 0)
	case lower == "feedrate":
		return Address{Kind: kindFeedrate}, nil
	case strings.HasPrefix(lower, "g54."), strings.HasPrefix(lower, "g55."), strings.HasPrefix(lower, "g56."), strings.HasPrefix(lower, "g57."), strings.HasPrefix(lower, "g58."), strings.HasPrefix(lower, "g59."):
		return workCoordAddress(value, dataType)
	case lower == "workcoorscur":
		return Address{Kind: kindWorkCoorsCur}, nil
	case strings.HasPrefix(lower, "vars."):
		return indexedAddress(value, kindVars, "vars.", 0)
	case lower == "progcur":
		return Address{Kind: kindProgCur}, nil
	case strings.HasPrefix(lower, "progexecstatus."):
		return semantic(value, kindProgExecStatus, "progExecStatus.")
	default:
		return Address{}, fmt.Errorf("unsupported KND address %q", value)
	}
}

func parsePLCAddress(value string, dataType string) (Address, bool, error) {
	upper := strings.ToUpper(value)
	letter := upper[0]
	if letter < 'A' || letter > 'Z' {
		return Address{}, false, nil
	}
	if strings.HasPrefix(upper, "TL") {
		number, err := strconv.Atoi(upper[2:])
		if err != nil {
			return Address{}, true, fmt.Errorf("invalid KND TL address %q", value)
		}
		size, err := addressSize(dataType, 0)
		if err != nil {
			return Address{}, true, err
		}
		return Address{Kind: kindPLC, Area: "TL", Number: number, Size: size}, true, nil
	}
	if (letter == 'T' || letter == 'C') && len(upper) > 1 && upper[1] >= '0' && upper[1] <= '9' {
		index := strings.Index(upper, ".")
		if index < 2 {
			return Address{}, true, fmt.Errorf("KND %c address requires .conf or .cur", letter)
		}
		number, err := strconv.Atoi(upper[1:index])
		if err != nil {
			return Address{}, true, fmt.Errorf("invalid KND %c address %q", letter, value)
		}
		suffix := strings.ToLower(upper[index+1:])
		if suffix != "conf" && suffix != "cur" {
			return Address{}, true, fmt.Errorf("KND %c address requires .conf or .cur", letter)
		}
		if !strings.EqualFold(dataType, "int32") && !strings.EqualFold(dataType, "uint32") {
			return Address{}, true, fmt.Errorf("KND %c address requires int32 or uint32", letter)
		}
		return Address{Kind: kindPLC, Area: string(letter), Number: number, Suffix: suffix, Size: 4}, true, nil
	}
	if !strings.ContainsRune("XYFGRSKD", rune(letter)) {
		return Address{}, false, nil
	}
	if len(upper) < 2 || upper[1] < '0' || upper[1] > '9' {
		return Address{}, false, nil
	}
	rest := upper[1:]
	numberPart := rest
	suffix := ""
	if dot := strings.Index(rest, "."); dot >= 0 {
		numberPart = rest[:dot]
		suffix = rest[dot+1:]
	}
	number, err := strconv.Atoi(numberPart)
	if err != nil {
		return Address{}, true, fmt.Errorf("invalid KND address %q", value)
	}
	address := Address{Kind: kindPLC, Area: string(letter), Number: number}
	if suffix != "" {
		if strings.EqualFold(dataType, "bit") || strings.EqualFold(dataType, "bool") {
			bit, err := strconv.Atoi(suffix)
			if err != nil || bit < 0 || bit > 7 {
				return Address{}, true, fmt.Errorf("invalid KND bit address %q", value)
			}
			address.Bit = uint8(bit)
		} else if strings.EqualFold(dataType, "string") {
			length, err := strconv.Atoi(suffix)
			if err != nil || length < 1 || length > 128 {
				return Address{}, true, fmt.Errorf("invalid KND string length %q", value)
			}
			address.Size = length
			return address, true, nil
		} else {
			return Address{}, false, nil
		}
	}
	size, err := addressSize(dataType, 0)
	if err != nil {
		return Address{}, true, err
	}
	address.Size = size
	return address, true, nil
}

func semantic(value string, kind addressKind, prefix string) (Address, error) {
	key := strings.TrimSpace(value[len(prefix):])
	if key == "" {
		return Address{}, fmt.Errorf("KND address key is empty")
	}
	return Address{Kind: kind, Key: key}, nil
}

func axisAddress(value string, kind addressKind, prefix string) (Address, error) {
	axis := strings.TrimSpace(value[len(prefix):])
	index := axisIndex(axis)
	if index < 0 {
		return Address{}, fmt.Errorf("invalid KND axis %q", axis)
	}
	return Address{Kind: kind, Axis: index, Key: axis}, nil
}

func indexedAddress(value string, kind addressKind, prefix string, _ int) (Address, error) {
	text := strings.TrimSpace(value[len(prefix):])
	index, err := strconv.Atoi(text)
	if err != nil || index < 0 {
		return Address{}, fmt.Errorf("invalid KND index %q", text)
	}
	return Address{Kind: kind, Index: index}, nil
}

func workCoordAddress(value string, dataType string) (Address, error) {
	index := strings.LastIndex(value, ".")
	if index < 0 {
		return Address{}, fmt.Errorf("invalid KND work coordinate address %q", value)
	}
	base := strings.ToLower(value[:index])
	key := value[index+1:]
	axis := axisIndex(key)
	if axis < 0 {
		return Address{}, fmt.Errorf("invalid KND axis %q", key)
	}
	numbers := map[string]int{"g54": 54, "g55": 55, "g56": 56, "g57": 57, "g58": 58, "g59": 59}
	number, ok := numbers[base]
	if !ok {
		return Address{}, fmt.Errorf("invalid KND work coordinate %q", base)
	}
	return Address{Kind: kindWorkCoors, Number: number, Axis: axis, Key: key}, nil
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

func addressSize(dataType string, suffix int) (int, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bit", "bool", "int8", "uint8":
		return 1, nil
	case "int16", "uint16":
		return 2, nil
	case "int32", "uint32", "float":
		return 4, nil
	case "int64", "uint64", "double":
		return 8, nil
	case "string":
		if suffix <= 0 || suffix > 128 {
			return 0, fmt.Errorf("KND string requires length 1..128")
		}
		return suffix, nil
	default:
		return 0, fmt.Errorf("unsupported KND data type %q", dataType)
	}
}
