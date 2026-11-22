package iec102

import "encoding/json"

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location: c.Location, Host: c.Location, Link: c.Link,
		LA: c.LA, CA: c.CA, Interval: c.Interval, Timeout: c.Timeout,
		Device: c.Device, Baud: c.Baud, Parity: c.Parity, Data: c.Data, StopBits: c.StopBits,
	}
	config.Normalize()
	return json.Marshal(config)
}

func (c *Collection) ApplyConfig(raw []byte) error {
	var config ProtocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	config.Normalize()
	c.Location = config.Location
	c.Link = config.Link
	c.LA = config.LA
	c.CA = config.CA
	c.Interval = config.Interval
	c.Timeout = config.Timeout
	c.Device = config.Device
	c.Baud = config.Baud
	c.Parity = config.Parity
	c.Data = config.Data
	c.StopBits = config.StopBits
	return nil
}
