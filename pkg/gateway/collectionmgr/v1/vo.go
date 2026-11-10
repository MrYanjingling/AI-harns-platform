package v1

import (
	"time"

	"harns-platform/pkg/gateway/collection"
)

// CollectionVo is one collection summary.
type CollectionVo struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Version          string    `json:"version"`
	DriverType       string    `json:"driverType"`
	DriverModel      string    `json:"driverModel"`
	CollectStatus    string    `json:"collectStatus"`
	CollectorCycleMs uint64    `json:"collectorCycleMs"`
	ModTime          time.Time `json:"modTime"`
}

// AddVariablesVo is the batch variable add result.
type AddVariablesVo struct {
	Added     int                    `json:"added"`
	Variables []*collection.Variable `json:"variables"`
}

// VariableVo is one collection variable with its runtime value.
type VariableVo struct {
	Name         string                `json:"name"`
	Type         string                `json:"type"`
	DataType     string                `json:"dataType"`
	Address      string                `json:"address"`
	AccessMode   collection.AccessMode `json:"accessMode"`
	IsCalculated bool                  `json:"isCalculated"`
	Expression   string                `json:"expression,omitempty"`
	Offset       float64               `json:"offset"`
	Rate         float64               `json:"rate"`
	Precision    int                   `json:"precision"`
	Value        any                   `json:"value"`
}
