package options

import (
	"context"
	"github.com/spf13/pflag"
	"harns-platform/cmd/gateway/config"
	"harns-platform/pkg/gateway"
	"harns-platform/pkg/gateway/collection/modbusrtu"
	"harns-platform/pkg/gateway/collection/modbusrtuovertcp"
	"harns-platform/pkg/gateway/collection/modbustcp"
	"harns-platform/pkg/gateway/collection/opcua"
	"harns-platform/pkg/gateway/collection/s7"
	"harns-platform/pkg/gateway/collectionmgr"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/pushmgr"
	"harns-platform/pkg/gateway/sqlite"
	"harns-platform/pkg/generic"
	baseoptions "harns-platform/pkg/generic/options"
	"time"
)

type SQLiteOptions struct {
	Path string `json:"path"`
}

type Options struct {
	Port         string               `json:"port"`
	Wait         baseoptions.Duration `json:"graceful-timeout"`
	CertFile     string               `json:"cert-file"`
	KeyFile      string               `json:"key-file"`
	MaxBodyBytes int64                `json:"max-body-bytes"`
	UIDir        string               `json:"ui-dir"`
	CORS         generic.CORSConfig   `json:"cors"`
	SQLite       SQLiteOptions        `json:"sqlite"`
	baseoptions.BaseOptions
	// logs.BaseOptions
}

const (
	_defaultPort               = "32200"
	_defaultWait               = baseoptions.Duration(15 * time.Second)
	_defaultConfigFile         = "/etc/gateway/config.yaml"
	_defaultMaxBodyBytes int64 = 10 << 20
	_defaultSQLitePath         = "data/gateway.db"
	_defaultUIDir              = "ui/gateway-ui/dist"
)

func NewDefaultOptions() *Options {
	base := baseoptions.NewDefaultBaseOptions()
	base.ConfigFile = _defaultConfigFile
	return &Options{
		Port:         _defaultPort,
		Wait:         _defaultWait,
		BaseOptions:  base,
		CertFile:     "",
		KeyFile:      "",
		MaxBodyBytes: _defaultMaxBodyBytes,
		UIDir:        _defaultUIDir,
		CORS:         generic.DefaultCORSConfig(),
		SQLite:       SQLiteOptions{Path: _defaultSQLitePath},
		// BaseOptions: logs.NewOptions(),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port exposed")
	baseoptions.DurationVar(fs, &o.Wait, "graceful-timeout", "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	fs.StringVarP(&o.CertFile, "cert-file", "", o.CertFile, "The Cert file")
	fs.StringVarP(&o.KeyFile, "key-file", "", o.KeyFile, "The Key file")
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{MaxBodyBytes: o.MaxBodyBytes, UIDir: o.UIDir, CORS: o.CORS}

	db, err := sqlite.Open(o.SQLite.Path)
	if err != nil {
		return nil, err
	}
	gatewayStore := gateway.NewStore(db)
	if err := gatewayStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	gatewayService := gateway.NewService(gatewayStore)
	gatewayMgr := gateway.NewGatewayManager(gatewayService, stopCh)
	if err := gatewayMgr.Init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	c.GatewayMgr = gatewayMgr

	driverStore := drivermgr.NewStore(db)
	if err := driverStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	c.DB = db
	c.DriverMgr = drivermgr.NewService(driverStore)
	collectionStore := collectionmgr.NewStore(db)
	if err := collectionStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	collectionMgr, err := collectionmgr.NewService(collectionStore,
		&modbustcp.CreateHandler{},
		&modbusrtu.CreateHandler{},
		&modbusrtuovertcp.CreateHandler{},
		&opcua.CreateHandler{},
		&s7.CreateHandler{},
		s7.NewS7200CreateHandler(),
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	runtimeManager, err := collectionmgr.NewRuntimeManager(
		&modbustcp.CollectorFactory{},
		&modbusrtu.CollectorFactory{},
		&modbusrtuovertcp.CollectorFactory{},
		&opcua.CollectorFactory{},
		&s7.CollectorFactory{},
		s7.NewS7200CollectorFactory(),
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	collectionMgr.SetRuntimeManager(runtimeManager)
	if err := collectionMgr.StartRuntime(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	c.CollectionMgr = collectionMgr

	gatewayMeta, err := gatewayMgr.GetGatewayMeta()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	pushStore := pushmgr.NewStore(db)
	if err := pushStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	pushSource := newPushCollectionSource(runtimeManager, collectionMgr)
	pushRuntime, err := pushmgr.NewRuntimeManager(pushSource, gatewayMeta.ID, pushmgr.NewMQTTFactory())
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	pushService := pushmgr.NewService(pushStore, pushSource, pushRuntime)
	if err := pushService.StartRuntime(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	c.PushMgr = pushService

	c.KeyFile = o.KeyFile
	c.CertFile = o.CertFile
	return c, nil
}
