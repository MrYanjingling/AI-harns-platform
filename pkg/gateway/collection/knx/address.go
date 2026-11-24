package knx

import (
	"fmt"
	"strconv"
	"strings"
)

type Address struct {
	Group      uint16
	Individual uint16
	NBit       uint8
}

func ParseAddress(value string, dataType string) (Address, error) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	if len(parts) < 2 || len(parts) > 3 {
		return Address{}, fmt.Errorf("KNX address must be group,individual[,nbit]")
	}
	group, err := parseGroup(parts[0])
	if err != nil {
		return Address{}, err
	}
	individual, err := parseIndividual(parts[1])
	if err != nil {
		return Address{}, err
	}
	result := Address{Group: group, Individual: individual}
	if len(parts) == 3 {
		nbit, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil || (nbit != 2 && nbit != 4) {
			return Address{}, fmt.Errorf("KNX nbit must be 2 or 4")
		}
		if !strings.EqualFold(dataType, "uint8") {
			return Address{}, fmt.Errorf("KNX nbit is only valid for uint8")
		}
		result.NBit = uint8(nbit)
	}
	return result, nil
}

func parseGroup(value string) (uint16, error) {
	values, err := parseParts(value, "/")
	if err != nil {
		return 0, err
	}
	if values[0] < 0 || values[0] > 31 || values[1] < 0 || values[1] > 7 || values[2] < 0 || values[2] > 255 {
		return 0, fmt.Errorf("KNX group address %q is out of range", value)
	}
	return uint16(values[0]<<11 | values[1]<<8 | values[2]), nil
}

func parseIndividual(value string) (uint16, error) {
	values, err := parseParts(value, ".")
	if err != nil {
		return 0, err
	}
	if values[0] < 0 || values[0] > 15 || values[1] < 0 || values[1] > 15 || values[2] < 0 || values[2] > 255 {
		return 0, fmt.Errorf("KNX individual address %q is out of range", value)
	}
	return uint16(values[0]<<12 | values[1]<<8 | values[2]), nil
}

func parseParts(value, separator string) ([3]int, error) {
	var values [3]int
	parts := strings.Split(strings.TrimSpace(value), separator)
	if len(parts) != 3 {
		return values, fmt.Errorf("invalid KNX address %q", value)
	}
	for index, part := range parts {
		number, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return values, fmt.Errorf("invalid KNX address %q", value)
		}
		values[index] = number
	}
	return values, nil
}
