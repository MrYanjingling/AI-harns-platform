package version

import (
	"runtime/debug"
	"strings"
)

// CommitID returns the VCS commit that produced this binary.
func CommitID() string {
	if commit := strings.TrimSpace(Get().GitCommit); commit != "" && !strings.Contains(commit, "$Format") {
		return commit
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				if commit := strings.TrimSpace(setting.Value); commit != "" {
					return commit
				}
			}
		}
	}
	if version := strings.TrimSpace(Get().GitVersion); version != "" && !strings.Contains(version, "$Format") {
		return version
	}
	return "unknown"
}
