package storage

import (
	"os"
	"path/filepath"
)

var storePath = defaultStorePath()

func defaultStorePath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "harnsgateway")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".harnsgateway")
	}
	return "./harnsgateway"
}

func isEphemeralError(err error) bool {
	return false
}
