package drivermgr

const (
	DriverTypeModbus = "modbus"
	DriverTypeS7     = "s7"
	DriverTypeOpcUa  = "opcUa"

	ModelModbusTCP        = "modbusTcp"
	ModelModbusRTU        = "modbusRtu"
	ModelModbusRTUOverTCP = "modbusRtuOverTcp"
	ModelS71500           = "s71500"
	ModelOpcUa            = "opcUa"

	DriverModbusTCP        = "modbus_tcp"
	DriverModbusRTU        = "modbus_rtu"
	DriverModbusRTUOverTCP = "modbus_rtu_over_tcp"
	DriverS71500           = "s7_1500"
	DriverOpcUa            = "opcua"

	BuiltinModbusTCPID        = "01M706HSSR3Q8KY52DPGDWG9KE"
	BuiltinModbusRTUID        = "01M706HSSR3Q8KY52DPJ6XTA17"
	BuiltinModbusRTUOverTCPID = "01M706HSSR3Q8KY52DPNYR9MW5"
	BuiltinS71500ID           = "01M706HSSR3Q8KY52DPQZHT18E"
	BuiltinOpcUaID            = "01M706HSSR3Q8KY52DPSV88HRY"
)

const emptySchema = `{"fields":[]}`

const modbusTcpSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "IP地址:端口",
      "labelEn": "Host:Port",
      "regex": "^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9]):(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3})$",
      "sort": 1,
      "type": "input",
      "required": true
    },
    {
      "name": "backupLocation",
      "labelZh": "备用地址",
      "labelEn": "Backup Host:Port",
      "regex": "^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9]):(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3})$",
      "sort": 2,
      "type": "input"
    },
    {
      "name": "memoryLayout",
      "labelZh": "内存布局",
      "labelEn": "Memory Layout",
      "regex": "^(ABCD|BADC|CDAB|DCBA)$",
      "sort": 3,
      "type": "select",
      "required": true,
      "options": [
        {
          "value": "ABCD",
          "labelZh": "ABCD",
          "labelEn": "ABCD"
        },
        {
          "value": "BADC",
          "labelZh": "BADC",
          "labelEn": "BADC"
        },
        {
          "value": "CDAB",
          "labelZh": "CDAB",
          "labelEn": "CDAB"
        },
        {
          "value": "DCBA",
          "labelZh": "DCBA",
          "labelEn": "DCBA"
        }
      ]
    },
    {
      "name": "positionAddress",
      "labelZh": "起始地址",
      "labelEn": "Position Address",
      "regex": "^(0|1)$",
      "sort": 4,
      "type": "input"
    }
  ]
}`

// BuiltinDriver is a driver shipped with the gateway binary.
type BuiltinDriver struct {
	ID         string
	DriverType string
	Model      string
	PluginName string
	Schema     string
}

// BuiltinDrivers is the source of truth for built-in driver metadata.
var BuiltinDrivers = []BuiltinDriver{
	{BuiltinModbusTCPID, DriverTypeModbus, ModelModbusTCP, DriverModbusTCP, modbusTcpSchema},
	{BuiltinModbusRTUID, DriverTypeModbus, ModelModbusRTU, DriverModbusRTU, emptySchema},
	{BuiltinModbusRTUOverTCPID, DriverTypeModbus, ModelModbusRTUOverTCP, DriverModbusRTUOverTCP, emptySchema},
	{BuiltinS71500ID, DriverTypeS7, ModelS71500, DriverS71500, emptySchema},
	{BuiltinOpcUaID, DriverTypeOpcUa, ModelOpcUa, DriverOpcUa, emptySchema},
}
