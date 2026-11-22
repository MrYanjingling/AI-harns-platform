package iec104

import "encoding/json"

func (c *Collection) ConfigJSON() ([]byte, error) {
	config := ProtocolConfig{
		Location: c.Location, Port: c.Port, CA: c.CA, Interval: c.Interval,
		SyncTimeInterval: c.SyncTimeInterval, RemotePulseInterval: c.RemotePulseInterval,
		T0: c.T0, T1: c.T1, T2: c.T2, T3: c.T3, W: c.W, Timeout: c.Timeout, Active: c.Active,
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
	c.Port = config.Port
	c.CA = config.CA
	c.Interval = config.Interval
	c.SyncTimeInterval = config.SyncTimeInterval
	c.RemotePulseInterval = config.RemotePulseInterval
	c.T0 = config.T0
	c.T1 = config.T1
	c.T2 = config.T2
	c.T3 = config.T3
	c.W = config.W
	c.Timeout = config.Timeout
	c.Active = config.Active
	return nil
}
