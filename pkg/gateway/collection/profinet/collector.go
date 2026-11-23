package profinet

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/s7"
)

type CollectorFactory struct {
	inner *s7.CollectorFactory
}

func NewCollectorFactory() *CollectorFactory {
	return &CollectorFactory{inner: s7.NewCollectorFactoryFor(DriverType, DriverModel)}
}

func (f *CollectorFactory) DriverType() string  { return DriverType }
func (f *CollectorFactory) DriverModel() string { return DriverModel }

func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	return f.inner.NewCollector(meta, config)
}

var _ collection.CollectorFactory = (*CollectorFactory)(nil)
