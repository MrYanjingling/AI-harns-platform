package pushmgr

import (
	"context"
	"encoding/json"
	"time"
)

const (
	ConnectorTypeMQTT               = "mqtt"
	ConnectorStatusRunning          = "running"
	ConnectorStatusStopped          = "stopped"
	DefaultPushIntervalMs           = 1000
	DefaultMaxCollectionsPerMessage = 10
	DefaultMQTTQoS                  = 1
	DefaultMQTTKeepAlive            = 60
)

type ConnectorRecord struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	PushConfig  json.RawMessage `json:"pushConfig"`
	Version     int64           `json:"version"`
	CreatedTime time.Time       `json:"createdTime"`
	UpdatedTime time.Time       `json:"updatedTime"`
}

type PushGroup struct {
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

type Action struct {
	Name  string
	Value string
}

type CollectionSnapshot struct {
	CollectionID string
	Status       string
	Time         time.Time
	Values       map[string]any
}

type CollectionSource interface {
	Snapshots(ctx context.Context, all bool, collectionIDs []string) ([]CollectionSnapshot, error)
	Exists(ctx context.Context, collectionID string) (bool, error)
	DeliverActions(ctx context.Context, collectionID string, actions []Action) error
}
