package modbustcp

import "encoding/json"

type protocolConfig struct {
	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress"`
	Concurrency     int    `json:"concurrency"`
}

// ConfigJSON serializes only the Modbus TCP protocol configuration.
func (c *ModbusTcpCollection) ConfigJSON() ([]byte, error) {
	concurrency := c.Concurrency
	if concurrency == 0 {
		concurrency = DefaultConcurrency
	}
	return json.Marshal(protocolConfig{
		Location:        c.Location,
		BackupLocation:  c.BackupLocation,
		MemoryLayout:    c.MemoryLayout,
		PositionAddress: c.PositionAddress,
		Concurrency:     concurrency,
	})
}

// ApplyConfig replaces the Modbus TCP protocol configuration from JSON.
func (c *ModbusTcpCollection) ApplyConfig(raw []byte) error {
	var config protocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	c.Location = config.Location
	c.BackupLocation = config.BackupLocation
	c.MemoryLayout = config.MemoryLayout
	c.PositionAddress = config.PositionAddress
	c.Concurrency = config.Concurrency
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConcurrency
	}
	return nil
}
