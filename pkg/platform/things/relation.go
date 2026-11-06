package things

import "context"

// DeletedThingRef identifies a thing removed during a batch delete.
type DeletedThingRef struct {
	ID           string
	Token        string
	MQTTClientID string
}

// ThingRelationCleaner removes rule and alarm configuration relations for
// deleted things. Implementations run asynchronously after the thing delete
// commits, so failures may leave orphan relations for periodic cleanup.
type ThingRelationCleaner interface {
	CleanupThingRelations(ctx context.Context, thingIDs []string) error
}
