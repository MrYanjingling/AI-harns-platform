package updatemgr

import (
	"database/sql"
	"time"
)

const (
	StateReceived    = "received"
	StateDownloading = "downloading"
	StateReady       = "ready"
	StateApplying    = "applying"
	StateRestarting  = "restarting"
	StateSucceeded   = "succeeded"
	StateFailed      = "failed"
	StateRejected    = "rejected"
	StateIgnored     = "ignored"
)

const (
	defaultStagingDir     = "data/updates"
	defaultMaxPackageSize = 512 << 20
	defaultMaxManifest    = 2 << 20
	defaultMQTTQoS        = 1
	defaultMQTTKeepAlive  = 60
	defaultServiceName    = "harns-gateway"
)

// Options configures the OTA service.
type Options struct {
	GatewayID       string
	DB              *sql.DB
	Secret          string
	PublicKeyFile   string
	StagingDir      string
	UIDir           string
	ServiceName     string
	MaxPackageBytes int64
}

// Installed describes the currently installed gateway release.
type Installed struct {
	ReleaseVersion  string `json:"releaseVersion"`
	BackendVersion  string `json:"backendVersion"`
	BackendSHA256   string `json:"backendSha256"`
	FrontendVersion string `json:"frontendVersion"`
	FrontendSHA256  string `json:"frontendSha256,omitempty"`
}

// Task is the single current OTA task kept by the gateway.
type Task struct {
	TaskID          string    `json:"taskId"`
	ManifestURL     string    `json:"manifestUrl"`
	State           string    `json:"state"`
	ReleaseVersion  string    `json:"releaseVersion,omitempty"`
	BackendVersion  string    `json:"backendVersion,omitempty"`
	FrontendVersion string    `json:"frontendVersion,omitempty"`
	BackendSHA256   string    `json:"backendSha256,omitempty"`
	FrontendSHA256  string    `json:"frontendSha256,omitempty"`
	RestartRequired bool      `json:"restartRequired"`
	Progress        int       `json:"progress"`
	ErrorCode       string    `json:"errorCode,omitempty"`
	Message         string    `json:"message,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// State is the complete local OTA state. Task is intentionally a single
// record so the gateway does not keep an OTA history.
type State struct {
	Installed Installed `json:"installed"`
	Task      *Task     `json:"task,omitempty"`
}

// TaskMessage is the MQTT payload published by the platform.
type TaskMessage struct {
	APIVersion  int    `json:"apiVersion"`
	Type        string `json:"type"`
	TaskID      string `json:"taskId"`
	GatewayID   string `json:"gatewayId"`
	ManifestURL string `json:"manifestUrl"`
	ExpiresAt   string `json:"expiresAt"`
}

// StatusMessage is the MQTT payload published by the gateway.
type StatusMessage struct {
	APIVersion      int       `json:"apiVersion"`
	Type            string    `json:"type"`
	TaskID          string    `json:"taskId"`
	GatewayID       string    `json:"gatewayId"`
	State           string    `json:"state"`
	Progress        int       `json:"progress"`
	ReleaseVersion  string    `json:"releaseVersion,omitempty"`
	BackendVersion  string    `json:"backendVersion,omitempty"`
	FrontendVersion string    `json:"frontendVersion,omitempty"`
	RestartRequired bool      `json:"restartRequired"`
	ErrorCode       string    `json:"errorCode,omitempty"`
	Message         string    `json:"message,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Manifest is the signed gateway release descriptor.
type Manifest struct {
	Format         int             `json:"format"`
	APIVersion     int             `json:"apiVersion"`
	Product        string          `json:"product"`
	ReleaseVersion string          `json:"releaseVersion"`
	OS             string          `json:"os"`
	Arch           string          `json:"arch"`
	Backend        ManifestPart    `json:"backend"`
	Frontend       ManifestPart    `json:"frontend"`
	Package        ManifestPackage `json:"package"`
	Notes          string          `json:"notes"`
}

// ManifestPart describes one updatable gateway component.
type ManifestPart struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

// ManifestPackage describes the downloadable release archive.
type ManifestPackage struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
