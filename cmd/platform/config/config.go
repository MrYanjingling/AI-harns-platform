package config

import (
	"context"
	"database/sql"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"harns-platform/pkg/audit"
	"harns-platform/pkg/emqx"
	"harns-platform/pkg/generic"
	"harns-platform/pkg/platform/auth"
	"harns-platform/pkg/platform/files"
	"harns-platform/pkg/telemetry"
	"harns-platform/pkg/timeseries"
)

type ThingsConfig struct {
	OnlineTTLSeconds        int
	AuthSecret              string
	AuthDenyCacheTTLSeconds int
	AuthDenyCacheMaxEntries int
	Message                 MessageConfig
}

type MessageConfig struct {
	KafkaEnabled     bool
	TriggerQueueSize int
	Kafka            KafkaConfig
}

type KafkaConfig struct {
	Brokers                   []string
	Topic                     string
	GroupID                   string
	BatchSize                 int
	MaxWaitMs                 int
	AlarmPropertyTriggerTopic string
	RulePropertyTriggerTopic  string
}

type Config struct {
	CertFile     string
	KeyFile      string
	MaxBodyBytes int64

	Redis       redis.UniversalClient
	DB          *sql.DB
	MinIO       *minio.Client
	MinIOBucket string
	Security    auth.Config
	CORS        generic.CORSConfig
	Audit       audit.Config
	Things      ThingsConfig
	Telemetry   telemetry.Config
	Timeseries  timeseries.Config
	Files       files.Config
	EMQX        emqx.Config

	TelemetryShutdown func(context.Context) error
}
