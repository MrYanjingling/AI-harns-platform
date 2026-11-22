package collection

import "harns-platform/pkg/logging"

// ApplyCycleFailure updates the collection status after a failed cycle.
// It returns true when polling should continue, or false when the connection
// was lost and the collection should wait for the runtime retry loop.
func ApplyCycleFailure(c Collector, err error, succeeded, connectionFailures int, protocolFailure bool) bool {
	if succeeded == 0 && connectionFailures > 0 && !protocolFailure {
		return MarkConnectionLost(c, err)
	}
	c.SetCollectStatus(CollectStatusCollectingError)
	return true
}

// MarkConnectionLost logs the transport failure and moves the collection to
// unconnected so the runtime retry loop can reconnect with backoff. It returns
// false for callers that stop their polling loop.
func MarkConnectionLost(c Collector, err error) bool {
	if c != nil {
		meta := c.GetCollectionMeta()
		if meta != nil {
			logging.ErrorS(err, "Collection connection lost",
				"collectionId", meta.GetID(),
				"driverType", meta.GetDriverType(),
				"driverModel", meta.GetDriverModel(),
			)
		}
		c.SetCollectStatus(CollectStatusUnconnected)
	}
	return false
}
