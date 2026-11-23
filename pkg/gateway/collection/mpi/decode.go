package mpi

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"harns-platform/pkg/gateway/collection"
)

// DecodeValue decodes an S7-style item payload for MPI/PPI variables.
func DecodeValue(data []byte, address Address, variable *collection.Variable) (any, error) {
	return decodeValue(data, address, variable)
}

func decodeValue(data []byte, address parsedAddress, variable *collection.Variable) (any, error) {
	dataType := strings.ToLower(strings.TrimSpace(variable.DataType))
	switch dataType {
	case "bit":
		if len(data) == 0 {
			return nil, fmt.Errorf("short mpi bit response")
		}
		value := int((1 << address.Bit) & data[0])
		if value != 0 {
			return 1, nil
		}
		return 0, nil
	case "bool":
		if len(data) == 0 {
			return nil, fmt.Errorf("short mpi bool response")
		}
		return (1<<address.Bit)&data[0] != 0, nil
	case "int16":
		if len(data) < 2 {
			return nil, fmt.Errorf("short mpi int16 response")
		}
		return int16(binary.BigEndian.Uint16(data)), nil
	case "uint16":
		if len(data) < 2 {
			return nil, fmt.Errorf("short mpi uint16 response")
		}
		return binary.BigEndian.Uint16(data), nil
	case "int32":
		if len(data) < 4 {
			return nil, fmt.Errorf("short mpi int32 response")
		}
		return int32(binary.BigEndian.Uint32(data)), nil
	case "uint32":
		if len(data) < 4 {
			return nil, fmt.Errorf("short mpi uint32 response")
		}
		return binary.BigEndian.Uint32(data), nil
	case "float32":
		if len(data) < 4 {
			return nil, fmt.Errorf("short mpi float32 response")
		}
		return math.Float32frombits(binary.BigEndian.Uint32(data)), nil
	case "int64":
		if len(data) < 8 {
			return nil, fmt.Errorf("short mpi int64 response")
		}
		return int64(binary.BigEndian.Uint64(data)), nil
	case "uint64":
		if len(data) < 8 {
			return nil, fmt.Errorf("short mpi uint64 response")
		}
		return binary.BigEndian.Uint64(data), nil
	case "float64":
		if len(data) < 8 {
			return nil, fmt.Errorf("short mpi float64 response")
		}
		return math.Float64frombits(binary.BigEndian.Uint64(data)), nil
	case "string":
		length := int(address.Length)
		if length > len(data) {
			length = len(data)
		}
		return strings.TrimRight(string(data[:length]), "\x00"), nil
	case "byte_array":
		length := int(address.Length)
		if length > len(data) {
			length = len(data)
		}
		values := make([]int, 0, length)
		for _, value := range data[:length] {
			values = append(values, int(value))
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported mpi data type %q", variable.DataType)
	}
}
