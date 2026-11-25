package collection

import (
	"runtime"
	"strings"
)

// NormalizeSerialPort accepts both full device paths and short Linux/macOS
// serial names. COM ports and Windows paths are returned unchanged.
func NormalizeSerialPort(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || runtime.GOOS == "windows" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "\\") {
		return value
	}
	if len(value) >= 3 && strings.EqualFold(value[:3], "COM") {
		return value
	}
	return "/dev/" + value
}
