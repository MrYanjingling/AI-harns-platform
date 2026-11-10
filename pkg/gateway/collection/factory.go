package collection

import "encoding/json"

// CollectorFactory creates a collector for one driver model.
type CollectorFactory interface {
	DriverType() string
	DriverModel() string
	NewCollector(meta *CollectionMeta, config json.RawMessage) (Collector, error)
}
