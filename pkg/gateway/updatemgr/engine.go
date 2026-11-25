package updatemgr

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"harns-platform/pkg/logging"
)

func (s *Service) applyTask(task Task) {
	dir := s.taskDir(task.TaskID)
	stagingDir := filepath.Join(dir, "staging")
	frontendSource := filepath.Join(stagingDir, "payload", "ui", "gateway-ui")
	backendSource := filepath.Join(stagingDir, "payload", "bin", "gateway")
	backupDir := filepath.Join(s.opts.StagingDir, "backup")

	if !task.RestartRequired {
		if err := s.installFrontend(frontendSource); err != nil {
			s.failTask(task.TaskID, "apply_failed", err)
			return
		}
		s.completeTask(task, false)
		go s.cleanupTask(task.TaskID)
		return
	}

	previous := s.Snapshot().Installed
	if err := s.backupCurrent(backupDir); err != nil {
		s.failTask(task.TaskID, "backup_failed", err)
		return
	}
	if err := s.installFrontend(frontendSource); err != nil {
		_ = s.restoreBackup(backupDir)
		s.restoreInstalled(previous)
		s.failTask(task.TaskID, "apply_failed", err)
		return
	}
	if err := s.installBackend(backendSource); err != nil {
		_ = s.restoreBackup(backupDir)
		s.restoreInstalled(previous)
		s.failTask(task.TaskID, "apply_failed", err)
		return
	}

	s.mu.Lock()
	if s.state != nil {
		s.state.Installed.ReleaseVersion = task.ReleaseVersion
		s.state.Installed.BackendVersion = task.BackendVersion
		s.state.Installed.BackendSHA256 = task.BackendSHA256
		s.state.Installed.FrontendVersion = task.FrontendVersion
		s.state.Installed.FrontendSHA256 = task.FrontendSHA256
		if s.state.Task != nil && s.state.Task.TaskID == task.TaskID {
			s.state.Task.State = StateRestarting
			s.state.Task.Progress = 100
			s.state.Task.ErrorCode = ""
			s.state.Task.Message = ""
			s.state.Task.UpdatedAt = time.Now().UTC()
			task = *s.state.Task
		}
		_ = saveState(s.statePath, s.state)
	}
	s.mu.Unlock()
	s.publishStatus(task)

	if err := restartGateway(s.opts.ServiceName, task.TaskID); err != nil {
		_ = s.restoreBackup(backupDir)
		s.restoreInstalled(previous)
		s.failTask(task.TaskID, "restart_failed", err)
	}
}

func (s *Service) restoreInstalled(installed Installed) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return
	}
	s.state.Installed = installed
	_ = saveState(s.statePath, s.state)
}

func (s *Service) completeTask(task Task, restart bool) {
	s.mu.Lock()
	if s.state != nil {
		s.state.Installed.ReleaseVersion = task.ReleaseVersion
		if restart {
			if task.BackendVersion != "" {
				s.state.Installed.BackendVersion = task.BackendVersion
			}
			if task.BackendSHA256 != "" {
				s.state.Installed.BackendSHA256 = task.BackendSHA256
			}
		}
		s.state.Installed.FrontendVersion = task.FrontendVersion
		s.state.Installed.FrontendSHA256 = task.FrontendSHA256
		if s.state.Task != nil && s.state.Task.TaskID == task.TaskID {
			s.state.Task.State = StateSucceeded
			s.state.Task.Progress = 100
			s.state.Task.RestartRequired = restart
			s.state.Task.ErrorCode = ""
			s.state.Task.Message = ""
			s.state.Task.UpdatedAt = time.Now().UTC()
			task = *s.state.Task
		}
		_ = saveState(s.statePath, s.state)
	}
	s.mu.Unlock()
	s.publishStatus(task)
}

func (s *Service) installFrontend(source string) error {
	if strings.TrimSpace(s.opts.UIDir) == "" {
		return fmt.Errorf("gateway UI directory is not configured")
	}
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		return fmt.Errorf("staged frontend directory is missing")
	}
	return copyDir(source, s.opts.UIDir, true)
}

func (s *Service) installBackend(source string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve gateway executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	temp, err := os.CreateTemp(filepath.Dir(executable), ".gateway-*.tmp")
	if err != nil {
		return fmt.Errorf("create gateway binary temp file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	sourceFile, err := os.Open(source)
	if err != nil {
		_ = temp.Close()
		return fmt.Errorf("open staged gateway binary: %w", err)
	}
	defer sourceFile.Close()
	if _, err := io.Copy(temp, sourceFile); err != nil {
		_ = temp.Close()
		return fmt.Errorf("copy staged gateway binary: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync staged gateway binary: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close staged gateway binary: %w", err)
	}
	if err := os.Chmod(tempName, 0o755); err != nil {
		return fmt.Errorf("chmod staged gateway binary: %w", err)
	}
	if err := os.Rename(tempName, executable); err != nil {
		return fmt.Errorf("replace gateway binary: %w", err)
	}
	return nil
}

func (s *Service) backupCurrent(backupDir string) error {
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("remove old update backup: %w", err)
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("create update backup: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve gateway executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	if err := copyFile(executable, filepath.Join(backupDir, "gateway"), 0o755); err != nil {
		return fmt.Errorf("backup gateway binary: %w", err)
	}
	if strings.TrimSpace(s.opts.UIDir) != "" {
		if info, err := os.Stat(s.opts.UIDir); err == nil && info.IsDir() {
			if err := copyDir(s.opts.UIDir, filepath.Join(backupDir, "gateway-ui"), false); err != nil {
				return fmt.Errorf("backup gateway UI: %w", err)
			}
		}
	}
	return nil
}

func (s *Service) restoreBackup(backupDir string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve gateway executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	if err := copyFile(filepath.Join(backupDir, "gateway"), executable, 0o755); err != nil {
		return fmt.Errorf("restore gateway binary: %w", err)
	}
	if strings.TrimSpace(s.opts.UIDir) != "" {
		if info, err := os.Stat(filepath.Join(backupDir, "gateway-ui")); err == nil && info.IsDir() {
			if err := copyDir(filepath.Join(backupDir, "gateway-ui"), s.opts.UIDir, false); err != nil {
				return fmt.Errorf("restore gateway UI: %w", err)
			}
		}
	}
	return nil
}

func restartGateway(serviceName, taskID string) error {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		serviceName = defaultServiceName
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl is not available: %w", err)
	}
	var command *exec.Cmd
	if systemdRun, err := exec.LookPath("systemd-run"); err == nil {
		unit := "harns-gateway-ota-" + sanitizeUnitName(taskID)
		command = exec.Command(systemdRun, "--unit", unit, "--collect", "--no-block", "systemctl", "restart", serviceName)
	} else {
		command = exec.Command("systemctl", "restart", serviceName)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("restart gateway service: %w", err)
	}
	go func() {
		if err := command.Wait(); err != nil {
			logging.ErrorS(err, "Gateway OTA restart command failed")
		}
	}()
	return nil
}

func sanitizeUnitName(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "task"
	}
	if len(result) > 48 {
		return result[:48]
	}
	return result
}

func copyDir(source, target string, indexLast bool) error {
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	var indexEntry string
	for _, entry := range entries {
		if indexLast && entry.Name() == "index.html" {
			indexEntry = entry.Name()
			continue
		}
		if err := copyEntry(filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name())); err != nil {
			return err
		}
	}
	if indexEntry != "" {
		return copyEntry(filepath.Join(source, indexEntry), filepath.Join(target, indexEntry))
	}
	return nil
}

func copyEntry(source, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyEntry(filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("non-regular file %s is not supported", source)
	}
	return copyFile(source, target, info.Mode().Perm())
}

func copyFile(source, target string, mode os.FileMode) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(targetFile, sourceFile)
	closeErr := targetFile.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func (s *Service) cleanupTask(taskID string) {
	_ = os.RemoveAll(s.taskDir(taskID))
}
