package host

import (
	"context"
	"runtime"
	"strings"

	gopsutilcpu "github.com/shirou/gopsutil/v3/cpu"
	gopsutilhost "github.com/shirou/gopsutil/v3/host"
)

type SystemInfo struct {
	HardwareID    string
	Architecture  string
	CPUModel      string
	OSVersion     string
	KernelVersion string
}

func GetSystemInfo(ctx context.Context) SystemInfo {
	info, err := gopsutilhost.InfoWithContext(ctx)
	if err != nil {
		return SystemInfo{
			HardwareID:    "unknown",
			Architecture:  runtime.GOARCH,
			CPUModel:      "unknown",
			OSVersion:     runtime.GOOS,
			KernelVersion: "unknown",
		}
	}
	hardwareID := strings.TrimSpace(info.HostID)
	if hardwareID == "" {
		hardwareID = "unknown"
	}
	architecture := strings.TrimSpace(info.KernelArch)
	if architecture == "" {
		architecture = runtime.GOARCH
	}
	osVersion := strings.TrimSpace(info.Platform)
	if platformVersion := strings.TrimSpace(info.PlatformVersion); platformVersion != "" {
		osVersion = strings.TrimSpace(osVersion + " " + platformVersion)
	}
	if osVersion == "" {
		osVersion = runtime.GOOS
	}
	kernelVersion := strings.TrimSpace(info.KernelVersion)
	if kernelVersion == "" {
		kernelVersion = "unknown"
	}
	cpuModel := "unknown"
	if cpuInfo, err := gopsutilcpu.InfoWithContext(ctx); err == nil {
		for _, item := range cpuInfo {
			if model := strings.TrimSpace(item.ModelName); model != "" {
				cpuModel = model
				break
			}
		}
	}
	return SystemInfo{
		HardwareID:    hardwareID,
		Architecture:  architecture,
		CPUModel:      cpuModel,
		OSVersion:     osVersion,
		KernelVersion: kernelVersion,
	}
}
