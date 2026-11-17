package gatewaymgr

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"harns-platform/pkg/host"
	"harns-platform/pkg/logging"
	"harns-platform/pkg/version"
)

type Option func(*Manager)

type Manager struct {
	cpus        []float64
	mux         *sync.RWMutex
	metaMu      sync.RWMutex
	gatewayMeta *GatewayMeta
	service     *Service
	stopCh      <-chan struct{}
	storagePath string
}

// WithStoragePath sets the directory used for database and log disk usage.
func WithStoragePath(path string) Option {
	return func(m *Manager) {
		m.storagePath = strings.TrimSpace(path)
	}
}

func NewGatewayManager(service *Service, stop <-chan struct{}, opts ...Option) *Manager {
	m := &Manager{
		cpus:        make([]float64, 0, 15),
		mux:         &sync.RWMutex{},
		gatewayMeta: &GatewayMeta{},
		service:     service,
		stopCh:      stop,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Manager) Init(ctx context.Context) error {
	if m.service == nil {
		return fmt.Errorf("gateway service is nil")
	}
	meta, err := m.service.LoadOrCreate(ctx)
	if err != nil {
		return err
	}
	m.metaMu.Lock()
	m.gatewayMeta = meta
	m.metaMu.Unlock()
	logging.V(3).InfoS("Gateway metadata loaded", "gatewayId", meta.ID)
	go m.collectCpu()
	return nil
}

func (m *Manager) GetGatewayMeta() (*GatewayMeta, error) {
	if m == nil {
		return nil, fmt.Errorf("gateway manager is nil")
	}
	m.metaMu.RLock()
	defer m.metaMu.RUnlock()
	if m.gatewayMeta == nil {
		return nil, nil
	}
	copied := *m.gatewayMeta
	return &copied, nil
}

// GetGatewayMetaVo returns gateway metadata with current uptime values.
func (m *Manager) GetGatewayMetaVo() (*GatewayMetaVo, error) {
	if m == nil {
		return nil, fmt.Errorf("gateway metadata is not initialized")
	}
	m.metaMu.RLock()
	defer m.metaMu.RUnlock()
	if m.gatewayMeta == nil {
		return nil, fmt.Errorf("gateway metadata is not initialized")
	}
	return m.gatewayMeta.ToVo(time.Now().UTC()), nil
}

func (m *Manager) GetSystemInfoMeta(ctx context.Context) *SystemInfoMeta {
	info := host.GetSystemInfo(ctx)
	return &SystemInfoMeta{
		Version:       version.Version(),
		CommitID:      version.CommitID(),
		HardwareID:    info.HardwareID,
		Architecture:  info.Architecture,
		CPUModel:      info.CPUModel,
		OSVersion:     info.OSVersion,
		KernelVersion: info.KernelVersion,
	}
}

func (m *Manager) UpdateGatewayName(ctx context.Context, name string) (*GatewayMetaVo, error) {
	if m == nil || m.service == nil {
		return nil, fmt.Errorf("gateway service is nil")
	}
	m.metaMu.Lock()
	defer m.metaMu.Unlock()
	if m.gatewayMeta == nil {
		return nil, fmt.Errorf("gateway metadata is not initialized")
	}
	name = strings.TrimSpace(name)
	eTag, err := m.service.Rename(ctx, m.gatewayMeta.ID, name)
	if err != nil {
		return nil, err
	}
	m.gatewayMeta.Name = name
	m.gatewayMeta.Version = eTag
	return m.gatewayMeta.ToVo(time.Now().UTC()), nil
}

func (m *Manager) getGatewayCpu() (map[string]string, error) {
	m.mux.RLock()
	data := make(map[string]string, 0)
	if len(m.cpus) == 0 {
		data["1min"] = fmt.Sprintf("%0.0f%%", 0.0)
		data["5min"] = fmt.Sprintf("%0.0f%%", 0.0)
		data["15min"] = fmt.Sprintf("%0.0f%%", 0.0)
		return data, nil
	}
	data["1min"] = fmt.Sprintf("%0.0f%%", m.cpus[len(m.cpus)-1])
	if len(m.cpus) < 5 {
		count := 0.0
		for _, cpu := range m.cpus {
			count += cpu
		}
		data["5min"] = fmt.Sprintf("%0.0f%%", count/float64(len(m.cpus)))
		data["15min"] = fmt.Sprintf("%0.0f%%", count/float64(len(m.cpus)))
	} else {
		count := 0.0
		for i := len(m.cpus) - 1; i > len(m.cpus)-6; i-- {
			count += m.cpus[i]
		}
		data["5min"] = fmt.Sprintf("%0.0f%%", count/float64(5))
		for i := 0; i < len(m.cpus)-5; i++ {
			count += m.cpus[i]
		}
		data["15min"] = fmt.Sprintf("%0.0f%%", count/float64(len(m.cpus)))
	}
	m.mux.RUnlock()
	return data, nil
}

func (m *Manager) getGatewayMem() (*MemUsageInfo, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		logging.V(2).InfoS("Failed to get gateway mem info", "err", err)
		return nil, err
	}
	return &MemUsageInfo{
		Total:       formatGB(v.Total),
		Used:        formatGB(v.Used),
		UsedPercent: fmt.Sprintf("%0.0f%%", v.UsedPercent),
	}, nil
}

func (m *Manager) getGatewayDisk() (map[string]*DiskUsageInfo, error) {
	data := map[string]*DiskUsageInfo{}
	osDisk := host.GetOSDisk()
	if info, err := diskUsageInfo(osDisk); err != nil {
		logging.V(2).InfoS("Failed to get gateway os disk info", "path", osDisk, "err", err)
	} else {
		data[osDisk] = info
	}
	if m.storagePath != "" && m.storagePath != osDisk {
		if info, err := diskUsageInfo(m.storagePath); err != nil {
			logging.V(2).InfoS("Failed to get gateway storage disk info", "path", m.storagePath, "err", err)
		} else {
			data[m.storagePath] = info
		}
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("failed to get gateway disk info")
	}
	return data, nil
}

func diskUsageInfo(path string) (*DiskUsageInfo, error) {
	usage, err := disk.Usage(path)
	if err != nil {
		return nil, err
	}
	return &DiskUsageInfo{
		Total:       formatGB(usage.Total),
		Used:        formatGB(usage.Used),
		UsedPercent: fmt.Sprintf("%0.0f%%", usage.UsedPercent),
	}, nil
}

func formatGB(bytes uint64) string {
	return fmt.Sprintf("%.1fGB", float64(bytes)/(1024*1024*1024))
}

func (m *Manager) collectCpu() {
	for {
		select {
		case _, ok := <-m.stopCh:
			if !ok {
				return
			}
		default:
			percent, err := cpu.Percent(time.Minute, false)
			if err != nil {
				logging.V(2).InfoS("Failed to collect gateway cpu info")
			}
			m.mux.Lock()
			if len(m.cpus) < 15 {
				m.cpus = append(m.cpus, percent[0])
			} else {
				m.cpus = append(m.cpus[1:], percent[0])
			}
			m.mux.Unlock()
		}
	}
}
