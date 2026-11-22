package pushmgr

import "sync"

type connectorRuntime struct {
	record    *ConnectorRecord
	connector Connector
	mu        sync.RWMutex
	status    string
	lastError string
}

func (c *connectorRuntime) setStatus(status string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.status = status
	c.mu.Unlock()
}

func (c *connectorRuntime) getStatus() string {
	if c == nil {
		return ConnectorStatusStopped
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.status == "" {
		return ConnectorStatusStopped
	}
	return c.status
}

func (c *connectorRuntime) setRunning() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.status = ConnectorStatusRunning
	c.lastError = ""
	c.mu.Unlock()
}

func (c *connectorRuntime) setError(err error) {
	if c == nil {
		return
	}
	message := "connector failed"
	if err != nil {
		message = err.Error()
	}
	c.mu.Lock()
	c.status = ConnectorStatusError
	c.lastError = message
	c.mu.Unlock()
}

func (c *connectorRuntime) setStopped() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.status = ConnectorStatusStopped
	c.lastError = ""
	c.mu.Unlock()
}

func (c *connectorRuntime) getLastError() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}
