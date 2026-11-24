package iec61850

import (
	"fmt"
	"strings"

	iec "github.com/otfabric/go-iec61850"
)

type Address struct {
	Reference iec.Ref
}

func ParseAddress(value, defaultFC string) (Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Address{}, fmt.Errorf("IEC 61850 address is empty")
	}
	reference, err := iec.ParseRef(value)
	if err != nil {
		return Address{}, err
	}
	if reference.FC == "" {
		fc := strings.ToUpper(strings.TrimSpace(defaultFC))
		if fc == "" {
			fc = string(iec.FCST)
		}
		reference.FC = iec.FunctionalConstraint(fc)
	}
	if err := reference.Validate(); err != nil {
		return Address{}, err
	}
	return Address{Reference: reference}, nil
}
