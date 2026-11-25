package config

import (
	"database/sql"

	"harns-platform/pkg/gateway/ai/assistant"
	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/aimgr"
	"harns-platform/pkg/gateway/collectionmgr"
	"harns-platform/pkg/gateway/device"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/gatewaymgr"
	"harns-platform/pkg/gateway/licensemgr"
	"harns-platform/pkg/gateway/pushmgr"
	"harns-platform/pkg/gateway/simulatormgr"
	"harns-platform/pkg/gateway/updatemgr"
	"harns-platform/pkg/gateway/usermgr"
	"harns-platform/pkg/generic"
)

type Config struct {
	DeviceMgr     *device.Manager
	AIMgr         *aimgr.Service
	Assistant     *assistant.Service
	ToolRegistry  *tool.Registry
	GatewayMgr    *gatewaymgr.Manager
	DriverMgr     *drivermgr.Service
	CollectionMgr *collectionmgr.Service
	PushMgr       *pushmgr.Service
	SimulatorMgr  *simulatormgr.Service
	UserMgr       *usermgr.Service
	LicenseMgr    *licensemgr.Service
	UpdateMgr     *updatemgr.Service
	DB            *sql.DB
	CertFile      string
	KeyFile       string
	MaxBodyBytes  int64
	UIDir         string
	LogFilePath   string
	CORS          generic.CORSConfig
}
