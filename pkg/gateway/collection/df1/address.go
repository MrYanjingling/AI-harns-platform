package df1

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	fileStatus  = 0x84
	fileBit     = 0x85
	fileTimer   = 0x86
	fileCounter = 0x87
	fileControl = 0x88
	fileInteger = 0x89
	fileFloat   = 0x8A
	fileOutput  = 0x8B
	fileInput   = 0x8C
	fileString  = 0x8D
	fileASCII   = 0x8E
	fileBCD     = 0x8F
)

type Address struct {
	FileType byte
	FileNum  byte
	Element  byte
	Bit      uint8
	Sub      string
	ByteSize byte
}

func ParseAddress(value, dataType string) (Address, error) {
	value = strings.TrimSpace(value)
	index := strings.Index(value, ":")
	if index < 2 {
		return Address{}, fmt.Errorf("DF1 address must be Fn:element")
	}
	prefix := strings.ToUpper(value[:index])
	rest := value[index+1:]
	sub := ""
	if dot := strings.Index(rest, "."); dot >= 0 {
		sub = strings.ToUpper(rest[dot+1:])
		rest = rest[:dot]
	}
	bit := ""
	if slash := strings.Index(rest, "/"); slash >= 0 {
		bit = rest[slash+1:]
		rest = rest[:slash]
	}
	file, ok := fileTypes[prefix[0]]
	if !ok {
		return Address{}, fmt.Errorf("unsupported DF1 file %q", prefix)
	}
	fileNum, err := strconv.Atoi(prefix[1:])
	if err != nil || fileNum < 0 || fileNum > 255 {
		return Address{}, fmt.Errorf("invalid DF1 file number %q", value)
	}
	element, err := strconv.Atoi(rest)
	if err != nil || element < 0 || element > 255 {
		return Address{}, fmt.Errorf("invalid DF1 element %q", value)
	}
	address := Address{FileType: file, FileNum: byte(fileNum), Element: byte(element), Sub: sub}
	if bit != "" {
		number, err := strconv.Atoi(bit)
		if err != nil || number < 0 || number > 15 {
			return Address{}, fmt.Errorf("invalid DF1 bit %q", value)
		}
		address.Bit = uint8(number)
	}
	if file == fileTimer || file == fileCounter || file == fileControl {
		if sub != "PRE" && sub != "ACC" {
			return Address{}, fmt.Errorf("DF1 timer/counter/control requires .PRE or .ACC")
		}
		address.ByteSize = 6
	} else {
		address.ByteSize = byteSize(dataType)
	}
	if address.ByteSize == 0 {
		return Address{}, fmt.Errorf("unsupported DF1 data type %q", dataType)
	}
	return address, nil
}

var fileTypes = map[byte]byte{
	'A': fileASCII, 'B': fileBit, 'C': fileCounter, 'F': fileFloat,
	'N': fileInteger, 'R': fileControl, 'S': fileString, 'T': fileTimer,
	'U': fileStatus, 'O': fileOutput, 'I': fileInput,
}

func byteSize(dataType string) byte {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "bit", "int8", "uint8":
		return 2
	case "int16", "uint16":
		return 2
	case "int32", "uint32", "float":
		return 4
	case "string":
		return 0x52
	default:
		return 0
	}
}
