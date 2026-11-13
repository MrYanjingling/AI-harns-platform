package config

import (
	"database/sql"

	"harns-platform/pkg/gateway"
	"harns-platform/pkg/gateway/collectionmgr"
	"harns-platform/pkg/gateway/device"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/pushmgr"
	"harns-platform/pkg/gateway/usermgr"
	"harns-platform/pkg/generic"
)

type Config struct {
	DeviceMgr     *device.Manager
	GatewayMgr    *gateway.Manager
	DriverMgr     *drivermgr.Service
	CollectionMgr *collectionmgr.Service
	PushMgr       *pushmgr.Service
	UserMgr       *usermgr.Service
	DB            *sql.DB
	CertFile      string
	KeyFile       string
	MaxBodyBytes  int64
	UIDir         string
	LogFilePath   string
	CORS          generic.CORSConfig
}
