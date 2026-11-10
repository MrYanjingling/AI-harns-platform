package collectionmgr

import (
	"encoding/json"
	"time"

	"harns-platform/pkg/gateway/collection"
)

// CollectionRecord is the persisted representation of a collection.
type CollectionRecord struct {
	ID               string
	Name             string
	Version          string
	DriverType       string
	DriverModel      string
	CollectStatus    string
	CollectorCycleMs uint64
	ProtocolConfig   json.RawMessage
	Variables        []*collection.Variable
	CreatedTime      time.Time
	UpdatedTime      time.Time
}
