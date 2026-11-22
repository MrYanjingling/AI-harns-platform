package iec103

import "encoding/json"

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{Location: c.Location, Host: c.Location, Link: c.Link,
		Interval: c.Interval, Timeout: c.Timeout, Device: c.Device, Baud: c.Baud,
		Parity: c.Parity, Data: c.Data, StopBits: c.StopBits}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.Location, c.Link = config.Location, config.Link
	c.Interval, c.Timeout = config.Interval, config.Timeout
	c.Device, c.Baud, c.Parity = config.Device, config.Baud, config.Parity
	c.Data, c.StopBits = config.Data, config.StopBits
	return nil
}
