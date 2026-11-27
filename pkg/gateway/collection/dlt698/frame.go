package dlt698

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

const (
	controlGetRequest  = 0x43
	controlGetResponse = 0x83
	apduGetRequest     = 0x05
	apduGetResponse    = 0x85
)

var fcsTable [256]uint16

func init() {
	for index := 0; index < 256; index++ {
		value := uint16(index)
		for bit := 0; bit < 8; bit++ {
			if value&1 != 0 {
				value = value>>1 ^ 0x8408
			} else {
				value >>= 1
			}
		}
		fcsTable[index] = value
	}
}

func fcs16(data []byte) uint16 {
	value := uint16(0xFFFF)
	for _, item := range data {
		value = value>>8 ^ fcsTable[(value^uint16(item))&0xFF]
	}
	return value ^ 0xFFFF
}

func appendFCS(dst, data []byte) []byte {
	value := fcs16(data)
	return append(dst, byte(value), byte(value>>8))
}

func buildGetRequest(meter []byte, client byte, oad [4]byte, piid byte) []byte {
	address := buildAddressArea(meter, client)
	// GET-Request-Normal: service(0x05), choice(0x01), PIID, OAD(4B), TimeTag OPTIONAL(0x00).
	apdu := []byte{apduGetRequest, 0x01, piid, oad[0], oad[1], oad[2], oad[3], 0x00}
	total := 1 + 2 + 1 + len(address) + 2 + len(apdu) + 2 + 1
	length := uint16(total - 2)
	header := []byte{byte(length), byte(length >> 8), controlGetRequest}
	header = append(header, address...)
	body := appendFCS(append([]byte{}, header...), header)
	body = append(body, apdu...)
	body = appendFCS(body, body)
	frame := append([]byte{0x68}, body...)
	return append(frame, 0x16)
}

func parseGetResponse(frame []byte, meter []byte, oad [4]byte, piid byte) ([]byte, error) {
	if len(frame) < 12 || frame[0] != 0x68 || frame[len(frame)-1] != 0x16 {
		return nil, fmt.Errorf("invalid DL/T 698 frame")
	}
	length := int(binary.LittleEndian.Uint16(frame[1:3]))
	if len(frame) != length+2 {
		return nil, fmt.Errorf("invalid DL/T 698 length")
	}
	if frame[3] != controlGetResponse {
		if frame[3]&0x40 != 0 {
			return nil, fmt.Errorf("meter returned an error control code 0x%02X", frame[3])
		}
		return nil, fmt.Errorf("unexpected control code 0x%02X", frame[3])
	}
	addressBytes := int(frame[4]&0x0F) + 1
	addressEnd := 4 + 1 + addressBytes + 1
	if addressEnd+2 > len(frame)-3 {
		return nil, fmt.Errorf("invalid DL/T 698 address")
	}
	if string(frame[5:5+addressBytes]) != string(reverse(meter)) {
		return nil, fmt.Errorf("meter address mismatch")
	}
	if got := binary.LittleEndian.Uint16(frame[addressEnd : addressEnd+2]); got != fcs16(frame[1:addressEnd]) {
		return nil, fmt.Errorf("DL/T 698 header checksum mismatch")
	}
	apduStart := addressEnd + 2
	apduEnd := len(frame) - 3
	if got := binary.LittleEndian.Uint16(frame[apduEnd : apduEnd+2]); got != fcs16(frame[1:apduEnd]) {
		return nil, fmt.Errorf("DL/T 698 frame checksum mismatch")
	}
	apdu := frame[apduStart:apduEnd]
	if len(apdu) < 8 || apdu[0] != apduGetResponse || apdu[1] != 0x01 {
		return nil, fmt.Errorf("invalid DL/T 698 GET response")
	}
	if apdu[2] != piid || string(apdu[3:7]) != string(oad[:]) {
		return nil, fmt.Errorf("DL/T 698 response does not match request")
	}
	if apdu[7] != 0x01 {
		if apdu[7] == 0x00 && len(apdu) >= 9 {
			return nil, fmt.Errorf("DL/T 698 object read failed, dar 0x%02X", apdu[8])
		}
		return nil, fmt.Errorf("DL/T 698 object read failed, result 0x%02X", apdu[7])
	}
	return apdu[8:], nil
}

func reverse(data []byte) []byte {
	out := make([]byte, len(data))
	for index, value := range data {
		out[len(data)-1-index] = value
	}
	return out
}

func decodeData(data []byte, dataType string) (any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty DL/T 698 data")
	}
	if data[0] == 0x01 {
		if len(data) < 3 {
			return nil, fmt.Errorf("invalid DL/T 698 array")
		}
		data = data[2:]
	}
	if len(data) < 2 {
		return nil, fmt.Errorf("invalid DL/T 698 TLV")
	}
	typeCode := data[0]
	payload := data[1:]
	var raw uint64
	var signed int64
	var isSigned bool
	switch typeCode {
	case 0x05, 0x06:
		if len(payload) < 4 {
			return nil, fmt.Errorf("short DL/T 698 integer")
		}
		raw = uint64(binary.BigEndian.Uint32(payload[:4]))
		signed = int64(int32(raw))
		isSigned = typeCode == 0x06
	case 0x10:
		if len(payload) < 2 {
			return nil, fmt.Errorf("short DL/T 698 integer")
		}
		raw = uint64(binary.BigEndian.Uint16(payload[:2]))
		signed = int64(int16(raw))
		isSigned = true
	case 0x11:
		if len(payload) < 1 {
			return nil, fmt.Errorf("short DL/T 698 integer")
		}
		raw = uint64(payload[0])
		signed = int64(int8(payload[0]))
	case 0x12:
		if len(payload) < 2 {
			return nil, fmt.Errorf("short DL/T 698 integer")
		}
		raw = uint64(binary.BigEndian.Uint16(payload[:2]))
	case 0x14, 0x15:
		if len(payload) < 8 {
			return nil, fmt.Errorf("short DL/T 698 integer")
		}
		raw = binary.BigEndian.Uint64(payload[:8])
		signed = int64(raw)
		isSigned = typeCode == 0x15
	default:
		return nil, fmt.Errorf("unsupported DL/T 698 data type 0x%02X", typeCode)
	}
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "float32", "float64":
		if isSigned {
			return float64(signed), nil
		}
		return float64(raw), nil
	case "int16", "int32", "int64":
		if isSigned {
			return signed, nil
		}
		if raw > math.MaxInt64 {
			return int64(math.MaxInt64), nil
		}
		return int64(raw), nil
	default:
		return raw, nil
	}
}
