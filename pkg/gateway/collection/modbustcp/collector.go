package modbustcp

import (
	"context"
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

// CollectorFactory creates placeholder Modbus TCP collectors.
type CollectorFactory struct{}

// DriverType returns the Modbus driver type.
func (f *CollectorFactory) DriverType() string { return DriverType }

// DriverModel returns the Modbus TCP driver model.
func (f *CollectorFactory) DriverModel() string { return DriverModel }

// NewCollector creates a Modbus TCP collector placeholder.
func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &ModbusTcpCollection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

// Connect is intentionally not implemented yet.
func (c *ModbusTcpCollection) Connect(ctx context.Context) error {
	c.SetCollectStatus(collection.CollectStatusError)
	return collection.ErrNotImplemented
}

// Start is intentionally not implemented yet.
func (c *ModbusTcpCollection) Start(ctx context.Context) error {
	c.SetCollectStatus(collection.CollectStatusError)
	return collection.ErrNotImplemented
}

// Stop marks the placeholder collector stopped.
func (c *ModbusTcpCollection) Stop(ctx context.Context) error {
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

var _ collection.Collector = (*ModbusTcpCollection)(nil)
var _ collection.CollectorFactory = (*CollectorFactory)(nil)
