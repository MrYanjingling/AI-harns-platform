package v1

import "encoding/json"

type CreateConnectorReq struct {
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	PushConfig json.RawMessage `json:"pushConfig"`
}

type UpdateConnectorReq struct {
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	PushConfig json.RawMessage `json:"pushConfig"`
}

type CreatePushGroupReq struct {
	Name                         string   `json:"name"`
	ConnectorID                  string   `json:"connectorId"`
	DataTopic                    string   `json:"dataTopic"`
	CommandTopic                 string   `json:"commandTopic"`
	PushIntervalMs               int64    `json:"pushIntervalMs"`
	AllCollections               bool     `json:"allCollections"`
	CollectionIDs                []string `json:"collectionIds"`
	MaxCollectionsPerPushMessage *int     `json:"maxCollectionsPerPushMessage"`
}

type UpdatePushGroupReq = CreatePushGroupReq
