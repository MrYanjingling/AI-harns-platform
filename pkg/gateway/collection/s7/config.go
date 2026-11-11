package s7

import "encoding/json"

type protocolConfig struct {
	Location string `json:"location"`
	Port     int    `json:"port"`
	Rack     uint8  `json:"rack"`
	Slot     uint8  `json:"slot"`
	TCPCount int    `json:"tcpCount"`
}

func (c *S7Collection) ConfigJSON() ([]byte, error) {
	tcpCount := c.TCPCount
	if tcpCount == 0 {
		tcpCount = DefaultTCPCount
	}
	port := c.Port
	if port == 0 {
		port = DefaultPort
	}
	return json.Marshal(protocolConfig{
		Location: c.Location,
		Port:     port,
		Rack:     c.Rack,
		Slot:     c.Slot,
		TCPCount: tcpCount,
	})
}

func (c *S7Collection) ApplyConfig(raw []byte) error {
	var config protocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	c.Location = config.Location
	c.Port = config.Port
	c.Rack = config.Rack
	c.Slot = config.Slot
	c.TCPCount = config.TCPCount
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
	return nil
}
