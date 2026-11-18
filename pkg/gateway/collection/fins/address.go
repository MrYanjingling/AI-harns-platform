package fins

import (
	"fmt"
	"strconv"
	"strings"
)

type wordOrder struct {
	swapBytes    bool
	reverseWords bool
}

type parsedAddress struct {
	areaWord     byte
	areaBit      byte
	start        uint16
	bit          uint8
	length       uint16
	isBit        bool
	bitExplicit  bool
	order        wordOrder
	stringHigh   bool
	stringLength uint16
}

func parseAddress(value, dataType string, bits uint8) (parsedAddress, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return parsedAddress{}, fmt.Errorf("fins address is empty")
	}
	order := wordOrder{}
	if index := strings.LastIndex(value, "#"); index >= 0 {
		suffix := strings.ToUpper(value[index+1:])
		value = value[:index]
		for _, char := range suffix {
			switch char {
			case 'B':
				order.swapBytes = true
			case 'L':
				order.reverseWords = true
			default:
				return parsedAddress{}, fmt.Errorf("unsupported fins byte order %q", suffix)
			}
		}
	}
	area, rest, err := splitArea(value)
	if err != nil {
		return parsedAddress{}, err
	}
	result := parsedAddress{areaWord: area.word, areaBit: area.bit, order: order}
	dot := strings.Index(rest, ".")
	addressPart := rest
	suffix := ""
	if dot >= 0 {
		addressPart = rest[:dot]
		suffix = rest[dot+1:]
	}
	address, err := strconv.ParseUint(addressPart, 10, 16)
	if err != nil {
		return parsedAddress{}, fmt.Errorf("invalid fins address %q", value)
	}
	result.start = uint16(address)
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "string", "byte_array":
		if suffix == "" {
			return parsedAddress{}, fmt.Errorf("fins string address requires a length suffix")
		}
		lengthPart := suffix
		high := true
		upper := strings.ToUpper(suffix)
		if strings.HasSuffix(upper, "H") {
			lengthPart = suffix[:len(suffix)-1]
			high = true
		} else if strings.HasSuffix(upper, "L") {
			lengthPart = suffix[:len(suffix)-1]
			high = false
		} else {
			return parsedAddress{}, fmt.Errorf("fins string address requires H or L direction")
		}
		length, err := strconv.ParseUint(lengthPart, 10, 16)
		if err != nil || length == 0 || length > 128 || length%2 != 0 {
			return parsedAddress{}, fmt.Errorf("invalid fins string length %q", suffix)
		}
		if area.bitOnly || area.word == 0 {
			return parsedAddress{}, fmt.Errorf("fins area %q does not support string reads", value)
		}
		result.stringLength = uint16(length)
		result.stringHigh = high
		result.length = uint16(length) / 2
		if uint16(length)%2 != 0 {
			result.length++
		}
		return result, nil
	case "bit", "bool":
		result.isBit = true
		result.length = 1
		if area.wordOnly {
			return parsedAddress{}, fmt.Errorf("fins area %q does not support bit reads", value)
		}
		if suffix != "" {
			bit, err := strconv.ParseUint(suffix, 10, 8)
			if err != nil || bit > 15 {
				return parsedAddress{}, fmt.Errorf("invalid fins bit address %q", value)
			}
			result.bit = uint8(bit)
			result.bitExplicit = true
		} else if bits <= 15 {
			result.bit = bits
		}
		return result, nil
	}
	if suffix != "" {
		return parsedAddress{}, fmt.Errorf("fins suffix is only valid for bit and string addresses")
	}
	if area.bitOnly {
		result.isBit = true
		result.length = 1
		result.bit = 0
		return result, nil
	}
	result.length = dataTypeWords(dataType)
	if result.length == 0 {
		return parsedAddress{}, fmt.Errorf("unsupported fins data type %q", dataType)
	}
	return result, nil
}

type areaCodes struct {
	word     byte
	bit      byte
	bitOnly  bool
	wordOnly bool
}

func splitArea(value string) (areaCodes, string, error) {
	upper := strings.ToUpper(value)
	if strings.HasPrefix(upper, "EM") {
		rest := upper[2:]
		if strings.HasPrefix(rest, "CW") {
			number := rest[2:]
			if number == "" {
				return areaCodes{}, "", fmt.Errorf("invalid fins address %q", value)
			}
			return emCodes(12), number, nil
		}
		index := strings.Index(rest, "W")
		if index <= 0 {
			return areaCodes{}, "", fmt.Errorf("invalid fins address %q", value)
		}
		bank, err := strconv.Atoi(rest[:index])
		if err != nil || bank < 0 || bank > 12 {
			return areaCodes{}, "", fmt.Errorf("invalid fins EM bank in %q", value)
		}
		return emCodes(bank), rest[index+1:], nil
	}
	if len(upper) > 2 && upper[0] == 'E' && upper[1] >= '0' && upper[1] <= '9' && upper[2] == 'W' {
		bank, err := strconv.Atoi(string(upper[1]))
		if err != nil {
			return areaCodes{}, "", fmt.Errorf("invalid fins EM bank in %q", value)
		}
		return emCodes(bank), upper[3:], nil
	}
	areas := []struct {
		prefix string
		codes  areaCodes
	}{
		{"CIO", areaCodes{word: 0xB0, bit: 0x30}},
		{"W", areaCodes{word: 0xB1, bit: 0x31}},
		{"H", areaCodes{word: 0xB2, bit: 0x32}},
		{"A", areaCodes{word: 0xB3, bit: 0x33}},
		{"D", areaCodes{word: 0x82, bit: 0x02}},
		{"P", areaCodes{word: 0x89, wordOnly: true}},
		{"F", areaCodes{bit: 0x09, bitOnly: true}},
	}
	for _, item := range areas {
		if strings.HasPrefix(upper, item.prefix) {
			rest := value[len(item.prefix):]
			if rest == "" {
				return areaCodes{}, "", fmt.Errorf("invalid fins address %q", value)
			}
			return item.codes, rest, nil
		}
	}
	return areaCodes{}, "", fmt.Errorf("unsupported fins area in %q", value)
}

func emCodes(bank int) areaCodes {
	return areaCodes{word: byte(0xA0 + bank), bit: byte(0x20 + bank)}
}

func dataTypeWords(dataType string) uint16 {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "int16", "uint16", "word":
		return 1
	case "int32", "uint32", "float32", "dword":
		return 2
	case "int64", "uint64", "float64", "lword":
		return 4
	default:
		return 0
	}
}

func (a parsedAddress) areaCode() byte {
	if a.isBit {
		return a.areaBit
	}
	return a.areaWord
}
