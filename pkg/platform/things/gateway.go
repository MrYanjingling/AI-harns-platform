package things

import (
	"context"
	"errors"
)

// ErrGatewayCollectionResolverUnavailable is returned until gateway collection
// mappings are configured.
var ErrGatewayCollectionResolverUnavailable = errors.New("gateway collection resolver is not configured")

// GatewayPropertyBinding maps one gateway collection to one thing model.
type GatewayPropertyBinding struct {
	ThingID       string
	ThingModelID  string
	ThingModelKey string
	DeviceKey     string
	DeviceName    string
	// PropertyMapping maps raw gateway keys to model identifiers.
	// Empty means the reported keys are already model identifiers.
	PropertyMapping map[string]string
}

// GatewayCollectionResolver resolves one collectionId to one or more devices.
type GatewayCollectionResolver interface {
	Resolve(ctx context.Context, gatewayType, gatewayID, collectionID string) ([]GatewayPropertyBinding, error)
}

// UnavailableGatewayCollectionResolver is the placeholder resolver.
type UnavailableGatewayCollectionResolver struct{}

func (UnavailableGatewayCollectionResolver) Resolve(ctx context.Context, gatewayType, gatewayID, collectionID string) ([]GatewayPropertyBinding, error) {
	return nil, ErrGatewayCollectionResolverUnavailable
}
