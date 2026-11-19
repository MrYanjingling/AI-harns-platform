package s7

import "encoding/json"

type protocolConfig struct {
	Location       string       `json:"location"`
	Port           int          `json:"port"`
	Rack           uint8        `json:"rack"`
	Slot           uint8        `json:"slot"`
	PDUSize        FlexibleInt  `json:"pduSize"`
	ConnectionType FlexibleInt  `json:"connectionType"`
	Concurrency    int          `json:"concurrency"`
	LocalTSAP      FlexibleTSAP `json:"localTsap,omitempty"`
	RemoteTSAP     FlexibleTSAP `json:"remoteTsap,omitempty"`
}

type s7200ProtocolConfig struct {
	Location    string       `json:"location"`
	Port        int          `json:"port"`
	PDUSize     FlexibleInt  `json:"pduSize"`
	Concurrency int          `json:"concurrency"`
	LocalTSAP   FlexibleTSAP `json:"localTsap"`
	RemoteTSAP  FlexibleTSAP `json:"remoteTsap"`
}

func (c *S7Collection) ConfigJSON() ([]byte, error) {
	concurrency := c.Concurrency
	if concurrency == 0 {
		concurrency = DefaultConcurrency
	}
	pduSize := c.PDUSize
	if pduSize == 0 {
		pduSize = DefaultPDUSize
	}
	port := c.Port
	if port == 0 {
		port = DefaultPort
	}
	localTSAP := c.LocalTSAP
	if localTSAP == 0 {
		localTSAP = DefaultLocalTSAP
	}
	remoteTSAP := c.RemoteTSAP
	if remoteTSAP == 0 {
		remoteTSAP = DefaultRemoteTSAP
	}
	if c.GetDriverModel() == DriverModelS7200 {
		return json.Marshal(s7200ProtocolConfig{
			Location:    c.Location,
			Port:        port,
			PDUSize:     FlexibleInt(pduSize),
			Concurrency: concurrency,
			LocalTSAP:   FlexibleTSAP(localTSAP),
			RemoteTSAP:  FlexibleTSAP(remoteTSAP),
		})
	}
	connectionType := c.ConnectionType
	if connectionType == 0 {
		connectionType = DefaultConnectionType
	}
	return json.Marshal(protocolConfig{
		Location:       c.Location,
		Port:           port,
		Rack:           c.Rack,
		Slot:           c.Slot,
		PDUSize:        FlexibleInt(pduSize),
		ConnectionType: FlexibleInt(connectionType),
		Concurrency:    concurrency,
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
	c.Concurrency = config.Concurrency
	c.LocalTSAP = uint16(config.LocalTSAP)
	c.RemoteTSAP = uint16(config.RemoteTSAP)
	if c.PDUSize == 0 {
		c.PDUSize = DefaultPDUSize
	}
	if c.GetDriverModel() == DriverModelS7200 {
		if c.LocalTSAP == 0 {
			c.LocalTSAP = DefaultLocalTSAP
		}
		if c.RemoteTSAP == 0 {
			c.RemoteTSAP = DefaultRemoteTSAP
		}
	} else if c.ConnectionType == 0 {
		c.ConnectionType = DefaultConnectionType
	}
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConcurrency
	}
	return nil
}
