package main

import (
	"harns-platform/cmd/gateway/app"
	"harns-platform/pkg/logging"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if dir, ok := packagedBinDir(); ok {
		_ = os.Chdir(dir)
	}
	cmd := app.NewGatewayCmd()
	defer logging.Flush()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func packagedBinDir() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	if !strings.EqualFold(filepath.Base(dir), "bin") {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "etc", "config.yaml")); err != nil {
		return "", false
	}
	return dir, true
}
