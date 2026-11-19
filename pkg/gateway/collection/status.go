package collection

// Collection status values shared with the gateway API.
const (
	CollectStatusCollecting      = "collecting"
	CollectStatusCollectingError = "collectingError"
	CollectStatusUnconnected     = "unconnected"
	CollectStatusEmptyVariable   = "emptyVariable"
	CollectStatusStopped         = "stopped"
	CollectStatusError           = "error"
)

// ShouldStopBeforeConnect reports whether the previous runtime is actively running
// and must be stopped before a new connection attempt.
func ShouldStopBeforeConnect(status string) bool {
	return status == CollectStatusCollecting || status == CollectStatusCollectingError
}
