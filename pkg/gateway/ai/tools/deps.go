package tools

import (
	"harns-platform/pkg/gateway/collectionmgr"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/gatewaymgr"
	"harns-platform/pkg/gateway/licensemgr"
	"harns-platform/pkg/gateway/pushmgr"
	"harns-platform/pkg/gateway/usermgr"
)

// Deps contains the services exposed to the AI assistant.
type Deps struct {
	Collections *collectionmgr.Service
	Drivers     *drivermgr.Service
	Push        *pushmgr.Service
	Users       *usermgr.Service
	License     *licensemgr.Service
	Gateway     *gatewaymgr.Manager
}
