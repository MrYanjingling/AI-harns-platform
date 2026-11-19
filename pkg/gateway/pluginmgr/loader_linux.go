//go:build linux

package pluginmgr

import (
	"fmt"
	"plugin"
	"runtime"
)

// Load opens one custom driver plugin and validates its ABI and build target.
func Load(path string) (Driver, error) {
	handle, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open plugin: %w", err)
	}
	symbol, err := handle.Lookup(DriverSymbolName)
	if err != nil {
		return nil, fmt.Errorf("lookup %s: %w", DriverSymbolName, err)
	}
	factory, ok := symbol.(func() Driver)
	if !ok {
		return nil, fmt.Errorf("plugin symbol %s must be func() Driver", DriverSymbolName)
	}
	driver := factory()
	if driver == nil {
		return nil, fmt.Errorf("plugin driver is nil")
	}
	descriptor := driver.Descriptor()
	if descriptor.ABIVersion != ABIVersion {
		return nil, fmt.Errorf("plugin ABI %q does not match host ABI %q", descriptor.ABIVersion, ABIVersion)
	}
	if descriptor.GOOS != "" && descriptor.GOOS != runtime.GOOS {
		return nil, fmt.Errorf("plugin GOOS %q does not match host %q", descriptor.GOOS, runtime.GOOS)
	}
	if descriptor.GOARCH != "" && descriptor.GOARCH != runtime.GOARCH {
		return nil, fmt.Errorf("plugin GOARCH %q does not match host %q", descriptor.GOARCH, runtime.GOARCH)
	}
	if descriptor.GoVersion != "" && descriptor.GoVersion != runtime.Version() {
		return nil, fmt.Errorf("plugin Go version %q does not match host %q", descriptor.GoVersion, runtime.Version())
	}
	return driver, nil
}
