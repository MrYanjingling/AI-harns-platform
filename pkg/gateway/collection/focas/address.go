package focas

import (
	"fmt"
	"strconv"
	"strings"
)

type Address struct {
	Kind   string
	Area   byte
	Number int
	Bit    uint8
	Axis   int
	Index  int
	Key    string
}

func ParseAddress(value, dataType string) (Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Address{}, fmt.Errorf("FOCAS address is empty")
	}
	upper := strings.ToUpper(value)
	if len(upper) >= 2 && upper[1] >= '0' && upper[1] <= '9' && strings.ContainsRune("ACDEFGKMNRTXY", rune(upper[0])) {
		return parsePMC(value, dataType)
	}
	lower := strings.ToLower(value)
	switch {
	case lower == "actf", lower == "acts", lower == "runstate", lower == "controlmode", lower == "programmain", lower == "programmainfile", lower == "alarm":
		return Address{Kind: lower}, nil
	case strings.HasPrefix(lower, "absolute."), strings.HasPrefix(lower, "absolute2."), strings.HasPrefix(lower, "relative."), strings.HasPrefix(lower, "relative2."), strings.HasPrefix(lower, "machine."), strings.HasPrefix(lower, "distance."):
		return axisAddress(value, lower)
	case strings.HasPrefix(lower, "acts2."):
		return prefixIndexAddress(value, "acts2")
	case strings.HasPrefix(lower, "spload."), strings.HasPrefix(lower, "spmaxrpm."), strings.HasPrefix(lower, "spgear."):
		return prefixIndexAddress(value, strings.SplitN(lower, ".", 2)[0])
	case strings.HasPrefix(lower, "spcss_"):
		return Address{Kind: "spcss", Key: lower}, nil
	case strings.HasPrefix(lower, "proginfo."):
		return prefixIndexAddress(value, "proginfo")
	case strings.HasPrefix(lower, "param."), strings.HasPrefix(lower, "alarmmsg2."), strings.HasPrefix(lower, "tooloffset."), strings.HasPrefix(lower, "zofs."):
		return Address{Kind: strings.SplitN(lower, ".", 2)[0], Key: lower}, nil
	case strings.HasPrefix(lower, "alarmmsg."), strings.HasPrefix(lower, "tooloffsetinfo."), strings.HasPrefix(lower, "wkcdsfms."), strings.HasPrefix(lower, "wkcdshft."):
		return prefixIndexAddress(value, strings.SplitN(lower, ".", 2)[0])
	case strings.HasPrefix(lower, "pmcinfo"), strings.HasPrefix(lower, "pmctitle"), strings.HasPrefix(lower, "pmcnum"), strings.HasPrefix(lower, "pmcunittypes"), strings.HasPrefix(lower, "pmckpmsiz"), strings.HasPrefix(lower, "pmcalmmsg"), strings.HasPrefix(lower, "pmccntldata"):
		return Address{Kind: lower}, nil
	default:
		return Address{}, fmt.Errorf("unsupported FOCAS address %q", value)
	}
}

func parsePMC(value, dataType string) (Address, error) {
	upper := strings.ToUpper(value)
	area := upper[0]
	rest := upper[1:]
	numberPart := rest
	bitPart := ""
	if dot := strings.Index(rest, "."); dot >= 0 {
		numberPart = rest[:dot]
		bitPart = rest[dot+1:]
	}
	number, err := strconv.Atoi(numberPart)
	if err != nil {
		return Address{}, fmt.Errorf("invalid FOCAS PMC address %q", value)
	}
	address := Address{Kind: "pmc", Area: area, Number: number}
	if bitPart != "" {
		if !strings.EqualFold(dataType, "bit") && !strings.EqualFold(dataType, "bool") {
			return Address{}, fmt.Errorf("FOCAS bit suffix requires BIT/BOOL")
		}
		bit, err := strconv.Atoi(bitPart)
		if err != nil || bit < 0 || bit > 7 {
			return Address{}, fmt.Errorf("invalid FOCAS bit address %q", value)
		}
		address.Bit = uint8(bit)
	}
	return address, nil
}

func axisAddress(value, lower string) (Address, error) {
	index := strings.LastIndex(lower, ".")
	if index < 0 {
		return Address{}, fmt.Errorf("invalid FOCAS axis address %q", value)
	}
	axis := value[index+1:]
	position := axisIndex(axis)
	if position < 0 {
		return Address{}, fmt.Errorf("invalid FOCAS axis %q", axis)
	}
	return Address{Kind: lower[:index], Key: axis, Axis: position}, nil
}

func prefixIndexAddress(value, kind string) (Address, error) {
	index := strings.LastIndex(value, ".")
	if index < 0 {
		return Address{}, fmt.Errorf("invalid FOCAS indexed address %q", value)
	}
	number, err := strconv.Atoi(value[index+1:])
	if err != nil || number < 0 {
		return Address{}, fmt.Errorf("invalid FOCAS index %q", value[index+1:])
	}
	return Address{Kind: kind, Index: number}, nil
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
