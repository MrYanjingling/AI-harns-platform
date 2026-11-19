package v1

type OpenReq struct {
	Port int `json:"port"`
}

type CreatePointReq struct {
	Address      uint16 `json:"address"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RegisterType string `json:"registerType"`
	DataType     string `json:"dataType"`
	Mode         string `json:"mode"`
}
