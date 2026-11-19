package collectionmgr

import (
	"fmt"
	"sync"

	"harns-platform/pkg/gateway/collection"
)

type driverKey struct {
	driverType  string
	driverModel string
}

type createRegistry struct {
	mu       sync.RWMutex
	handlers map[driverKey]collection.CreateHandler
}

func newCreateRegistry(handlers ...collection.CreateHandler) (*createRegistry, error) {
	registry := &createRegistry{handlers: make(map[driverKey]collection.CreateHandler, len(handlers))}
	for _, handler := range handlers {
		if err := registry.register(handler); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *createRegistry) register(handler collection.CreateHandler) error {
	if handler == nil {
		return fmt.Errorf("collection create handler is nil")
	}
	key := driverKey{driverType: handler.DriverType(), driverModel: handler.DriverModel()}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[key]; exists {
		return fmt.Errorf("duplicate collection create handler %s/%s", key.driverType, key.driverModel)
	}
	r.handlers[key] = handler
	return nil
}

func (r *createRegistry) resolve(driverType, driverModel string) (collection.CreateHandler, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[driverKey{driverType: driverType, driverModel: driverModel}]
	return handler, ok
}
