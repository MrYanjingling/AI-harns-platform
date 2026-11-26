package modbustcp

import (
	"fmt"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbus"
	"harns-platform/pkg/utils/binutil"
)

const tcpNonDataLength = 9

type wireFrame struct {
	frame   *modbus.Frame
	request []byte
}

func buildWireFrames(variables []*collection.Variable, positionAddress uint) ([]*wireFrame, error) {
	frames, err := modbus.BuildFrames(variables, positionAddress)
	if err != nil {
		return nil, err
	}
	result := make([]*wireFrame, 0, len(frames))
	for _, frame := range frames {
		result = append(result, &wireFrame{
			frame:   frame,
			request: buildTCPReadRequest(frame),
		})
	}
	return result, nil
}

func buildTCPReadRequest(frame *modbus.Frame) []byte {
	message := make([]byte, 12)
	binutil.WriteUint16BigEndian(message[2:], 0)
	binutil.WriteUint16BigEndian(message[4:], 6)
	message[6] = frame.Slave
	message[7] = frame.Function
	binutil.WriteUint16BigEndian(message[8:], frame.Start)
	binutil.WriteUint16BigEndian(message[10:], frame.Quantity)
	return message
}

func validateTCPReadResponse(response []byte, frame *modbus.Frame, transaction uint16) ([]byte, error) {
	if len(response) < tcpNonDataLength {
		return nil, fmt.Errorf("modbusTcp response is too short")
	}
	if binutil.ParseUint16BigEndian(response) != transaction {
		return nil, fmt.Errorf("modbusTcp transaction mismatch")
	}
	if binutil.ParseUint16BigEndian(response[2:]) != 0 {
		return nil, fmt.Errorf("modbusTcp protocol id mismatch")
	}
	length := int(binutil.ParseUint16BigEndian(response[4:]))
	if length+6 != len(response) {
		return nil, fmt.Errorf("modbusTcp response length mismatch")
	}
	if response[6] != frame.Slave {
		return nil, fmt.Errorf("modbusTcp slave mismatch")
	}
	if response[7]&0x80 != 0 {
		return nil, &modbus.ExceptionError{Function: response[7] & 0x7f, Code: response[8]}
	}
	if response[7] != frame.Function {
		return nil, fmt.Errorf("modbusTcp function mismatch")
	}
	dataLength := int(response[8])
	if dataLength+9 != len(response) {
		return nil, fmt.Errorf("modbusTcp byte count mismatch")
	}
	payload := response[9:]
	if frame.Function == modbus.FunctionReadCoils || frame.Function == modbus.FunctionReadDiscreteInputs {
		expected := int((frame.Quantity + 7) / 8)
		if dataLength != expected {
			return nil, fmt.Errorf("modbusTcp coil byte count mismatch")
		}
		return binutil.ExpandBool(payload, len(payload)), nil
	}
	expected := int(frame.Quantity * 2)
	if dataLength != expected {
		return nil, fmt.Errorf("modbusTcp register byte count mismatch")
	}
	return payload, nil
}
