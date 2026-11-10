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
	AccessMode AccessMode `json:"accessMode"`

	IsCalculated bool   `json:"isCalculated"`
	Expression   string `json:"expression,omitempty"`

	Offset    float64 `json:"offset"`
	Rate      float64 `json:"rate"`
	Precision int     `json:"precision"`

	Value any `json:"-"`
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
