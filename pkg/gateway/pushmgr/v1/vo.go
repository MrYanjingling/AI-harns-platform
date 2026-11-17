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
	RunState    string          `json:"runState"`
	GroupCount  int64           `json:"groupCount"`
	Status      string          `json:"status"`
	LastError   string          `json:"lastError,omitempty"`
	Version     int64           `json:"version"`
	CreatedTime time.Time       `json:"createdTime"`
	UpdatedTime time.Time       `json:"updatedTime"`
}

type PushGroupVo struct {
	ID                           string     `json:"id"`
	Name                         string     `json:"name"`
	ConnectorID                  string     `json:"connectorId"`
	DataTopic                    string     `json:"dataTopic"`
	CommandTopic                 string     `json:"commandTopic"`
	RunState                     string     `json:"runState"`
	Status                       string     `json:"status"`
	LastError                    string     `json:"lastError,omitempty"`
	LastPushTime                 *time.Time `json:"lastPushTime,omitempty"`
	PushIntervalMs               int64      `json:"pushIntervalMs"`
	AllCollections               bool       `json:"allCollections"`
	CollectionIDs                []string   `json:"collectionIds"`
	MaxCollectionsPerPushMessage int        `json:"maxCollectionsPerPushMessage"`
	Version                      int64      `json:"version"`
	CreatedTime                  time.Time  `json:"createdTime"`
	UpdatedTime                  time.Time  `json:"updatedTime"`
}
