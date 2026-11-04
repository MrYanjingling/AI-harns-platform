package options

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"

	_ "github.com/go-sql-driver/mysql"

	"harns-platform/cmd/platform/config"
	"harns-platform/pkg/audit"
	"harns-platform/pkg/emqx"
	"harns-platform/pkg/generic"
	baseoptions "harns-platform/pkg/generic/options"
	"harns-platform/pkg/logging"
	"harns-platform/pkg/platform/auth"
	"harns-platform/pkg/platform/files"
	"harns-platform/pkg/platform/migrate"
	"harns-platform/pkg/telemetry"
	"harns-platform/pkg/timeseries"
)

type Options struct {
	Port     string               `json:"port"`
	Wait     baseoptions.Duration `json:"graceful-timeout"`
	CertFile string               `json:"cert-file"`
	KeyFile  string               `json:"key-file"`
	Redis    RedisOptions         `json:"redis"`
	MySQL    MySQLOptions         `json:"mysql"`
	MinIO    MinIOOptions         `json:"minio"`
	Security auth.Config          `json:"security"`
	CORS     generic.CORSConfig   `json:"cors"`
	Audit    audit.Config         `json:"audit"`
	Things   ThingsOptions        `json:"things"`
	// Migrate runs the embedded database migrations on startup. Default: true.
	Migrate      bool              `json:"migrate"`
	MaxBodyBytes int64             `json:"max-body-bytes"`
	Telemetry    telemetry.Config  `json:"telemetry"`
	Timeseries   timeseries.Config `json:"timeseries"`
	Files        files.Config      `json:"files"`
	EMQX         emqx.Config       `json:"emqx"`
	baseoptions.BaseOptions
}

type RedisOptions struct {
	Addrs               []string `json:"addrs"`
	Username            string   `json:"username"`
	Password            string   `json:"password"`
	DB                  int      `json:"db"`
	ReadTimeoutSeconds  int      `json:"read-timeout-seconds"`
	WriteTimeoutSeconds int      `json:"write-timeout-seconds"`
}

type MySQLOptions struct {
	Host                   string `json:"host"`
	Port                   int    `json:"port"`
	Username               string `json:"username"`
	Password               string `json:"password"`
	Database               string `json:"database"`
	Charset                string `json:"charset"`
	MaxOpenConns           int    `json:"max-open-conns"`
	MaxIdleConns           int    `json:"max-idle-conns"`
	ConnMaxLifetimeMinutes int    `json:"conn-max-lifetime-minutes"`
}

type ThingsOptions struct {
	// OnlineTTLSeconds is the window used to decide whether a thing is online.
	OnlineTTLSeconds int `json:"online-ttl-seconds"`
	// AuthSecret protects the EMQX endpoints. Empty disables the check.
	AuthSecret string         `json:"auth-secret"`
	Message    MessageOptions `json:"message"`
}

type MessageOptions struct {
	KafkaEnabled     bool         `json:"kafka-enabled"`
	TriggerQueueSize int          `json:"trigger-queue-size"`
	Kafka            KafkaOptions `json:"kafka"`
}

type KafkaOptions struct {
	Brokers                   []string `json:"brokers"`
	Topic                     string   `json:"topic"`
	GroupID                   string   `json:"group-id"`
	BatchSize                 int      `json:"batch-size"`
	MaxWaitMs                 int      `json:"max-wait-ms"`
	AlarmPropertyTriggerTopic string   `json:"alarm-property-trigger-topic"`
	RulePropertyTriggerTopic  string   `json:"rule-property-trigger-topic"`
}

type MinIOOptions struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access-key"`
	SecretKey string `json:"secret-key"`
	UseSSL    bool   `json:"use-ssl"`
	Bucket    string `json:"bucket"`
}

const (
	_defaultPort                              = "32300"
	_defaultWait                              = baseoptions.Duration(15 * time.Second)
	_defaultConfigFile                        = "/etc/platform/config.yaml"
	_defaultMySQLHost                         = "127.0.0.1"
	_defaultMySQLPort                         = 3306
	_defaultMySQLCharset                      = "utf8mb4"
	_defaultMySQLMaxOpenConns                 = 50
	_defaultMySQLMaxIdleConns                 = 10
	_defaultMySQLConnMaxLifetimeMinutes       = 30
	_defaultRedisTimeoutSeconds               = 3
	_defaultMinIOEndpoint                     = "127.0.0.1:9000"
	_defaultMinIOBucket                       = "harns"
	_defaultMaxBodyBytes                int64 = 1 << 20
	_defaultOnlineTTLSeconds                  = 60
	_defaultKafkaTopic                        = "iot.things.pub-message"
	_defaultKafkaGroupID                      = "harns-platform-things-message"
	_defaultKafkaBatchSize                    = 100
	_defaultKafkaMaxWaitMs                    = 1000
	_defaultAlarmPropertyTriggerTopic         = "iot.alarm.property.trigger"
	_defaultRulePropertyTriggerTopic          = "iot.rule.property.trigger"
	_defaultTriggerQueueSize                  = 1024
)

var (
	_defaultRedisAddrs = []string{"127.0.0.1:6379"}
)

func NewDefaultOptions() *Options {
	base := baseoptions.NewDefaultBaseOptions()
	base.ConfigFile = _defaultConfigFile
	return &Options{
		Port:        _defaultPort,
		Wait:        _defaultWait,
		CertFile:    "",
		KeyFile:     "",
		BaseOptions: base,
		Redis: RedisOptions{
			Addrs:               _defaultRedisAddrs,
			ReadTimeoutSeconds:  _defaultRedisTimeoutSeconds,
			WriteTimeoutSeconds: _defaultRedisTimeoutSeconds,
		},
		MySQL: MySQLOptions{
			Host:                   _defaultMySQLHost,
			Port:                   _defaultMySQLPort,
			Charset:                _defaultMySQLCharset,
			MaxOpenConns:           _defaultMySQLMaxOpenConns,
			MaxIdleConns:           _defaultMySQLMaxIdleConns,
			ConnMaxLifetimeMinutes: _defaultMySQLConnMaxLifetimeMinutes,
		},
		MinIO: MinIOOptions{
			Endpoint: _defaultMinIOEndpoint,
			Bucket:   _defaultMinIOBucket,
		},
		Security: auth.DefaultConfig(),
		CORS:     generic.DefaultCORSConfig(),
		Audit:    audit.DefaultConfig(),
		Things: ThingsOptions{
			OnlineTTLSeconds: _defaultOnlineTTLSeconds,
			Message: MessageOptions{
				TriggerQueueSize: _defaultTriggerQueueSize,
				Kafka: KafkaOptions{
					Topic:                     _defaultKafkaTopic,
					GroupID:                   _defaultKafkaGroupID,
					BatchSize:                 _defaultKafkaBatchSize,
					MaxWaitMs:                 _defaultKafkaMaxWaitMs,
					AlarmPropertyTriggerTopic: _defaultAlarmPropertyTriggerTopic,
					RulePropertyTriggerTopic:  _defaultRulePropertyTriggerTopic,
				},
			},
		},
		Migrate:      true,
		MaxBodyBytes: _defaultMaxBodyBytes,
		Telemetry:    telemetry.DefaultConfig(),
		Timeseries:   timeseries.DefaultConfig(),
		Files:        files.DefaultConfig(),
		EMQX:         emqx.DefaultConfig(),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port exposed")
	baseoptions.DurationVar(fs, &o.Wait, "graceful-timeout", "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	fs.StringVarP(&o.CertFile, "cert-file", "", o.CertFile, "The Cert file")
	fs.StringVarP(&o.KeyFile, "key-file", "", o.KeyFile, "The Key file")
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{
		CertFile:    o.CertFile,
		KeyFile:     o.KeyFile,
		MinIOBucket: o.MinIO.Bucket,
		Security:    o.Security,
		CORS:        o.CORS,
		Audit:       o.Audit,
		Timeseries:  o.Timeseries,
		Files:       o.Files,
		EMQX:        o.EMQX,
		Things: config.ThingsConfig{
			OnlineTTLSeconds: o.Things.OnlineTTLSeconds,
			AuthSecret:       o.Things.AuthSecret,
			Message: config.MessageConfig{
				KafkaEnabled:     o.Things.Message.KafkaEnabled,
				TriggerQueueSize: o.Things.Message.TriggerQueueSize,
				Kafka: config.KafkaConfig{
					Brokers:                   o.Things.Message.Kafka.Brokers,
					Topic:                     o.Things.Message.Kafka.Topic,
					GroupID:                   o.Things.Message.Kafka.GroupID,
					BatchSize:                 o.Things.Message.Kafka.BatchSize,
					MaxWaitMs:                 o.Things.Message.Kafka.MaxWaitMs,
					AlarmPropertyTriggerTopic: o.Things.Message.Kafka.AlarmPropertyTriggerTopic,
					RulePropertyTriggerTopic:  o.Things.Message.Kafka.RulePropertyTriggerTopic,
				},
			},
		},
		MaxBodyBytes: o.MaxBodyBytes,
	}

	c.Redis = redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        o.Redis.Addrs,
		Username:     o.Redis.Username,
		Password:     o.Redis.Password,
		DB:           o.Redis.DB,
		ReadTimeout:  time.Duration(o.Redis.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(o.Redis.WriteTimeoutSeconds) * time.Second,
	})
	redisCtx, cancelRedis := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelRedis()
	if err := c.Redis.Ping(redisCtx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis failed: %w", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local&timeout=5s&readTimeout=10s&writeTimeout=10s",
		o.MySQL.Username, o.MySQL.Password, o.MySQL.Host, o.MySQL.Port, o.MySQL.Database, o.MySQL.Charset)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql failed: %w", err)
	}
	db.SetMaxOpenConns(o.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(o.MySQL.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(o.MySQL.ConnMaxLifetimeMinutes) * time.Minute)

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect mysql failed: %w", err)
	}
	c.DB = db

	if o.Migrate {
		logging.V(1).InfoS("Running database migrations", "database", o.MySQL.Database)
		migCtx, cancelMig := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancelMig()
		if err := migrate.Run(migCtx, dsn); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("run migrations failed: %w", err)
		}
		logging.V(1).InfoS("Database migrations completed", "database", o.MySQL.Database)
	} else {
		logging.Warnf("Automatic database migrations are disabled (migrate: false)")
	}

	minioClient, err := minio.New(o.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(o.MinIO.AccessKey, o.MinIO.SecretKey, ""),
		Secure: o.MinIO.UseSSL,
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	c.MinIO = minioClient
	c.Telemetry = o.Telemetry

	shutdown, err := telemetry.Setup(context.Background(), o.Telemetry)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	c.TelemetryShutdown = shutdown

	return c, nil
}
