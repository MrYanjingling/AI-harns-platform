package modbustcp

import (
	"harns-platform/pkg/gateway/collection"
)

func compileCalculations(variables []*collection.Variable) ([]*collection.Calculation, error) {
	return collection.CompileCalculations(variables)
}

func (c *ModbusTcpCollection) calculate() error {
	runtime := c.runtime
	if runtime == nil {
		return nil
	}
	return collection.Calculate(c.GetCollectionMeta(), runtime.calcs)
}
