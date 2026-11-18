package options

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"harns-platform/cmd/gateway/config"
	"harns-platform/pkg/gateway/collection/fins"
	"harns-platform/pkg/gateway/collection/hostlinkcmode"
	"harns-platform/pkg/gateway/collection/modbusrtu"
	"harns-platform/pkg/gateway/collection/modbusrtuovertcp"
	"harns-platform/pkg/gateway/collection/modbustcp"
	"harns-platform/pkg/gateway/collection/omroncip"
	"harns-platform/pkg/gateway/collection/opcua"
	"harns-platform/pkg/gateway/collection/s7"
	"harns-platform/pkg/gateway/collectionmgr"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/gatewaymgr"
	"harns-platform/pkg/gateway/licensemgr"
	"harns-platform/pkg/gateway/pushmgr"
	"harns-platform/pkg/gateway/sqlite"
	"harns-platform/pkg/gateway/usermgr"
	"harns-platform/pkg/generic"
	baseoptions "harns-platform/pkg/generic/options"
	"harns-platform/pkg/logging"
	"time"
)

type SQLiteOptions struct {
	Path string `json:"path"`
}

type LicenseOptions struct {
	PublicKeyFile string `json:"public-key-file"`
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
	License      LicenseOptions       `json:"license"`
	baseoptions.BaseOptions
	// logs.BaseOptions
}

const (
	_defaultPort                   = "32200"
	_defaultWait                   = baseoptions.Duration(15 * time.Second)
	_defaultConfigFile             = "/etc/gateway/config.yaml"
	_defaultMaxBodyBytes     int64 = 10 << 20
	_defaultSQLitePath             = "data/gateway.db"
	_defaultUIDir                  = "ui/gateway-ui/dist"
	_defaultLicensePublicKey       = "etc/gateway/license-public.pem"
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
		License:      LicenseOptions{PublicKeyFile: _defaultLicensePublicKey},
		// BaseOptions: logs.NewOptions(),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port exposed")
	baseoptions.DurationVar(fs, &o.Wait, "graceful-timeout", "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	fs.StringVarP(&o.CertFile, "cert-file", "", o.CertFile, "The Cert file")
	fs.StringVarP(&o.KeyFile, "key-file", "", o.KeyFile, "The Key file")
	fs.StringVar(&o.License.PublicKeyFile, "license-public-key-file", o.License.PublicKeyFile, "License JWT verification public key PEM file")
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{
		MaxBodyBytes: o.MaxBodyBytes,
		UIDir:        o.UIDir,
		LogFilePath:  o.Logging.FilePath(),
		CORS:         o.CORS,
	}

	db, err := sqlite.Open(o.SQLite.Path)
	if err != nil {
		return nil, err
	}
	gatewayStore := gatewaymgr.NewStore(db)
	if err := gatewayStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	gatewayService := gatewaymgr.NewService(gatewayStore)
	gatewayMgr := gatewaymgr.NewGatewayManager(
		gatewayService,
		stopCh,
		gatewaymgr.WithStoragePath(resolveStorageDirectory(o.SQLite.Path, c.LogFilePath)),
	)
	if err := gatewayMgr.Init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	c.GatewayMgr = gatewayMgr

	licenseStore := licensemgr.NewStore(db)
	if err := licenseStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	licenseService := licensemgr.NewService(licenseStore, func() (string, string, error) {
		meta, err := gatewayMgr.GetGatewayMeta()
		if err != nil {
			return "", "", err
		}
		if meta == nil {
			return "", "", fmt.Errorf("gateway metadata is not initialized")
		}
		return meta.ID, meta.Secret, nil
	}, o.License.PublicKeyFile)
	if err := licenseService.Load(context.Background()); err != nil {
		logging.Warnf("Failed to load persisted license: %v", err)
	}
	c.LicenseMgr = licenseService

	driverStore := drivermgr.NewStore(db)
	if err := driverStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	c.DB = db
	pluginDir := ""
	if storageDir := resolveStorageDirectory(o.SQLite.Path, c.LogFilePath); storageDir != "" {
		pluginDir = filepath.Join(storageDir, "plugins")
	}
	c.DriverMgr = drivermgr.NewService(driverStore, pluginDir)
	userStore := usermgr.NewStore(db)
	if err := userStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	userService := usermgr.NewService(userStore)
	if err := userService.BootstrapSuperuser(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	c.UserMgr = userService
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
		&fins.CreateHandler{},
		&hostlinkcmode.CreateHandler{},
		&omroncip.CreateHandler{},
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
		&fins.CollectorFactory{},
		&hostlinkcmode.CollectorFactory{},
		&omroncip.CollectorFactory{},
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	collectionMgr.SetRuntimeManager(runtimeManager)
	collectionMgr.SetVariableLimitProvider(licenseService)
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
	pushRuntime, err := pushmgr.NewRuntimeManager(pushSource, gatewayMeta.ID, pushmgr.NewMQTTFactory(), pushmgr.NewKafkaFactory())
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
	startLicenseRunGuard(stopCh, licenseService, collectionMgr, pushService)

	c.KeyFile = o.KeyFile
	c.CertFile = o.CertFile
	return c, nil
}

func resolveStorageDirectory(dbPath, logFilePath string) string {
	dbDir := absoluteDirectory(dbPath)
	if dbDir != "" && !isRootPath(dbDir) {
		return dbDir
	}
	logDir := absoluteDirectory(logFilePath)
	if logDir != "" && !isRootPath(logDir) {
		return logDir
	}
	return ""
}

func absoluteDirectory(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == ":memory:" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return filepath.Dir(abs)
}

func isRootPath(path string) bool {
	cleaned := filepath.Clean(path)
	return cleaned == filepath.VolumeName(cleaned)+string(filepath.Separator)
}
