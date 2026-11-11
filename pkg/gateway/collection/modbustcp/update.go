package modbustcp

// UpdateReq updates a Modbus TCP collection without touching variables.
type UpdateReq struct {
	Name             string         `json:"name"`
	CollectorCycleMs uint64         `json:"collectorCycleMs"`
	ProtocolConfig   ProtocolConfig `json:"protocolConfig"`
}

// ApplyUpdate applies mutable Modbus TCP fields.
func (c *ModbusTcpCollection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.BackupLocation = req.ProtocolConfig.BackupLocation
	c.MemoryLayout = req.ProtocolConfig.MemoryLayout
	c.PositionAddress = req.ProtocolConfig.PositionAddress
	c.TCPCount = req.ProtocolConfig.TCPCount
}
