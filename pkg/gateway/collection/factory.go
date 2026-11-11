package collection

import "encoding/json"

// CollectorFactory creates a collector for one driver model.
// Implementations that retain meta must index it themselves; implementations
// that copy it must call SetCollectionMeta, which rebuilds the variable index.
type CollectorFactory interface {
	DriverType() string
	DriverModel() string
	NewCollector(meta *CollectionMeta, config json.RawMessage) (Collector, error)
}
