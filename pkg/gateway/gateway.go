package gateway

import (
	"fmt"
	"time"

	"harns-platform/pkg/gateway/runtime"
)

type GatewayMeta struct {
	Secret   string    `json:"secret"`
	InitTime time.Time `json:"initTime"`
	runtime.ObjectMeta
}

// GatewayMetaVo is the gateway metadata response with computed uptime fields.
type GatewayMetaVo struct {
	GatewayMeta
	TotalUptime   string `json:"totalUptime"`
	CurrentUptime string `json:"currentUptime"`
}

// ToVo builds a response copy with uptime values at the given time.
func (m *GatewayMeta) ToVo(now time.Time) *GatewayMetaVo {
	if m == nil {
		return nil
	}
	copied := *m
	return &GatewayMetaVo{
		GatewayMeta:   copied,
		TotalUptime:   formatUptime(now.Sub(m.InitTime)),
		CurrentUptime: formatUptime(now.Sub(m.ModTime)),
	}
}

func formatUptime(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	days := int64(duration / (24 * time.Hour))
	duration -= time.Duration(days) * 24 * time.Hour
	hours := int64(duration / time.Hour)
	duration -= time.Duration(hours) * time.Hour
	minutes := int64(duration / time.Minute)
	duration -= time.Duration(minutes) * time.Minute
	seconds := int64(duration / time.Second)
	return fmt.Sprintf("%dd%dh%dm%ds", days, hours, minutes, seconds)
}

type ResponseModel struct {
	Cpus  interface{} `json:"cpus,omitempty"`
	Mem   interface{} `json:"mem,omitempty"`
	Disks interface{} `json:"disk,omitempty"`
}

type MemUsageInfo struct {
	Total       string
	Used        string
	UsedPercent string
}

type DiskUsageInfo struct {
	Total       string
	Used        string
	UsedPercent string
}
