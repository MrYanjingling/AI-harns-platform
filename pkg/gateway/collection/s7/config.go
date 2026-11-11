package s7

import "encoding/json"

type protocolConfig struct {
	Location       string      `json:"location"`
	Port           int         `json:"port"`
	Rack           uint8       `json:"rack"`
	Slot           uint8       `json:"slot"`
	PDUSize        FlexibleInt `json:"pduSize"`
	ConnectionType FlexibleInt `json:"connectionType"`
	TCPCount       int         `json:"tcpCount"`
}

func (c *S7Collection) ConfigJSON() ([]byte, error) {
	tcpCount := c.TCPCount
	if tcpCount == 0 {
		tcpCount = DefaultTCPCount
	}
	pduSize := c.PDUSize
	if pduSize == 0 {
		pduSize = DefaultPDUSize
	}
	connectionType := c.ConnectionType
	if connectionType == 0 {
		connectionType = DefaultConnectionType
	}
	port := c.Port
	if port == 0 {
		port = DefaultPort
	}
	return json.Marshal(protocolConfig{
		Location:       c.Location,
		Port:           port,
		Rack:           c.Rack,
		Slot:           c.Slot,
		PDUSize:        FlexibleInt(pduSize),
		ConnectionType: FlexibleInt(connectionType),
		TCPCount:       tcpCount,
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
	c.PDUSize = int(config.PDUSize)
	c.ConnectionType = uint8(config.ConnectionType)
	c.TCPCount = config.TCPCount
	if c.PDUSize == 0 {
		c.PDUSize = DefaultPDUSize
	}
	if c.ConnectionType == 0 {
		c.ConnectionType = DefaultConnectionType
	}
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
	return nil
}
