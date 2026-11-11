package modbustcp

import "encoding/json"

type protocolConfig struct {
	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
	TCPCount        int    `json:"tcpCount"`
}

// ConfigJSON serializes only the Modbus TCP protocol configuration.
func (c *ModbusTcpCollection) ConfigJSON() ([]byte, error) {
	tcpCount := c.TCPCount
	if tcpCount == 0 {
		tcpCount = DefaultTCPCount
	}
	return json.Marshal(protocolConfig{
		Location:        c.Location,
		BackupLocation:  c.BackupLocation,
		MemoryLayout:    c.MemoryLayout,
		PositionAddress: c.PositionAddress,
		TCPCount:        tcpCount,
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
	c.TCPCount = config.TCPCount
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
	return nil
}
