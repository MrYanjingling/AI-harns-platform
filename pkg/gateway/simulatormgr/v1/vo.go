package v1

type StateVo struct {
	Running      bool       `json:"running"`
	Port         int        `json:"port"`
	UnitID       uint8      `json:"unitId"`
	MemoryLayout string     `json:"memoryLayout"`
	Points       []*PointVo `json:"points"`
}

type PointVo struct {
	Address      uint16 `json:"address"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RegisterType string `json:"registerType"`
	DataType     string `json:"dataType"`
	Mode         string `json:"mode"`
	Value        any    `json:"value"`
}
