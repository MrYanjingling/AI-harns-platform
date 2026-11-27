package ethernetip1400

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

var pcccRequestorID = []byte{7, 0x1B, 0x05, 0x6B, 0xC5, 0xBC, 0x1A}

const (
	pcccServiceExecute = 0x4B
	pcccServiceRead    = 0xA2
	pcccServiceWrite   = 0xAB
	pcccCommandRequest = 0x0F
)

func readRequest(address pcccAddress, dataType string) ([]byte, error) {
	size, err := requestByteSize(address, dataType, false)
	if err != nil {
		return nil, err
	}
	return buildRequest(address, pcccServiceRead, size, nil, nil), nil
}

func writeRequest(address pcccAddress, value any, dataType string) ([]byte, error) {
	if isTimerCounter(address) && address.SubElement == subElementACC {
		return nil, fmt.Errorf("timer/counter ACC write requires read-modify-write")
	}
	data, mask, err := encodeWriteValue(address, value, dataType)
	if err != nil {
		return nil, err
	}
	size, err := requestByteSize(address, dataType, true)
	if err != nil {
		return nil, err
	}
	return buildRequest(address, pcccServiceWrite, size, data, mask), nil
}

// timerCounterWriteRequest builds a full-element protected write for T/C RMW.
func timerCounterWriteRequest(address pcccAddress, current []byte, value any, dataType string) ([]byte, error) {
	if !isTimerCounter(address) {
		return nil, fmt.Errorf("address is not a timer/counter file")
	}
	if len(current) < 6 {
		return nil, fmt.Errorf("timer/counter element is too short")
	}
	data := append([]byte(nil), current[:6]...)
	encoded, err := encodeTimerCounterValue(value, dataType)
	if err != nil {
		return nil, err
	}
	offset := 2
	if address.SubElement == subElementACC {
		offset = 4
	}
	copy(data[offset:offset+2], encoded)
	mask := make([]byte, 6)
	for index := 0; index < len(mask); index += 2 {
		binary.LittleEndian.PutUint16(mask[index:], 0xFFFF)
	}
	return buildRequest(address, pcccServiceWrite, 6, data, mask), nil
}

func encodeTimerCounterValue(value any, dataType string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "int16":
		v, ok := value.(int16)
		if !ok {
			return nil, fmt.Errorf("value is not int16")
		}
		return []byte{byte(v), byte(v >> 8)}, nil
	case "uint16":
		v, ok := value.(uint16)
		if !ok {
			return nil, fmt.Errorf("value is not uint16")
		}
		return []byte{byte(v), byte(v >> 8)}, nil
	default:
		return nil, fmt.Errorf("timer/counter only supports INT16/UINT16")
	}
}

func buildRequest(address pcccAddress, function byte, size byte, data []byte, mask []byte) []byte {
	cip := []byte{pcccServiceExecute, 0x02, 0x20, 0x67, 0x24, 0x01}
	cip = append(cip, pcccRequestorID...)
	cip = append(cip, pcccCommandRequest, 0x00, 0x00, 0x00, function)
	cip = append(cip, size, address.FileNum, address.FileType, address.Element, 0)
	if function == pcccServiceWrite {
		cip = append(cip, mask...)
		cip = append(cip, data...)
	}
	return cip
}

func encodeWriteValue(address pcccAddress, value any, dataType string) ([]byte, []byte, error) {
	size, err := valueByteSize(address, dataType)
	if err != nil {
		return nil, nil, err
	}
	data := make([]byte, 0, size)
	mask := make([]byte, 2)
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool":
		v, ok := value.(bool)
		if !ok {
			return nil, nil, fmt.Errorf("value is not bool")
		}
		bits := make([]byte, 2)
		if v {
			binary.LittleEndian.PutUint16(bits, uint16(1)<<address.Bit)
		}
		maskBits := make([]byte, 2)
		binary.LittleEndian.PutUint16(maskBits, uint16(1)<<address.Bit)
		return bits, maskBits, nil
	case "bit":
		v, ok := value.(int)
		if !ok || (v != 0 && v != 1) {
			return nil, nil, fmt.Errorf("value is not bit 0/1")
		}
		bits := make([]byte, 2)
		if v != 0 {
			binary.LittleEndian.PutUint16(bits, uint16(1)<<address.Bit)
		}
		maskBits := make([]byte, 2)
		binary.LittleEndian.PutUint16(maskBits, uint16(1)<<address.Bit)
		return bits, maskBits, nil
	case "int16":
		v, ok := value.(int16)
		if !ok {
			return nil, nil, fmt.Errorf("value is not int16")
		}
		data = append(data, byte(v), byte(v>>8))
	case "uint16":
		v, ok := value.(uint16)
		if !ok {
			return nil, nil, fmt.Errorf("value is not uint16")
		}
		data = append(data, byte(v), byte(v>>8))
	case "int32":
		v, ok := value.(int32)
		if !ok {
			return nil, nil, fmt.Errorf("value is not int32")
		}
		data = append(data, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	case "uint32":
		v, ok := value.(uint32)
		if !ok {
			return nil, nil, fmt.Errorf("value is not uint32")
		}
		data = append(data, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	case "float32":
		v, ok := value.(float32)
		if !ok {
			return nil, nil, fmt.Errorf("value is not float32")
		}
		raw := math.Float32bits(v)
		data = append(data, byte(raw), byte(raw>>8), byte(raw>>16), byte(raw>>24))
	default:
		return nil, nil, fmt.Errorf("data type %q is not writable", dataType)
	}
	binary.LittleEndian.PutUint16(mask, 0xFFFF)
	return data, mask, nil
}

func parseReadResponse(response []byte, address pcccAddress, dataType string) (any, error) {
	data, err := parsePCCCResponse(response)
	if err != nil {
		return nil, err
	}
	if isTimerCounter(address) {
		if len(data) < 6 {
			return nil, fmt.Errorf("ethernet/ip 1400 timer/counter data is too short")
		}
		offset := 0
		if address.SubElement == subElementACC {
			offset = 2
		}
		data = data[offset : offset+2]
	}
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "bit":
		size := 2
		if address.FileType == fileTypeFloat {
			size = 4
		}
		if len(data) < size {
			return nil, fmt.Errorf("ethernet/ip 1400 bit data is too short")
		}
		var mask uint32
		if size == 4 {
			mask = binary.LittleEndian.Uint32(data)
		} else {
			mask = uint32(binary.LittleEndian.Uint16(data))
		}
		value := (mask>>address.Bit)&1 != 0
		if strings.EqualFold(dataType, "bit") {
			if value {
				return 1, nil
			}
			return 0, nil
		}
		return value, nil
	case "int16":
		if len(data) < 2 {
			return nil, fmt.Errorf("ethernet/ip 1400 int16 data is too short")
		}
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "uint16":
		if len(data) < 2 {
			return nil, fmt.Errorf("ethernet/ip 1400 uint16 data is too short")
		}
		return binary.LittleEndian.Uint16(data), nil
	case "int32":
		if len(data) < 4 {
			return nil, fmt.Errorf("ethernet/ip 1400 int32 data is too short")
		}
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "uint32":
		if len(data) < 4 {
			return nil, fmt.Errorf("ethernet/ip 1400 uint32 data is too short")
		}
		return binary.LittleEndian.Uint32(data), nil
	case "float32":
		if len(data) < 4 {
			return nil, fmt.Errorf("ethernet/ip 1400 float32 data is too short")
		}
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	default:
		return nil, fmt.Errorf("unsupported PCCC data type %q", dataType)
	}
}

func parseWriteResponse(response []byte) error {
	_, err := parsePCCCResponse(response)
	return err
}

func parsePCCCResponse(response []byte) ([]byte, error) {
	if len(response) < 4 {
		return nil, fmt.Errorf("ethernet/ip 1400 response is too short")
	}
	if response[0] != 0xCB {
		if response[2] != 0 {
			return nil, fmt.Errorf("ethernet/ip 1400 response status %d", response[2])
		}
		return nil, fmt.Errorf("ethernet/ip 1400 response service %02X", response[0])
	}
	if response[2] != 0 {
		return nil, fmt.Errorf("ethernet/ip 1400 response status %d", response[2])
	}
	offset := 4 + int(response[3])*2
	if offset+7+4 > len(response) {
		return nil, fmt.Errorf("ethernet/ip 1400 response body is too short")
	}
	offset += 7
	pcccStatus := response[offset+1]
	if pcccStatus != 0 {
		return nil, fmt.Errorf("ethernet/ip 1400 PCCC status %d", pcccStatus)
	}
	offset += 4
	return response[offset:], nil
}
