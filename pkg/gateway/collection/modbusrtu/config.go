package modbusrtu

import "encoding/json"

type protocolConfig struct {
	Location        string `json:"location"`
	BaudRate        int    `json:"baudRate"`
	DataBits        int    `json:"dataBits"`
	Parity          string `json:"parity"`
	StopBits        string `json:"stopBits"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
}

func (c *ModbusRtuCollection) ConfigJSON() ([]byte, error) {
	return json.Marshal(protocolConfig{
		Location:        c.Location,
		BaudRate:        c.BaudRate,
		DataBits:        c.DataBits,
		Parity:          c.Parity,
		StopBits:        c.StopBits,
		MemoryLayout:    c.MemoryLayout,
		PositionAddress: c.PositionAddress,
	})
}

func (c *ModbusRtuCollection) ApplyConfig(raw []byte) error {
	var config protocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	c.Location = config.Location
	c.BaudRate = config.BaudRate
	c.DataBits = config.DataBits
	c.Parity = config.Parity
	c.StopBits = config.StopBits
	c.MemoryLayout = config.MemoryLayout
	c.PositionAddress = config.PositionAddress
	if c.BaudRate == 0 {
		c.BaudRate = 9600
	}
	if c.DataBits == 0 {
		c.DataBits = 8
	}
	if c.Parity == "" {
		c.Parity = "noParity"
	}
	if c.StopBits == "" {
		c.StopBits = "1"
	}
	return nil
}
