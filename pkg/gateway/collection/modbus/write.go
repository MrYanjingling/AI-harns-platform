package modbus

import (
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
	"harns-platform/pkg/utils/crcutil"
)

// WriteFrame is one Modbus write request.
type WriteFrame struct {
	Slave    uint8
	Function uint8
	Start    uint16
	Quantity uint16
	Data     []byte
}

// BuildWrite builds a Modbus write frame from a variable and raw value.
func BuildWrite(variable *collection.Variable, value any, positionAddress uint, memoryLayout string) (*WriteFrame, error) {
	if variable == nil {
		return nil, fmt.Errorf("variable is nil")
	}
	slave, function, address, err := ParseAddress(variable.Address, positionAddress)
	if err != nil {
		return nil, err
	}
	switch function {
	case FunctionReadDiscreteInputs, FunctionReadInputRegisters:
		return nil, fmt.Errorf("function code %d is read-only", function)
	case FunctionReadCoils:
		boolean, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("coil variable %s requires bool", variable.Name)
		}
		data := []byte{0x00, 0x00}
		if boolean {
			data[0], data[1] = 0xFF, 0x00
		}
		return &WriteFrame{Slave: slave, Function: 5, Start: address, Quantity: 1, Data: data}, nil
	case FunctionReadHoldingRegisters:
		if strings.EqualFold(variable.DataType, "bool") {
			return nil, fmt.Errorf("register bool write requires read-modify-write")
		}
		words, err := DataTypeWords(variable.DataType)
		if err != nil {
			return nil, err
		}
		data, err := encodeWriteValue(value, variable.DataType, memoryLayout, words)
		if err != nil {
			return nil, err
		}
		if words == 1 {
			return &WriteFrame{Slave: slave, Function: 6, Start: address, Quantity: 1, Data: data}, nil
		}
		return &WriteFrame{Slave: slave, Function: 16, Start: address, Quantity: words, Data: data}, nil
	default:
		return nil, fmt.Errorf("function code %d is not writable", function)
	}
}

func encodeWriteValue(value any, dataType, layout string, words uint16) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "int16":
		v, ok := value.(int16)
		if !ok {
			return nil, fmt.Errorf("value is not int16")
		}
		return encodeUint16(uint16(v), layout), nil
	case "uint16":
		v, ok := value.(uint16)
		if !ok {
			return nil, fmt.Errorf("value is not uint16")
		}
		return encodeUint16(v, layout), nil
	case "int32":
		v, ok := value.(int32)
		if !ok {
			return nil, fmt.Errorf("value is not int32")
		}
		return encodeUint32(uint32(v), layout), nil
	case "int64":
		v, ok := value.(int64)
		if !ok {
			return nil, fmt.Errorf("value is not int64")
		}
		return encodeUint64(uint64(v), layout), nil
	case "float32":
		v, ok := value.(float32)
		if !ok {
			return nil, fmt.Errorf("value is not float32")
		}
		return encodeFloat32(v, layout), nil
	case "float64":
		v, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("value is not float64")
		}
		return encodeFloat64(v, layout), nil
	default:
		return nil, fmt.Errorf("data type %s is not writable", dataType)
	}
}

func encodeUint16(value uint16, layout string) []byte {
	if layout == "BADC" || layout == "DCBA" {
		return binutil.Uint16ToBytesLittleEndian(value)
	}
	return binutil.Uint16ToBytesBigEndian(value)
}

func encodeUint32(value uint32, layout string) []byte {
	switch layout {
	case "ABCD":
		return binutil.Uint32ToBytesBigEndian(value)
	case "BADC":
		return binutil.Uint32ToBytesBigEndianByteSwap(value)
	case "CDAB":
		return binutil.Uint32ToBytesLittleEndianByteSwap(value)
	default:
		return binutil.Uint32ToBytesLittleEndian(value)
	}
}

func encodeUint64(value uint64, layout string) []byte {
	switch layout {
	case "ABCD":
		return binutil.Uint64ToBytesBigEndian(value)
	case "BADC":
		return binutil.Uint64ToBytesBigEndianByteSwap(value)
	case "CDAB":
		return binutil.Uint64ToBytesLittleEndianByteSwap(value)
	default:
		return binutil.Uint64ToBytesLittleEndian(value)
	}
}

func encodeFloat32(value float32, layout string) []byte {
	switch layout {
	case "ABCD":
		return binutil.Float32ToBytesBigEndian(value)
	case "BADC":
		return binutil.Float32ToBytesBigEndianByteSwap(value)
	case "CDAB":
		return binutil.Float32ToBytesLittleEndianByteSwap(value)
	default:
		return binutil.Float32ToBytesLittleEndian(value)
	}
}

func encodeFloat64(value float64, layout string) []byte {
	switch layout {
	case "ABCD":
		return binutil.Float64ToBytesBigEndian(value)
	case "BADC":
		return binutil.Float64ToBytesBigEndianByteSwap(value)
	case "CDAB":
		return binutil.Float64ToBytesLittleEndianByteSwap(value)
	default:
		return binutil.Float64ToBytesLittleEndian(value)
	}
}

// PDU returns the Modbus PDU.
func (w *WriteFrame) PDU() []byte {
	pdu := []byte{w.Function}
	pdu = append(pdu, binutil.Uint16ToBytesBigEndian(w.Start)...)
	switch w.Function {
	case 5, 6:
		pdu = append(pdu, w.Data...)
	case 16:
		pdu = append(pdu, binutil.Uint16ToBytesBigEndian(w.Quantity)...)
		pdu = append(pdu, byte(len(w.Data)))
		pdu = append(pdu, w.Data...)
	}
	return pdu
}

// BuildTCPWriteRequest builds a Modbus TCP write request.
func BuildTCPWriteRequest(slave uint8, transaction uint16, pdu []byte) []byte {
	request := make([]byte, 7+len(pdu))
	binutil.WriteUint16BigEndian(request, transaction)
	binutil.WriteUint16BigEndian(request[2:], 0)
	binutil.WriteUint16BigEndian(request[4:], uint16(1+len(pdu)))
	request[6] = slave
	copy(request[7:], pdu)
	return request
}

// ValidateTCPWriteResponse validates a Modbus TCP write response.
func ValidateTCPWriteResponse(response []byte, transaction uint16, slave, function uint8) error {
	if len(response) < 12 {
		return fmt.Errorf("modbusTcp write response is too short")
	}
	if binutil.ParseUint16BigEndian(response) != transaction {
		return fmt.Errorf("modbusTcp write transaction mismatch")
	}
	if binutil.ParseUint16BigEndian(response[2:]) != 0 {
		return fmt.Errorf("modbusTcp write protocol id mismatch")
	}
	if int(binutil.ParseUint16BigEndian(response[4:]))+6 != len(response) {
		return fmt.Errorf("modbusTcp write length mismatch")
	}
	if response[6] != slave {
		return fmt.Errorf("modbusTcp write slave mismatch")
	}
	if response[7]&0x80 != 0 {
		return fmt.Errorf("modbusTcp write function error %d", response[7]&0x7f)
	}
	if response[7] != function {
		return fmt.Errorf("modbusTcp write function mismatch")
	}
	if len(response) != 12 {
		return fmt.Errorf("modbusTcp write response length mismatch")
	}
	return nil
}

// BuildRTUWriteRequest builds a Modbus RTU write request.
func BuildRTUWriteRequest(write *WriteFrame) []byte {
	message := append([]byte{write.Slave}, write.PDU()...)
	crc := make([]byte, 2)
	binutil.WriteUint16BigEndian(crc, crcutil.CheckCrc16sum(message))
	return append(message, crc...)
}

// ValidateRTUWriteResponse validates a Modbus RTU write response.
func ValidateRTUWriteResponse(response []byte, slave, function uint8) error {
	if len(response) != 8 {
		return fmt.Errorf("modbusRtu write response length mismatch")
	}
	crc := binutil.ParseUint16BigEndian(response[len(response)-2:])
	if crc != crcutil.CheckCrc16sum(response[:len(response)-2]) {
		return fmt.Errorf("modbusRtu write crc mismatch")
	}
	if response[0] != slave {
		return fmt.Errorf("modbusRtu write slave mismatch")
	}
	if response[1]&0x80 != 0 {
		return fmt.Errorf("modbusRtu write function error %d", response[1]&0x7f)
	}
	if response[1] != function {
		return fmt.Errorf("modbusRtu write function mismatch")
	}
	return nil
}
