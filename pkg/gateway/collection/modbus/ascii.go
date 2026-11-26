package modbus

import (
	"encoding/hex"
	"fmt"
	"strings"

	"harns-platform/pkg/utils/binutil"
)

// ComputeLRC returns the Modbus ASCII LRC for one slave + PDU payload.
func ComputeLRC(data []byte) byte {
	var sum byte
	for _, value := range data {
		sum += value
	}
	return byte((256 - int(sum)) & 0xff)
}

// BuildASCIIRequest builds a Modbus ASCII read request.
func BuildASCIIRequest(frame *Frame) []byte {
	pdu := []byte{frame.Function}
	pdu = append(pdu, binutil.Uint16ToBytesBigEndian(frame.Start)...)
	pdu = append(pdu, binutil.Uint16ToBytesBigEndian(frame.Quantity)...)
	return buildASCIIFrame(frame.Slave, pdu)
}

// BuildASCIIWriteRequest builds a Modbus ASCII write request.
func BuildASCIIWriteRequest(write *WriteFrame) []byte {
	return buildASCIIFrame(write.Slave, write.PDU())
}

// ParseASCIIResponse validates an ASCII read response and returns its data payload.
func ParseASCIIResponse(raw []byte, frame *Frame) ([]byte, error) {
	slave, pdu, err := parseASCIIFrame(raw)
	if err != nil {
		return nil, err
	}
	if slave != frame.Slave {
		return nil, fmt.Errorf("modbusAscii slave mismatch")
	}
	if len(pdu) < 2 {
		return nil, fmt.Errorf("modbusAscii response is too short")
	}
	if pdu[0]&0x80 != 0 {
		return nil, &ExceptionError{Function: pdu[0] & 0x7f, Code: pdu[1]}
	}
	if pdu[0] != frame.Function {
		return nil, fmt.Errorf("modbusAscii function mismatch")
	}
	dataLength := int(pdu[1])
	if dataLength != len(pdu)-2 {
		return nil, fmt.Errorf("modbusAscii byte count mismatch")
	}
	payload := pdu[2:]
	if frame.Function == FunctionReadCoils || frame.Function == FunctionReadDiscreteInputs {
		expected := int((frame.Quantity + 7) / 8)
		if dataLength != expected {
			return nil, fmt.Errorf("modbusAscii coil byte count mismatch")
		}
		return binutil.ExpandBool(payload, len(payload)), nil
	}
	if dataLength != int(frame.Quantity*2) {
		return nil, fmt.Errorf("modbusAscii register byte count mismatch")
	}
	return payload, nil
}

// ValidateASCIIWriteResponse validates an ASCII write response.
func ValidateASCIIWriteResponse(raw []byte, write *WriteFrame) error {
	slave, pdu, err := parseASCIIFrame(raw)
	if err != nil {
		return err
	}
	if slave != write.Slave {
		return fmt.Errorf("modbusAscii write slave mismatch")
	}
	if len(pdu) < 2 {
		return fmt.Errorf("modbusAscii write response is too short")
	}
	if pdu[0]&0x80 != 0 {
		return &ExceptionError{Function: pdu[0] & 0x7f, Code: pdu[1]}
	}
	if pdu[0] != write.Function {
		return fmt.Errorf("modbusAscii write function mismatch")
	}
	if len(pdu) != 5 {
		return fmt.Errorf("modbusAscii write response length mismatch")
	}
	return nil
}

func buildASCIIFrame(slave uint8, pdu []byte) []byte {
	data := make([]byte, 0, 1+len(pdu))
	data = append(data, slave)
	data = append(data, pdu...)
	lrc := ComputeLRC(data)
	return []byte(fmt.Sprintf(":%s%02X\r\n", strings.ToUpper(hex.EncodeToString(data)), lrc))
}

func parseASCIIFrame(raw []byte) (uint8, []byte, error) {
	if len(raw) < 9 {
		return 0, nil, fmt.Errorf("modbusAscii response is too short")
	}
	if raw[0] != ':' {
		return 0, nil, fmt.Errorf("modbusAscii response has invalid start")
	}
	if raw[len(raw)-2] != '\r' || raw[len(raw)-1] != '\n' {
		return 0, nil, fmt.Errorf("modbusAscii response has invalid terminator")
	}
	body := raw[1 : len(raw)-2]
	if len(body)%2 != 0 {
		return 0, nil, fmt.Errorf("modbusAscii response has invalid hex length")
	}
	decoded, err := hex.DecodeString(string(body))
	if err != nil {
		return 0, nil, fmt.Errorf("modbusAscii response has invalid hex")
	}
	if len(decoded) < 4 {
		return 0, nil, fmt.Errorf("modbusAscii response is too short")
	}
	lrc := decoded[len(decoded)-1]
	if ComputeLRC(decoded[:len(decoded)-1]) != lrc {
		return 0, nil, fmt.Errorf("modbusAscii lrc mismatch")
	}
	return decoded[0], decoded[1 : len(decoded)-1], nil
}
