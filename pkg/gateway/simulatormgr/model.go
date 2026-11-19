package simulatormgr

import "errors"

const (
	DefaultPort         = 2502
	DefaultUnitID       = 1
	DefaultMemoryLayout = "CDAB"

	ModeRandom    Mode = "random"
	ModeIncrement Mode = "increment"
	ModeDecrement Mode = "decrement"

	RegisterTypeHolding = "holdingRegister"

	DataTypeInt16 = "int16"
	DataTypeInt32 = "int32"
	DataTypeFloat = "float"
)

type Mode string

type Point struct {
	Name         string
	Description  string
	RegisterType string
	DataType     string
	Mode         Mode
}

var (
	ErrPointExists   = errors.New("simulator point already exists")
	ErrPointNotFound = errors.New("simulator point not found")
)

func validMode(mode Mode) bool {
	switch mode {
	case ModeRandom, ModeIncrement, ModeDecrement:
		return true
	default:
		return false
	}
}

func validDataType(value string) bool {
	switch value {
	case DataTypeInt16, DataTypeInt32, DataTypeFloat:
		return true
	default:
		return false
	}
}

func dataTypeWords(value string) uint16 {
	switch value {
	case DataTypeInt32, DataTypeFloat:
		return 2
	default:
		return 1
	}
}
