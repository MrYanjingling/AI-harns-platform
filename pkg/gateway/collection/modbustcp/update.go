package modbustcp

// UpdateReq updates a Modbus TCP collection without touching variables.
type UpdateReq struct {
	Name             string `json:"name"`
	CollectorCycleMs uint64 `json:"collectorCycleMs"`

	Location        string `json:"location"`
	BackupLocation  string `json:"backupLocation,omitempty"`
	MemoryLayout    string `json:"memoryLayout"`
	PositionAddress uint   `json:"positionAddress,omitempty"`
}

// ApplyUpdate applies mutable Modbus TCP fields.
func (c *ModbusTcpCollection) ApplyUpdate(req *UpdateReq) {
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.Location
	c.BackupLocation = req.BackupLocation
	c.MemoryLayout = req.MemoryLayout
	c.PositionAddress = req.PositionAddress
}
