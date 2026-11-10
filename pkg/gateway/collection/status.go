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
