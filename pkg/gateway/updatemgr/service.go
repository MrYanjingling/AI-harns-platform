package updatemgr

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/logging"
	"harns-platform/pkg/version"
)

var taskIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,96}$`)

// Service manages the single current gateway OTA task.
type Service struct {
	opts       Options
	publicKey  *rsa.PublicKey
	downloader *downloader
	statePath  string
	stopCh     chan struct{}
	reloadCh   chan struct{}
	closeOnce  sync.Once
	store      *settingsStore

	mu    sync.Mutex
	state *State

	settingsMu sync.RWMutex
	settings   Settings

	mqttMu        sync.Mutex
	mqttClient    mqttClient
	mqttConnected bool
}

// NewService creates the OTA service and loads its local state.
func NewService(opts Options) (*Service, error) {
	opts.StagingDir = strings.TrimSpace(opts.StagingDir)
	if opts.StagingDir == "" {
		opts.StagingDir = defaultStagingDir
	}
	opts.ServiceName = strings.TrimSpace(opts.ServiceName)
	if opts.ServiceName == "" {
		opts.ServiceName = defaultServiceName
	}
	if opts.MaxPackageBytes <= 0 {
		opts.MaxPackageBytes = defaultMaxPackageSize
	}
	opts.GatewayID = strings.TrimSpace(opts.GatewayID)

	if opts.DB == nil {
		return nil, fmt.Errorf("OTA store database is nil")
	}
	store, err := newSettingsStore(opts.DB, opts.Secret)
	if err != nil {
		return nil, err
	}
	if err := store.Initialize(context.Background()); err != nil {
		return nil, err
	}
	settings, err := store.Get(context.Background())
	if err != nil {
		return nil, err
	}
	key, err := loadPublicKey(opts.PublicKeyFile)
	if err != nil {
		return nil, err
	}
	service := &Service{
		opts:       opts,
		publicKey:  key,
		downloader: newDownloader(),
		statePath:  filepath.Join(opts.StagingDir, "state.json"),
		stopCh:     make(chan struct{}),
		reloadCh:   make(chan struct{}, 1),
		store:      store,
		settings:   *settings,
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

// Start opens the OTA MQTT connection. It is non-blocking.
func (s *Service) Start() {
	if s == nil {
		return
	}
	go s.runMQTT()
}

// Close gracefully closes the OTA MQTT connection.
func (s *Service) Close(ctx context.Context) {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		close(s.stopCh)
		s.mqttMu.Lock()
		client := s.mqttClient
		s.mqttClient = nil
		s.mqttConnected = false
		s.mqttMu.Unlock()
		if client != nil {
			client.Disconnect(250)
		}
	})
}

// Snapshot returns a copy of the current OTA state.
func (s *Service) Snapshot() *State {
	if s == nil {
		return &State{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneState(s.state)
}

// View returns the current installed state, task and safe OTA settings.
func (s *Service) View() *View {
	if s == nil {
		return &View{}
	}
	state := s.Snapshot()
	settings := s.settingsSnapshot()
	return &View{
		Installed: state.Installed,
		Task:      state.Task,
		Settings:  settings.toVo(),
	}
}

// GetSettings returns the safe OTA settings.
func (s *Service) GetSettings() SettingsVo {
	if s == nil {
		return SettingsVo{}
	}
	return s.settingsSnapshot().toVo()
}

// UpdateSettings validates, persists and applies OTA settings.
func (s *Service) UpdateSettings(ctx context.Context, req UpdateSettingsReq) (*View, error) {
	if s == nil {
		return nil, errUpdateDisabled
	}
	req.BrokerURL = strings.TrimSpace(req.BrokerURL)
	if req.Enabled {
		if req.BrokerURL == "" {
			return nil, fmt.Errorf("mqtt broker url is required")
		}
		parsed, err := url.Parse(req.BrokerURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, fmt.Errorf("mqtt broker url is invalid")
		}
		switch strings.ToLower(parsed.Scheme) {
		case "tcp", "ssl", "tls", "mqtt", "ws", "wss":
		default:
			return nil, fmt.Errorf("mqtt broker url scheme %q is not supported", parsed.Scheme)
		}
	}
	current := s.settingsSnapshot()
	item := &Settings{
		Enabled:   req.Enabled,
		BrokerURL: req.BrokerURL,
		Username:  strings.TrimSpace(req.Username),
		Password:  current.Password,
	}
	if strings.TrimSpace(req.Password) != "" {
		item.Password = req.Password
	}
	if err := s.store.Update(ctx, item); err != nil {
		return nil, err
	}
	s.settingsMu.Lock()
	s.settings = *item
	s.settingsMu.Unlock()
	s.reloadMQTT()
	return s.View(), nil
}

func (s *Service) settingsSnapshot() Settings {
	if s == nil {
		return Settings{}
	}
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return s.settings
}

func (s *Service) reloadMQTT() {
	if s == nil {
		return
	}
	select {
	case s.reloadCh <- struct{}{}:
	default:
	}
}

// MarkRunning marks a restarting task as successful after the server starts.
func (s *Service) MarkRunning() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.state == nil || s.state.Task == nil || s.state.Task.State != StateRestarting {
		s.mu.Unlock()
		return
	}
	task := *s.state.Task
	task.State = StateSucceeded
	task.Progress = 100
	task.ErrorCode = ""
	task.Message = ""
	task.UpdatedAt = time.Now().UTC()
	s.state.Task = &task
	_ = saveState(s.statePath, s.state)
	s.mu.Unlock()
	s.publishStatus(task)
	go s.cleanupTask(task.TaskID)
}

// Apply starts installing the current ready task.
func (s *Service) Apply() error {
	if s == nil || !s.settingsSnapshot().Enabled {
		return errUpdateDisabled
	}
	s.mu.Lock()
	if s.state == nil || s.state.Task == nil {
		s.mu.Unlock()
		return errUpdateNotFound
	}
	if s.state.Task.State != StateReady {
		s.mu.Unlock()
		return errUpdateBusy
	}
	task := *s.state.Task
	task.State = StateApplying
	task.Progress = 100
	task.UpdatedAt = time.Now().UTC()
	s.state.Task = &task
	if err := saveState(s.statePath, s.state); err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	s.publishStatus(task)
	go s.applyTask(task)
	return nil
}

// Reject rejects the current ready task.
func (s *Service) Reject() error {
	if s == nil || !s.settingsSnapshot().Enabled {
		return errUpdateDisabled
	}
	s.mu.Lock()
	if s.state == nil || s.state.Task == nil {
		s.mu.Unlock()
		return errUpdateNotFound
	}
	if s.state.Task.State != StateReady {
		s.mu.Unlock()
		return errUpdateBusy
	}
	task := *s.state.Task
	task.State = StateRejected
	task.UpdatedAt = time.Now().UTC()
	s.state.Task = &task
	if err := saveState(s.statePath, s.state); err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	_ = os.RemoveAll(s.taskDir(task.TaskID))
	s.publishStatus(task)
	return nil
}

func (s *Service) load() error {
	state, err := loadState(s.statePath)
	if err != nil {
		return err
	}
	if state.Task != nil {
		switch state.Task.State {
		case StateReceived, StateDownloading, StateApplying:
			state.Task.State = StateFailed
			state.Task.ErrorCode = "interrupted"
			state.Task.Message = "gateway restarted before the update task completed"
			state.Task.UpdatedAt = time.Now().UTC()
		}
	}
	if state.Installed.BackendSHA256 == "" {
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve gateway executable: %w", err)
		}
		hash, err := sha256File(executable)
		if err != nil {
			return fmt.Errorf("hash gateway executable: %w", err)
		}
		backendVersion := version.Version()
		state.Installed = Installed{
			ReleaseVersion:  backendVersion,
			BackendVersion:  backendVersion,
			BackendSHA256:   hash,
			FrontendVersion: backendVersion,
		}
	}
	s.state = state
	return saveState(s.statePath, state)
}

func (s *Service) taskDir(taskID string) string {
	return filepath.Join(s.opts.StagingDir, taskID)
}

func (s *Service) setDownloading(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil || s.state.Task == nil || s.state.Task.TaskID != taskID {
		return
	}
	s.state.Task.State = StateDownloading
	s.state.Task.Progress = 0
	s.state.Task.ErrorCode = ""
	s.state.Task.Message = ""
	s.state.Task.UpdatedAt = time.Now().UTC()
	_ = saveState(s.statePath, s.state)
}

func (s *Service) setReady(taskID string, manifest *Manifest, backendHash string, restart bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil || s.state.Task == nil || s.state.Task.TaskID != taskID {
		return
	}
	task := s.state.Task
	task.State = StateReady
	task.Progress = 100
	task.ReleaseVersion = manifest.ReleaseVersion
	task.BackendVersion = manifest.Backend.Version
	task.FrontendVersion = manifest.Frontend.Version
	task.BackendSHA256 = backendHash
	task.FrontendSHA256 = manifest.Frontend.SHA256
	task.RestartRequired = restart
	task.ErrorCode = ""
	task.Message = ""
	task.UpdatedAt = time.Now().UTC()
	if task.BackendVersion == "" {
		task.BackendVersion = s.state.Installed.BackendVersion
	}
	_ = saveState(s.statePath, s.state)
}

func (s *Service) setTaskState(taskID, state, code, message string, progress int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil || s.state.Task == nil || s.state.Task.TaskID != taskID {
		return
	}
	s.state.Task.State = state
	s.state.Task.Progress = progress
	s.state.Task.ErrorCode = code
	s.state.Task.Message = message
	s.state.Task.UpdatedAt = time.Now().UTC()
	_ = saveState(s.statePath, s.state)
}

func (s *Service) setTaskIgnored(taskID, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil || s.state.Task == nil || s.state.Task.TaskID != taskID {
		return
	}
	s.state.Task.State = StateIgnored
	s.state.Task.Progress = 0
	s.state.Task.ErrorCode = "ignored"
	s.state.Task.Message = message
	s.state.Task.UpdatedAt = time.Now().UTC()
	_ = saveState(s.statePath, s.state)
}

func (s *Service) currentInstalledBackendVersion() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return ""
	}
	return s.state.Installed.BackendVersion
}

func backendVersionChanged(candidate, installed string) bool {
	candidate = strings.TrimSpace(candidate)
	installed = strings.TrimSpace(installed)
	if candidate == "" {
		return true
	}
	if installed == "" {
		return true
	}
	return candidate != installed
}

func (s *Service) currentInstalledVersion() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return ""
	}
	return s.state.Installed.ReleaseVersion
}

func (s *Service) acceptTask(message TaskMessage) {
	taskID := strings.TrimSpace(message.TaskID)
	if !taskIDPattern.MatchString(taskID) {
		s.publishIgnored(taskID, "invalid task id")
		return
	}
	if message.GatewayID != "" && s.opts.GatewayID != "" && message.GatewayID != s.opts.GatewayID {
		s.publishIgnored(taskID, "gateway id mismatch")
		return
	}
	if message.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, message.ExpiresAt)
		if err == nil && time.Now().After(expiresAt) {
			s.publishIgnored(taskID, "task expired")
			return
		}
	}
	if strings.TrimSpace(message.ManifestURL) == "" {
		s.publishIgnored(taskID, "manifest url is required")
		return
	}

	s.mu.Lock()
	if s.state == nil {
		s.mu.Unlock()
		return
	}
	if current := s.state.Task; current != nil {
		if current.TaskID == taskID {
			task := *current
			s.mu.Unlock()
			s.publishStatus(task)
			return
		}
		switch current.State {
		case StateDownloading, StateApplying, StateRestarting, StateReady:
			s.mu.Unlock()
			s.publishIgnored(taskID, "another update task is in progress")
			return
		}
	}
	task := &Task{
		TaskID:      taskID,
		ManifestURL: message.ManifestURL,
		State:       StateReceived,
		Progress:    0,
		UpdatedAt:   time.Now().UTC(),
	}
	s.state.Task = task
	_ = saveState(s.statePath, s.state)
	s.mu.Unlock()

	s.publishStatus(*task)
	go s.processTask(taskID)
}

func (s *Service) processTask(taskID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	dir := s.taskDir(taskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.failTask(taskID, "storage_error", err)
		return
	}
	s.setDownloading(taskID)
	s.publishCurrentIf(taskID)

	manifestURL := ""
	s.mu.Lock()
	if s.state != nil && s.state.Task != nil && s.state.Task.TaskID == taskID {
		manifestURL = s.state.Task.ManifestURL
	}
	s.mu.Unlock()
	if manifestURL == "" {
		s.failTask(taskID, "manifest_invalid", fmt.Errorf("manifest url is empty"))
		return
	}

	manifestData, err := s.downloader.fetch(ctx, manifestURL, defaultMaxManifest)
	if err != nil {
		s.failTask(taskID, "download_failed", err)
		return
	}
	signatureData, err := s.downloader.fetch(ctx, signatureURL(manifestURL), 64<<10)
	if err != nil {
		s.failTask(taskID, "signature_failed", err)
		return
	}
	if err := verifyManifestSignature(s.publicKey, manifestData, string(signatureData)); err != nil {
		s.failTask(taskID, "signature_failed", err)
		return
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		s.failTask(taskID, "manifest_invalid", err)
		return
	}
	if err := validateManifest(&manifest); err != nil {
		s.failTask(taskID, "manifest_invalid", err)
		return
	}
	installedVersion := s.currentInstalledVersion()
	newer, err := isNewerVersion(manifest.ReleaseVersion, installedVersion)
	if err != nil {
		s.failTask(taskID, "version_invalid", err)
		return
	}
	if !newer {
		s.setTaskIgnored(taskID, "installed version is not older than the task version")
		s.publishCurrentIf(taskID)
		_ = os.RemoveAll(dir)
		return
	}

	packageURL, err := resolvePackageURL(manifestURL, manifest.Package.Path)
	if err != nil {
		s.failTask(taskID, "package_url_invalid", err)
		return
	}
	if _, err := validateDownloadURL(packageURL, nil, true); err != nil {
		s.failTask(taskID, "package_url_invalid", err)
		return
	}
	if manifest.Package.Size > s.opts.MaxPackageBytes {
		s.failTask(taskID, "package_too_large", fmt.Errorf("package size %d exceeds limit", manifest.Package.Size))
		return
	}
	packagePath := filepath.Join(dir, "package.zip")
	if err := s.downloader.downloadTo(ctx, packageURL, packagePath, s.opts.MaxPackageBytes); err != nil {
		s.failTask(taskID, "download_failed", err)
		return
	}
	packageHash, err := sha256File(packagePath)
	if err != nil {
		s.failTask(taskID, "package_invalid", err)
		return
	}
	if !strings.EqualFold(packageHash, manifest.Package.SHA256) {
		s.failTask(taskID, "package_invalid", fmt.Errorf("package sha256 mismatch"))
		return
	}
	stagingDir := filepath.Join(dir, "staging")
	if err := os.RemoveAll(stagingDir); err != nil {
		s.failTask(taskID, "storage_error", err)
		return
	}
	if err := extractPackage(packagePath, stagingDir, s.opts.MaxPackageBytes*4); err != nil {
		s.failTask(taskID, "package_invalid", err)
		return
	}
	backendPath := filepath.Join(stagingDir, "payload", "bin", "gateway")
	backendHash, err := sha256File(backendPath)
	if err != nil {
		s.failTask(taskID, "package_invalid", err)
		return
	}
	if manifest.Backend.SHA256 != "" && !strings.EqualFold(backendHash, manifest.Backend.SHA256) {
		s.failTask(taskID, "package_invalid", fmt.Errorf("backend sha256 mismatch"))
		return
	}
	frontendDir := filepath.Join(stagingDir, "payload", "ui", "gateway-ui")
	if info, err := os.Stat(frontendDir); err != nil || !info.IsDir() {
		s.failTask(taskID, "package_invalid", fmt.Errorf("frontend assets are missing"))
		return
	}
	restart := backendVersionChanged(manifest.Backend.Version, s.currentInstalledBackendVersion())
	s.setReady(taskID, &manifest, backendHash, restart)
	s.publishCurrentIf(taskID)
}

func (s *Service) failTask(taskID, code string, err error) {
	if err == nil {
		err = fmt.Errorf("unknown update error")
	}
	logging.ErrorS(err, "Gateway update task failed", "taskId", taskID, "code", code)
	s.setTaskState(taskID, StateFailed, code, err.Error(), 0)
	s.publishCurrentIf(taskID)
}

func (s *Service) publishCurrentIf(taskID string) {
	s.mu.Lock()
	if s.state == nil || s.state.Task == nil || s.state.Task.TaskID != taskID {
		s.mu.Unlock()
		return
	}
	task := *s.state.Task
	s.mu.Unlock()
	s.publishStatus(task)
}
