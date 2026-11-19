//go:build !linux

package pluginmgr

import "errors"

// ErrUnsupported reports that Go plugins are only supported on Linux.
var ErrUnsupported = errors.New("custom driver plugins are only supported on Linux")

// Load is unavailable on non-Linux platforms.
func Load(string) (Driver, error) {
	return nil, ErrUnsupported
}
