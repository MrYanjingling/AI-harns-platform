package collection

// AccessMode describes how a variable can be used.
type AccessMode string

const (
	AccessModeRead      AccessMode = "r"
	AccessModeWrite     AccessMode = "w"
	AccessModeReadWrite AccessMode = "rw"
)

// Variable is the protocol-independent collection point definition.
type Variable struct {
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	DataType   string     `json:"dataType"`
	Address    string     `json:"address"`
	Bits       *uint8     `json:"bits"`
	AccessMode AccessMode `json:"accessMode"`

	IsCalculated bool   `json:"isCalculated"`
	Expression   string `json:"expression,omitempty"`

	Offset    *float64 `json:"offset"`
	Rate      *float64 `json:"rate"`
	Precision *int     `json:"precision"`

	Value any `json:"-"`
}

// BitsValue returns the configured bit index, defaulting to zero.
func (v *Variable) BitsValue() uint8 {
	if v == nil || v.Bits == nil {
		return 0
	}
	return *v.Bits
}

// RateValue returns the configured rate, defaulting to one.
func (v *Variable) RateValue() float64 {
	if v == nil || v.Rate == nil {
		return 1
	}
	return *v.Rate
}

// OffsetValue returns the configured offset, defaulting to zero.
func (v *Variable) OffsetValue() float64 {
	if v == nil || v.Offset == nil {
		return 0
	}
	return *v.Offset
}

// PrecisionValue returns the configured precision, defaulting to zero.
func (v *Variable) PrecisionValue() int {
	if v == nil || v.Precision == nil {
		return 0
	}
	return *v.Precision
}

// ValidAccessMode reports whether mode is a supported variable access mode.
func ValidAccessMode(mode AccessMode) bool {
	switch mode {
	case AccessModeRead, AccessModeWrite, AccessModeReadWrite:
		return true
	default:
		return false
	}
}
