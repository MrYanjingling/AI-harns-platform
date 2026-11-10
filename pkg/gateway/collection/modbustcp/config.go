package modbustcp

import "encoding/json"

type protocolConfig struct {
	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
}

// ConfigJSON serializes only the Modbus TCP protocol configuration.
func (c *ModbusTcpCollection) ConfigJSON() ([]byte, error) {
	return json.Marshal(protocolConfig{
		Location:        c.Location,
		BackupLocation:  c.BackupLocation,
		MemoryLayout:    c.MemoryLayout,
		PositionAddress: c.PositionAddress,
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
	return nil
}
