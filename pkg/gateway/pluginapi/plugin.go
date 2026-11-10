// Package pluginapi defines the shared Gateway plugin ABI.
package pluginapi

import (
	"encoding/json"

	"harns-platform/pkg/gateway/collection"
)

// Descriptor identifies one plugin.
type Descriptor struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	ABIVersion string `json:"abiVersion"`
}

// Plugin is the symbol exported by a Gateway plugin.
type Plugin interface {
	Descriptor() Descriptor
	NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error)
}
