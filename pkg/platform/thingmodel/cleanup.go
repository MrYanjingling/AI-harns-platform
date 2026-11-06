package thingmodel

import "context"

// ThingRef identifies a thing removed with its model.
type ThingRef struct {
	ID           string
	Token        string
	MQTTClientID string
}

// DeleteThingModelResult contains the data needed for asynchronous cleanup.
type DeleteThingModelResult struct {
	ModelKey      string
	DeletedThings int64
	ThingRefs     []ThingRef
}

// ThingCacheCleaner removes cached identities for deleted things.
type ThingCacheCleaner interface {
	CleanupDeletedThingCaches(ctx context.Context, thingIDs, tokens []string)
}

// ThingSessionKicker disconnects MQTT clients for deleted things.
type ThingSessionKicker interface {
	KickMQTTClients(ctx context.Context, clientIDs []string) error
}

// ThingModelRelationCleaner removes rule and alarm configuration relations by
// thing model key.
type ThingModelRelationCleaner interface {
	CleanupThingModelRelations(ctx context.Context, modelKey string) error
}

// ThingModelTimeSeriesDropper deletes model-scoped time series tables.
type ThingModelTimeSeriesDropper interface {
	DropThingModelTables(ctx context.Context, modelKey string) error
}
