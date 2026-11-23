package ethernetip1400

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	fileTypeOutput  byte = 0x82
	fileTypeInput   byte = 0x83
	fileTypeStatus  byte = 0x84
	fileTypeBinary  byte = 0x85
	fileTypeTimer   byte = 0x86
	fileTypeCounter byte = 0x87
	fileTypeInteger byte = 0x89
	fileTypeFloat   byte = 0x8A
)

const (
	subElementNone byte = 0
	subElementPRE  byte = 1
	subElementACC  byte = 2
)

type pcccAddress struct {
	FileNum    byte
	FileType   byte
	Element    byte
	Bit        byte
	BitSet     bool
	SubElement byte
}

func parseAddress(value, dataType string) (pcccAddress, error) {
	text := strings.ToUpper(strings.TrimSpace(value))
	if text == "" {
		return pcccAddress{}, fmt.Errorf("ethernet/ip 1400 address is empty")
	}
	dataType = strings.ToLower(strings.TrimSpace(dataType))
	subElement := subElementNone
	if index := strings.Index(text, "."); index >= 0 {
		suffix := text[index+1:]
		text = text[:index]
		switch suffix {
		case "PRE":
			subElement = subElementPRE
		case "ACC":
			subElement = subElementACC
		default:
			return pcccAddress{}, fmt.Errorf("unsupported sub-element %q", value)
		}
	}
	bitType := dataType == "bit" || dataType == "bool"
	main := text
	bitText := ""
	if index := strings.Index(text, "/"); index >= 0 {
		main = text[:index]
		bitText = text[index+1:]
		if bitText == "" {
			return pcccAddress{}, fmt.Errorf("invalid bit address %q", value)
		}
		if !bitType {
			return pcccAddress{}, fmt.Errorf("bit suffix is only valid for BIT/BOOL address %q", value)
		}
	}

	prefix := main[0]
	fileNum := 0
	element := 0
	hasElement := false
	var fileType byte
	switch prefix {
	case 'B', 'C', 'F', 'N', 'T':
		if index := strings.Index(main[1:], ":"); index >= 0 {
			var err error
			fileNum, err = parseByte(main[1 : 1+index])
			if err != nil {
				return pcccAddress{}, fmt.Errorf("invalid file number in %q", value)
			}
			element, err = parseByte(main[2+index:])
			if err != nil {
				return pcccAddress{}, fmt.Errorf("invalid element number in %q", value)
			}
			hasElement = true
		} else if bitType && bitText != "" {
			var err error
			fileNum, err = parseByte(main[1:])
			if err != nil {
				return pcccAddress{}, fmt.Errorf("invalid file number in %q", value)
			}
		} else {
			return pcccAddress{}, fmt.Errorf("invalid address %q, expected X<file>:<element>", value)
		}
		switch prefix {
		case 'B':
			fileType = fileTypeBinary
		case 'C':
			fileType = fileTypeCounter
		case 'F':
			fileType = fileTypeFloat
		case 'N':
			fileType = fileTypeInteger
		case 'T':
			fileType = fileTypeTimer
		}
	case 'I', 'O', 'S':
		body := main[1:]
		body = strings.TrimPrefix(body, ":")
		if body == "" {
			return pcccAddress{}, fmt.Errorf("invalid address %q, expected element number", value)
		}
		var err error
		element, err = parseByte(body)
		if err != nil {
			return pcccAddress{}, fmt.Errorf("invalid element number in %q", value)
		}
		hasElement = true
		switch prefix {
		case 'I':
			fileType = fileTypeInput
			fileNum = 1
		case 'O':
			fileType = fileTypeOutput
		case 'S':
			fileType = fileTypeStatus
			fileNum = 2
		}
	default:
		return pcccAddress{}, fmt.Errorf("unsupported PCCC file type %q", value)
	}
	_ = hasElement

	if subElement != subElementNone && fileType != fileTypeTimer && fileType != fileTypeCounter {
		return pcccAddress{}, fmt.Errorf("sub-element %q is only valid for timer/counter files", value)
	}
	if (fileType == fileTypeTimer || fileType == fileTypeCounter) && subElement == subElementNone {
		subElement = subElementPRE
	}
	address := pcccAddress{FileNum: byte(fileNum), FileType: fileType, Element: byte(element), BitSet: bitType, SubElement: subElement}
	if bitType {
		bit := 0
		if bitText != "" {
			value, err := strconv.Atoi(bitText)
			if err != nil || value < 0 {
				return pcccAddress{}, fmt.Errorf("invalid bit number in %q", value)
			}
			bit = value
		}
		limit := 16
		if fileType == fileTypeFloat {
			limit = 32
		}
		if bitText != "" && strings.Contains(main, ":") {
			if bit >= limit {
				return pcccAddress{}, fmt.Errorf("bit number %d is out of range for %q", bit, value)
			}
		} else {
			address.Element = byte(bit / limit)
			bit = bit % limit
		}
		address.Bit = byte(bit)
	}
	return address, nil
}

func parseByte(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number < 0 || number > 255 {
		return 0, fmt.Errorf("value %q must be between 0 and 255", value)
	}
	return number, nil
}

func valueByteSize(address pcccAddress, dataType string) (byte, error) {
	if address.FileType == fileTypeTimer || address.FileType == fileTypeCounter {
		switch strings.ToLower(strings.TrimSpace(dataType)) {
		case "int16", "uint16":
			return 2, nil
		default:
			return 0, fmt.Errorf("timer/counter sub-element only supports INT16/UINT16")
		}
	}
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "bit":
		if address.FileType == fileTypeFloat {
			return 4, nil
		}
		return 2, nil
	case "int16", "uint16":
		return 2, nil
	case "int32", "uint32", "float32":
		return 4, nil
	default:
		return 0, fmt.Errorf("unsupported PCCC data type %q", dataType)
	}
}

func requestByteSize(address pcccAddress, dataType string, write bool) (byte, error) {
	if address.FileType == fileTypeTimer || address.FileType == fileTypeCounter {
		if write {
			return 2, nil
		}
		return 6, nil
	}
	return valueByteSize(address, dataType)
}

func isTimerCounter(address pcccAddress) bool {
	return address.FileType == fileTypeTimer || address.FileType == fileTypeCounter
}
