package v1

import (
	"encoding/json"
	"time"
)

type ConnectorVo struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	PushConfig  json.RawMessage `json:"pushConfig"`
	Status      string          `json:"status"`
	Version     int64           `json:"version"`
	CreatedTime time.Time       `json:"createdTime"`
	UpdatedTime time.Time       `json:"updatedTime"`
}

type PushGroupVo struct {
	ID                           string    `json:"id"`
	Name                         string    `json:"name"`
	ConnectorID                  string    `json:"connectorId"`
	PushIntervalMs               int64     `json:"pushIntervalMs"`
	AllCollections               bool      `json:"allCollections"`
	CollectionIDs                []string  `json:"collectionIds"`
	MaxCollectionsPerPushMessage int       `json:"maxCollectionsPerPushMessage"`
	Version                      int64     `json:"version"`
	CreatedTime                  time.Time `json:"createdTime"`
	UpdatedTime                  time.Time `json:"updatedTime"`
}
