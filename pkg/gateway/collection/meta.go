// Package collection defines the gateway's independent collection model.
package collection

import (
	"sync"
	"time"
)

// Meta contains the common identity fields of a collector.
type Meta struct {
	Name    string    `json:"name"`
	ID      string    `json:"id"`
	Version string    `json:"version"`
	ModTime time.Time `json:"modTime"`
}

// GetName returns the collector name.
func (m *Meta) GetName() string { return m.Name }

// SetName sets the collector name.
func (m *Meta) SetName(name string) { m.Name = name }

// GetID returns the collector ID.
func (m *Meta) GetID() string { return m.ID }

// SetID sets the collector ID.
func (m *Meta) SetID(id string) { m.ID = id }

// GetVersion returns the collector version.
func (m *Meta) GetVersion() string { return m.Version }

// SetVersion sets the collector version.
func (m *Meta) SetVersion(version string) { m.Version = version }

// GetModTime returns the collector modification time.
func (m *Meta) GetModTime() time.Time { return m.ModTime }

// SetModTime sets the collector modification time.
func (m *Meta) SetModTime(modTime time.Time) { m.ModTime = modTime }

// CollectionMeta contains the runtime metadata of a collector.
type CollectionMeta struct {
	Meta
	DriverType       string      `json:"driverType"`
	DriverModel      string      `json:"driverModel"`
	CollectStatus    string      `json:"collectStatus"`
	CollectorCycleMs uint64      `json:"collectorCycleMs"`
	Variables        []*Variable `json:"variables,omitempty"`

	variablesMap map[string]*Variable
	mu           sync.RWMutex
}

// GetDriverType returns the driver type.
func (m *CollectionMeta) GetDriverType() string { return m.DriverType }

// SetDriverType sets the driver type.
func (m *CollectionMeta) SetDriverType(driverType string) { m.DriverType = driverType }

// GetDriverModel returns the driver model.
func (m *CollectionMeta) GetDriverModel() string { return m.DriverModel }

// SetDriverModel sets the driver model.
func (m *CollectionMeta) SetDriverModel(driverModel string) { m.DriverModel = driverModel }

// GetCollectStatus returns the collection status.
func (m *CollectionMeta) GetCollectStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CollectStatus
}

// SetCollectStatus sets the collection status.
func (m *CollectionMeta) SetCollectStatus(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CollectStatus = status
}

// GetCollectorCycleMs returns the collection cycle in milliseconds.
func (m *CollectionMeta) GetCollectorCycleMs() uint64 { return m.CollectorCycleMs }

// SetCollectorCycleMs sets the collection cycle in milliseconds.
func (m *CollectionMeta) SetCollectorCycleMs(cycleMs uint64) { m.CollectorCycleMs = cycleMs }

// GetVariables returns a snapshot of the collection variables.
func (m *CollectionMeta) GetVariables() []*Variable { return m.SnapshotVariables() }

// SetVariables replaces the collection variables.
func (m *CollectionMeta) SetVariables(variables []*Variable) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Variables = variables
	m.rebuildIndexLocked()
}

// SetVariableValue sets one runtime variable value.
func (m *CollectionMeta) SetVariableValue(name string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.variablesMap == nil {
		m.rebuildIndexLocked()
	}
	if variable, ok := m.variablesMap[name]; ok {
		variable.Value = value
	}
}

// GetVariableValue returns one runtime variable value.
func (m *CollectionMeta) GetVariableValue(name string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.variablesMap == nil {
		return nil, false
	}
	variable, ok := m.variablesMap[name]
	if !ok {
		return nil, false
	}
	return variable.Value, true
}

// SnapshotVariableValues returns runtime values for the requested variable names.
func (m *CollectionMeta) SnapshotVariableValues(names []string) map[string]any {
	if m == nil || len(names) == 0 {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.variablesMap == nil {
		return nil
	}
	values := make(map[string]any, len(names))
	for _, name := range names {
		if variable, ok := m.variablesMap[name]; ok {
			values[name] = variable.Value
		}
	}
	return values
}

// SnapshotVariables returns a copy of variables with runtime values.
func (m *CollectionMeta) SnapshotVariables() []*Variable {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Variable, 0, len(m.Variables))
	for _, variable := range m.Variables {
		if variable == nil {
			continue
		}
		copied := *variable
		out = append(out, &copied)
	}
	return out
}

// GetCollectionMeta returns the collection metadata.
func (m *CollectionMeta) GetCollectionMeta() *CollectionMeta { return m }

// SetCollectionMeta replaces the collection metadata.
func (m *CollectionMeta) SetCollectionMeta(meta *CollectionMeta) {
	if meta == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Meta = meta.Meta
	m.DriverType = meta.DriverType
	m.DriverModel = meta.DriverModel
	m.CollectStatus = meta.CollectStatus
	m.CollectorCycleMs = meta.CollectorCycleMs
	m.Variables = meta.Variables
	m.rebuildIndexLocked()
}

// IndexCollection rebuilds the variable index.
func (m *CollectionMeta) IndexCollection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rebuildIndexLocked()
}

func (m *CollectionMeta) rebuildIndexLocked() {
	m.variablesMap = make(map[string]*Variable, len(m.Variables))
	for _, variable := range m.Variables {
		if variable != nil {
			m.variablesMap[variable.Name] = variable
		}
	}
}
