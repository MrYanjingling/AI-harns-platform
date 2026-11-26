package options

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"

	"github.com/spf13/pflag"
	"harns-platform/cmd/gateway/config"
	"harns-platform/pkg/gateway/ai/assistant"
	"harns-platform/pkg/gateway/ai/tools"
	"harns-platform/pkg/gateway/aimgr"
	"harns-platform/pkg/gateway/collection/a1e"
	"harns-platform/pkg/gateway/collection/ab5000"
	"harns-platform/pkg/gateway/collection/ads"
	"harns-platform/pkg/gateway/collection/bacnet"
	"harns-platform/pkg/gateway/collection/cjt188"
	"harns-platform/pkg/gateway/collection/codesys3"
	"harns-platform/pkg/gateway/collection/comli"
	"harns-platform/pkg/gateway/collection/custom"
	"harns-platform/pkg/gateway/collection/df1"
	"harns-platform/pkg/gateway/collection/dlt645"
	"harns-platform/pkg/gateway/collection/dlt698"
	"harns-platform/pkg/gateway/collection/dnp3"
	"harns-platform/pkg/gateway/collection/ethercat"
	"harns-platform/pkg/gateway/collection/ethernetip"
	"harns-platform/pkg/gateway/collection/ethernetip1400"
	"harns-platform/pkg/gateway/collection/ethernetip5500"
	"harns-platform/pkg/gateway/collection/fins"
	"harns-platform/pkg/gateway/collection/focas"
	"harns-platform/pkg/gateway/collection/fx"
	"harns-platform/pkg/gateway/collection/heidenhain"
	"harns-platform/pkg/gateway/collection/hj212"
	"harns-platform/pkg/gateway/collection/hostlinkcmode"
	"harns-platform/pkg/gateway/collection/iec101"
	"harns-platform/pkg/gateway/collection/iec102"
	"harns-platform/pkg/gateway/collection/iec103"
	"harns-platform/pkg/gateway/collection/iec104"
	"harns-platform/pkg/gateway/collection/iec61850"
	"harns-platform/pkg/gateway/collection/knd"
	"harns-platform/pkg/gateway/collection/knx"
	"harns-platform/pkg/gateway/collection/kuka"
	"harns-platform/pkg/gateway/collection/mewtocol"
	"harns-platform/pkg/gateway/collection/modbusascii"
	"harns-platform/pkg/gateway/collection/modbusrtu"
	"harns-platform/pkg/gateway/collection/modbusrtuovertcp"
	"harns-platform/pkg/gateway/collection/modbustcp"
	"harns-platform/pkg/gateway/collection/mpi"
	"harns-platform/pkg/gateway/collection/mtconnect"
	"harns-platform/pkg/gateway/collection/omroncip"
	"harns-platform/pkg/gateway/collection/opcua"
	"harns-platform/pkg/gateway/collection/ppi"
	"harns-platform/pkg/gateway/collection/profinet"
	"harns-platform/pkg/gateway/collection/qna3e"
	"harns-platform/pkg/gateway/collection/qna4e"
	"harns-platform/pkg/gateway/collection/s7"
	"harns-platform/pkg/gateway/collection/snmp"
	"harns-platform/pkg/gateway/collection/srtp"
	"harns-platform/pkg/gateway/collectionmgr"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/gatewaymgr"
	"harns-platform/pkg/gateway/licensemgr"
	"harns-platform/pkg/gateway/pluginmgr"
	"harns-platform/pkg/gateway/pushmgr"
	"harns-platform/pkg/gateway/simulatormgr"
	"harns-platform/pkg/gateway/sqlite"
	"harns-platform/pkg/gateway/updatemgr"
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
	PluginDir    string               `json:"plugin-dir"`
	License      LicenseOptions       `json:"license"`
	baseoptions.BaseOptions
	// logs.BaseOptions
}

const (
	_defaultPort                   = "32200"
	_defaultWait                   = baseoptions.Duration(15 * time.Second)
	_defaultConfigFile             = "/etc/gateway/config.yaml"
	_defaultMaxBodyBytes     int64 = 64 << 20
	_defaultSQLitePath             = "data/gateway.db"
	_defaultPluginDir              = "plugins"
	_defaultUIDir                  = "ui/gateway-ui/dist"
	_defaultGatewayPublicKey       = "etc/gateway/gateway-public.pem"
)

func NewDefaultOptions() *Options {
	base := baseoptions.NewDefaultBaseOptions()
	base.ConfigFile = defaultGatewayConfigFile()
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
		PluginDir:    _defaultPluginDir,
		License:      LicenseOptions{PublicKeyFile: _defaultGatewayPublicKey},
		// BaseOptions: logs.NewOptions(),
	}
}

func defaultGatewayConfigFile() string {
	exe, err := os.Executable()
	if err != nil {
		return _defaultConfigFile
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	candidate := filepath.Join(filepath.Dir(exe), "..", "etc", "config.yaml")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return _defaultConfigFile
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port exposed")
	baseoptions.DurationVar(fs, &o.Wait, "graceful-timeout", "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	fs.StringVarP(&o.CertFile, "cert-file", "", o.CertFile, "The Cert file")
	fs.StringVarP(&o.KeyFile, "key-file", "", o.KeyFile, "The Key file")
	fs.StringVar(&o.License.PublicKeyFile, "license-public-key-file", o.License.PublicKeyFile, "Gateway public key PEM file for license JWT and OTA manifest verification")
	fs.StringVar(&o.PluginDir, "plugin-dir", o.PluginDir, "Custom driver plugin directory")
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

	meta, err := gatewayMgr.GetGatewayMeta()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if meta == nil {
		_ = db.Close()
		return nil, fmt.Errorf("gateway metadata is not initialized")
	}
	aiStore, err := aimgr.NewStore(db, meta.Secret)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := aiStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	c.AIMgr = aimgr.NewService(aiStore)
	documentStore := assistant.NewDocumentStore(db)
	if err := documentStore.Initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

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
	licenseService.SetInitTimeProvider(func(ctx context.Context) (time.Time, error) {
		meta, err := gatewayMgr.GetGatewayMeta()
		if err != nil {
			return time.Time{}, err
		}
		if meta == nil {
			return time.Time{}, fmt.Errorf("gateway metadata is not initialized")
		}
		return meta.InitTime, nil
	})
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
	pluginDir := strings.TrimSpace(o.PluginDir)
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
		&modbusascii.CreateHandler{},
		&modbusrtuovertcp.CreateHandler{},
		&opcua.CreateHandler{},
		&s7.CreateHandler{},
		s7.NewS7200CreateHandler(),
		&mpi.CreateHandler{},
		&ppi.CreateHandler{},
		&fins.CreateHandler{},
		&iec101.CreateHandler{},
		&iec102.CreateHandler{},
		&iec103.CreateHandler{},
		&iec104.CreateHandler{},
		&hostlinkcmode.CreateHandler{},
		&mewtocol.CreateHandler{},
		&omroncip.CreateHandler{},
		&ethernetip.CreateHandler{},
		&ethernetip5500.CreateHandler{},
		&ethernetip1400.CreateHandler{},
		&ethercat.CreateHandler{},
		profinet.NewCreateHandler(),
		&dnp3.CreateHandler{},
		dlt645.NewCreateHandler(drivermgr.ModelDLT6452007),
		dlt645.NewCreateHandler(drivermgr.ModelDLT6451997),
		cjt188.NewCreateHandler(drivermgr.ModelCJT1882004),
		cjt188.NewCreateHandler(drivermgr.ModelCJT1882018),
		&dlt698.CreateHandler{},
		&knx.CreateHandler{},
		&knd.CreateHandler{},
		&heidenhain.CreateHandler{},
		&focas.CreateHandler{},
		&mtconnect.CreateHandler{},
		&kuka.CreateHandler{},
		&codesys3.CreateHandler{},
		&comli.CreateHandler{},
		a1e.NewCreateHandler(),
		qna3e.NewCreateHandler(),
		qna4e.NewCreateHandler(),
		&fx.CreateHandler{},
		&ab5000.CreateHandler{},
		&df1.CreateHandler{},
		&hj212.CreateHandler{},
		&bacnet.CreateHandler{},
		&ads.CreateHandler{},
		&srtp.CreateHandler{},
		&snmp.CreateHandler{},
		&iec61850.CreateHandler{},
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	runtimeManager, err := collectionmgr.NewRuntimeManager(
		&modbustcp.CollectorFactory{},
		&modbusrtu.CollectorFactory{},
		&modbusascii.CollectorFactory{},
		&modbusrtuovertcp.CollectorFactory{},
		&opcua.CollectorFactory{},
		&s7.CollectorFactory{},
		s7.NewS7200CollectorFactory(),
		&mpi.CollectorFactory{},
		&ppi.CollectorFactory{},
		&fins.CollectorFactory{},
		&iec101.CollectorFactory{},
		&iec102.CollectorFactory{},
		&iec103.CollectorFactory{},
		&iec104.CollectorFactory{},
		&hostlinkcmode.CollectorFactory{},
		&mewtocol.CollectorFactory{},
		&omroncip.CollectorFactory{},
		&ethernetip.CollectorFactory{},
		&ethernetip5500.CollectorFactory{},
		&ethernetip1400.CollectorFactory{},
		&ethercat.CollectorFactory{},
		profinet.NewCollectorFactory(),
		&dnp3.CollectorFactory{},
		dlt645.NewCollectorFactory(drivermgr.ModelDLT6452007),
		dlt645.NewCollectorFactory(drivermgr.ModelDLT6451997),
		cjt188.NewCollectorFactory(drivermgr.ModelCJT1882004),
		cjt188.NewCollectorFactory(drivermgr.ModelCJT1882018),
		&dlt698.CollectorFactory{},
		&knx.CollectorFactory{},
		&knd.CollectorFactory{},
		&heidenhain.CollectorFactory{},
		&focas.CollectorFactory{},
		&mtconnect.CollectorFactory{},
		&kuka.CollectorFactory{},
		&codesys3.CollectorFactory{},
		&comli.CollectorFactory{},
		a1e.NewCollectorFactory(),
		qna3e.NewCollectorFactory(),
		qna4e.NewCollectorFactory(),
		&fx.CollectorFactory{},
		&ab5000.CollectorFactory{},
		&df1.CollectorFactory{},
		&hj212.CollectorFactory{},
		&bacnet.CollectorFactory{},
		&ads.CollectorFactory{},
		&srtp.CollectorFactory{},
		&snmp.CollectorFactory{},
		&iec61850.CollectorFactory{},
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	collectionMgr.SetRuntimeManager(runtimeManager)
	collectionMgr.SetVariableLimitProvider(licenseService)
	collectionMgr.SetLicenseStatusProvider(licenseService)
	loadCustomDrivers(context.Background(), c.DriverMgr, pluginDir, collectionMgr, runtimeManager)
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
	pushRuntime.SetReadySendStore(pushStore)
	pushService := pushmgr.NewService(pushStore, pushSource, pushRuntime)
	if err := pushService.StartRuntime(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	collectionMgr.SetReferenceCleaner(pushService)
	c.PushMgr = pushService
	c.SimulatorMgr = simulatormgr.NewService()
	toolRegistry, err := tools.NewRegistry(tools.Deps{
		Collections: collectionMgr,
		Drivers:     c.DriverMgr,
		Push:        pushService,
		Users:       userService,
		License:     licenseService,
		Gateway:     gatewayMgr,
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	c.ToolRegistry = toolRegistry
	c.Assistant = assistant.NewService(c.AIMgr, toolRegistry, documentStore)
	startLicenseRunGuard(stopCh, licenseService, collectionMgr, pushService)

	updateMgr, err := updatemgr.NewService(updatemgr.Options{
		GatewayID:     meta.ID,
		DB:            db,
		Secret:        meta.Secret,
		PublicKeyFile: o.License.PublicKeyFile,
		StagingDir:    resolveOTAStagingDir(o.SQLite.Path),
		UIDir:         c.UIDir,
		ServiceName:   "harns-gateway",
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	updateMgr.Start()
	c.UpdateMgr = updateMgr
	gatewayMgr.SetUpdateVersionProvider(func() (string, string, string) {
		state := updateMgr.Snapshot()
		return state.Installed.ReleaseVersion, state.Installed.BackendVersion, state.Installed.FrontendVersion
	})

	c.KeyFile = o.KeyFile
	c.CertFile = o.CertFile
	return c, nil
}

func resolveOTAStagingDir(sqlitePath string) string {
	dir := filepath.Dir(strings.TrimSpace(sqlitePath))
	if dir == "" || dir == "." {
		return "data/updates"
	}
	return filepath.Join(dir, "updates")
}

func loadCustomDrivers(ctx context.Context, drivers *drivermgr.Service, pluginDir string, collections *collectionmgr.Service, runtime *collectionmgr.RuntimeManager) {
	if drivers == nil || pluginDir == "" {
		return
	}
	items, err := drivers.List(ctx, "")
	if err != nil {
		logging.Warnf("Failed to list custom drivers: %v", err)
		return
	}
	for _, item := range items {
		if item == nil || item.BuiltinDriver || item.PluginName == "" {
			continue
		}
		path := filepath.Join(pluginDir, item.PluginName, "linux-"+stdruntime.GOARCH, item.PluginName+".so")
		driver, err := pluginmgr.Load(path)
		if err != nil {
			logging.Warnf("Failed to load custom driver plugin: driverType=%s driverModel=%s plugin=%s err=%v", item.DriverType, item.Model, item.PluginName, err)
			continue
		}
		descriptor := driver.Descriptor()
		if descriptor.DriverType != item.DriverType || descriptor.DriverModel != item.Model || descriptor.PluginName != item.PluginName {
			logging.Warnf("Custom driver plugin descriptor mismatch: driverType=%s driverModel=%s plugin=%s", item.DriverType, item.Model, item.PluginName)
			continue
		}
		factory := custom.NewFactory(item.DriverType, item.Model, driver)
		if err := collections.RegisterCreateHandler(factory); err != nil {
			logging.Warnf("Failed to register custom driver create handler: driverType=%s driverModel=%s err=%v", item.DriverType, item.Model, err)
			continue
		}
		if err := runtime.RegisterFactory(factory); err != nil {
			logging.Warnf("Failed to register custom driver runtime: driverType=%s driverModel=%s err=%v", item.DriverType, item.Model, err)
			continue
		}
		logging.Infof("Loaded custom driver plugin: driverType=%s driverModel=%s plugin=%s path=%s", item.DriverType, item.Model, item.PluginName, path)
	}
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
