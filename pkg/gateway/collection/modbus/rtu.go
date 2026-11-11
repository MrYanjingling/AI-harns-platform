package modbus

import (
	"fmt"

	"harns-platform/pkg/utils/binutil"
	"harns-platform/pkg/utils/crcutil"
)

// BuildRTURequest builds a Modbus RTU application data unit.
func BuildRTURequest(frame *Frame) []byte {
	message := make([]byte, 6)
	message[0] = frame.Slave
	message[1] = frame.Function
	binutil.WriteUint16BigEndian(message[2:], frame.Start)
	binutil.WriteUint16BigEndian(message[4:], frame.Quantity)
	crc := make([]byte, 2)
	binutil.WriteUint16BigEndian(crc, crcutil.CheckCrc16sum(message))
	return append(message, crc...)
}

// RTUResponseLength returns the expected RTU response length.
func RTUResponseLength(frame *Frame) int {
	if frame.Function == FunctionReadCoils || frame.Function == FunctionReadDiscreteInputs {
		return 5 + int((frame.Quantity+7)/8)
	}
	return 5 + int(frame.Quantity*2)
}

// ValidateRTUResponse validates an RTU response and returns its data payload.
func ValidateRTUResponse(response []byte, frame *Frame) ([]byte, error) {
	if len(response) < 5 {
		return nil, fmt.Errorf("modbusRtu response is too short")
	}
	crc := binutil.ParseUint16BigEndian(response[len(response)-2:])
	if crc != crcutil.CheckCrc16sum(response[:len(response)-2]) {
		return nil, fmt.Errorf("modbusRtu crc mismatch")
	}
	if response[0] != frame.Slave {
		return nil, fmt.Errorf("modbusRtu slave mismatch")
	}
	if response[1]&0x80 != 0 {
		return nil, fmt.Errorf("modbusRtu function error %d", response[1]&0x7f)
	}
	if response[1] != frame.Function {
		return nil, fmt.Errorf("modbusRtu function mismatch")
	}
	dataLength := int(response[2])
	if dataLength+5 != len(response) {
		return nil, fmt.Errorf("modbusRtu byte count mismatch")
	}
	payload := response[3 : 3+dataLength]
	if frame.Function == FunctionReadCoils || frame.Function == FunctionReadDiscreteInputs {
		expected := int((frame.Quantity + 7) / 8)
		if dataLength != expected {
			return nil, fmt.Errorf("modbusRtu coil byte count mismatch")
		}
		return binutil.ExpandBool(payload, len(payload)), nil
	}
	if dataLength != int(frame.Quantity*2) {
		return nil, fmt.Errorf("modbusRtu register byte count mismatch")
	}
	return payload, nil
}
