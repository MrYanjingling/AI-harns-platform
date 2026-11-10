package collectionmgr

import (
	"fmt"

	"harns-platform/pkg/gateway/collection"
)

type driverKey struct {
	driverType  string
	driverModel string
}

type createRegistry struct {
	handlers map[driverKey]collection.CreateHandler
}

func newCreateRegistry(handlers ...collection.CreateHandler) (*createRegistry, error) {
	registry := &createRegistry{handlers: make(map[driverKey]collection.CreateHandler, len(handlers))}
	for _, handler := range handlers {
		if handler == nil {
			return nil, fmt.Errorf("collection create handler is nil")
		}
		key := driverKey{driverType: handler.DriverType(), driverModel: handler.DriverModel()}
		if _, exists := registry.handlers[key]; exists {
			return nil, fmt.Errorf("duplicate collection create handler %s/%s", key.driverType, key.driverModel)
		}
		registry.handlers[key] = handler
	}
	return registry, nil
}

func (r *createRegistry) resolve(driverType, driverModel string) (collection.CreateHandler, bool) {
	if r == nil {
		return nil, false
	}
	handler, ok := r.handlers[driverKey{driverType: driverType, driverModel: driverModel}]
	return handler, ok
}
