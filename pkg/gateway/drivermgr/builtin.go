package drivermgr

const (
	DriverTypeModbus = "modbus"
	DriverTypeS7     = "s7"
	DriverTypeOpcUa  = "opcUa"

	ModelModbusTCP        = "modbusTcp"
	ModelModbusRTU        = "modbusRtu"
	ModelModbusRTUOverTCP = "modbusRtuOverTcp"
	ModelS71500           = "s71500"
	ModelS7200            = "s7-200"
	ModelOpcUa            = "opcUa"

	DriverModbusTCP        = "modbus-tcp"
	DriverModbusRTU        = "modbus-rtu"
	DriverModbusRTUOverTCP = "modbus-rtu-over-tcp"
	DriverS71500           = "s7-1500"
	DriverS7200            = "s7-200"
	DriverOpcUa            = "opcua"

	BuiltinModbusTCPID        = "01M706HSSR3Q8KY52DPGDWG9KE"
	BuiltinModbusRTUID        = "01M706HSSR3Q8KY52DPJ6XTA17"
	BuiltinModbusRTUOverTCPID = "01M706HSSR3Q8KY52DPNYR9MW5"
	BuiltinS71500ID           = "01M706HSSR3Q8KY52DPQZHT18E"
	BuiltinS7200ID            = "01M706HSSR3Q8KY52DPQZHT18F"
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
    },
    {
      "name": "tcpCount",
      "labelZh": "TCP 连接数",
      "labelEn": "TCP Connections",
      "regex": "^[1-3]$",
      "sort": 5,
      "type": "select",
      "options": [
        {"value": "1", "labelZh": "1", "labelEn": "1"},
        {"value": "2", "labelZh": "2", "labelEn": "2"},
        {"value": "3", "labelZh": "3", "labelEn": "3"}
      ]
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "1!40001",
    "placeholderEn": "1!40001",
    "descriptionZh": "格式：从站地址!功能码地址，例如 1!40001",
    "descriptionEn": "Format: slaveId!functionCodeAddress, for example 1!40001"

  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]{1,3}![1-4][0-9]{1,5}$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"}
  ]
}`

const modbusRtuSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "串口设备",
      "labelEn": "Serial Port",
      "regex": "^(/dev/[A-Za-z0-9._/-]+|COM[0-9]{1,3})$",
      "sort": 1,
      "type": "input",
      "required": true
    },
    {
      "name": "baudRate",
      "labelZh": "波特率",
      "labelEn": "Baud Rate",
      "regex": "^(1200|2400|4800|9600|19200|38400|57600|115200)$",
      "sort": 2,
      "type": "select",
      "options": [
        {"value":"1200","labelZh":"1200","labelEn":"1200"},
        {"value":"2400","labelZh":"2400","labelEn":"2400"},
        {"value":"4800","labelZh":"4800","labelEn":"4800"},
        {"value":"9600","labelZh":"9600","labelEn":"9600"},
        {"value":"19200","labelZh":"19200","labelEn":"19200"},
        {"value":"38400","labelZh":"38400","labelEn":"38400"},
        {"value":"57600","labelZh":"57600","labelEn":"57600"},
        {"value":"115200","labelZh":"115200","labelEn":"115200"}
      ]
    },
    {
      "name": "dataBits",
      "labelZh": "数据位",
      "labelEn": "Data Bits",
      "regex": "^(5|6|7|8)$",
      "sort": 3,
      "type": "select",
      "options": [
        {"value":"5","labelZh":"5","labelEn":"5"},
        {"value":"6","labelZh":"6","labelEn":"6"},
        {"value":"7","labelZh":"7","labelEn":"7"},
        {"value":"8","labelZh":"8","labelEn":"8"}
      ]
    },
    {
      "name": "parity",
      "labelZh": "校验位",
      "labelEn": "Parity",
      "regex": "^(noParity|oddParity|evenParity)$",
      "sort": 4,
      "type": "select",
      "options": [
        {"value":"noParity","labelZh":"无校验","labelEn":"None"},
        {"value":"oddParity","labelZh":"奇校验","labelEn":"Odd"},
        {"value":"evenParity","labelZh":"偶校验","labelEn":"Even"}
      ]
    },
    {
      "name": "stopBits",
      "labelZh": "停止位",
      "labelEn": "Stop Bits",
      "regex": "^(1|1\\.5|2)$",
      "sort": 5,
      "type": "select",
      "options": [
        {"value":"1","labelZh":"1","labelEn":"1"},
        {"value":"1.5","labelZh":"1.5","labelEn":"1.5"},
        {"value":"2","labelZh":"2","labelEn":"2"}
      ]
    },
    {
      "name": "memoryLayout",
      "labelZh": "内存布局",
      "labelEn": "Memory Layout",
      "regex": "^(ABCD|BADC|CDAB|DCBA)$",
      "sort": 6,
      "type": "select",
      "required": true,
      "options": [
        {"value":"ABCD","labelZh":"ABCD","labelEn":"ABCD"},
        {"value":"BADC","labelZh":"BADC","labelEn":"BADC"},
        {"value":"CDAB","labelZh":"CDAB","labelEn":"CDAB"},
        {"value":"DCBA","labelZh":"DCBA","labelEn":"DCBA"}
      ]
    },
    {
      "name": "positionAddress",
      "labelZh": "起始地址",
      "labelEn": "Position Address",
      "regex": "^(0|1)$",
      "sort": 7,
      "type": "input"
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "1!40001",
    "placeholderEn": "1!40001",
    "descriptionZh": "格式：从站地址!功能码地址，例如 1!40001",
    "descriptionEn": "Format: slaveId!functionCodeAddress, for example 1!40001"

  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]{1,3}![1-4][0-9]{1,5}$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"}
  ]
}`

const modbusRtuOverTcpSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "IP地址:端口",
      "labelEn": "Host:Port",
      "regex": "^[A-Za-z0-9.-]+:[0-9]{1,5}$",
      "sort": 1,
      "type": "input",
      "required": true
    },
    {
      "name": "backupLocation",
      "labelZh": "备用地址",
      "labelEn": "Backup Host:Port",
      "regex": "^[A-Za-z0-9.-]+:[0-9]{1,5}$",
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
        {"value":"ABCD","labelZh":"ABCD","labelEn":"ABCD"},
        {"value":"BADC","labelZh":"BADC","labelEn":"BADC"},
        {"value":"CDAB","labelZh":"CDAB","labelEn":"CDAB"},
        {"value":"DCBA","labelZh":"DCBA","labelEn":"DCBA"}
      ]
    },
    {
      "name": "positionAddress",
      "labelZh": "起始地址",
      "labelEn": "Position Address",
      "regex": "^(0|1)$",
      "sort": 4,
      "type": "input"
    },
    {
      "name": "tcpCount",
      "labelZh": "TCP 连接数",
      "labelEn": "TCP Connections",
      "regex": "^[1-3]$",
      "sort": 5,
      "type": "select",
      "options": [
        {"value":"1","labelZh":"1","labelEn":"1"},
        {"value":"2","labelZh":"2","labelEn":"2"},
        {"value":"3","labelZh":"3","labelEn":"3"}
      ]
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "1!40001",
    "placeholderEn": "1!40001",
    "descriptionZh": "格式：从站地址!功能码地址，例如 1!40001",
    "descriptionEn": "Format: slaveId!functionCodeAddress, for example 1!40001"

  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]{1,3}![1-4][0-9]{1,5}$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]{1,3}![34][0-9]{1,5}$"}
  ]
}`

const s71500Schema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "PLC 地址",
      "labelEn": "PLC Address",
      "regex": "^[A-Za-z0-9.-]+$",
      "sort": 1,
      "type": "input",
      "required": true
    },
    {
      "name": "port",
      "labelZh": "端口",
      "labelEn": "Port",
      "regex": "^[0-9]{1,5}$",
      "sort": 2,
      "type": "input"
    },
    {
      "name": "rack",
      "labelZh": "Rack",
      "labelEn": "Rack",
      "regex": "^[0-7]$",
      "sort": 3,
      "type": "input"
    },
    {
      "name": "slot",
      "labelZh": "Slot",
      "labelEn": "Slot",
      "regex": "^[0-9]{1,2}$",
      "sort": 4,
      "type": "input"
    },
    {
      "name": "pduSize",
      "labelZh": "PDU 大小",
      "labelEn": "PDU Size",
      "regex": "^(480|960)$",
      "sort": 5,
      "type": "select",
      "options": [
        {"value":"480","labelZh":"480","labelEn":"480"},
        {"value":"960","labelZh":"960","labelEn":"960"}
      ]
    },
    {
      "name": "connectionType",
      "labelZh": "连接类型",
      "labelEn": "Connection Type",
      "regex": "^[1-3]$",
      "sort": 6,
      "type": "select",
      "options": [
        {"value":"1","labelZh":"PG","labelEn":"PG"},
        {"value":"2","labelZh":"OP","labelEn":"OP"},
        {"value":"3","labelZh":"PC","labelEn":"PC"}
      ]
    },
    {
      "name": "tcpCount",
      "labelZh": "TCP 连接数",
      "labelEn": "TCP Connections",
      "regex": "^[1-3]$",
      "sort": 7,
      "type": "select",
      "options": [
        {"value":"1","labelZh":"1","labelEn":"1"},
        {"value":"2","labelZh":"2","labelEn":"2"},
        {"value":"3","labelZh":"3","labelEn":"3"}
      ]
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "DB1.DBW0 / DB1.DBW0.10",
    "placeholderEn": "DB1.DBW0 / DB1.DBW0.10",
    "descriptionZh": "支持 DB1.DBX0.0、DB1.DBB0、DB1.DBW0、DB1.DBD0、DB1.DBW0.10、I0、IW0、I0.0、Q0、QW0、Q0.0、M0、MW0、M0.0",
    "descriptionEn": "Supports DB1.DBX0.0, DB1.DBB0, DB1.DBW0, DB1.DBD0, DB1.DBW0.10, I0, IW0, I0.0, Q0, QW0, Q0.0, M0, MW0, M0.0"

  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^((DB[0-9]{1,5}\\.DBX[0-9]{1,5}\\.[0-7])|([IQM][XBWD]?[0-9]{1,5}(\\.[0-7])?))$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^((DB[0-9]{1,5}\\.DBX[0-9]{1,5}\\.[0-7])|([IQM][XBWD]?[0-9]{1,5}(\\.[0-7])?))$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5}))$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5}))$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5}))$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5}))$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5}))$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5}))$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5}\\.[0-9]{1,5}$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5}\\.[0-9]{1,5}$"}
  ]
}`

const s7200Schema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "PLC 地址",
      "labelEn": "PLC Address",
      "regex": "^[A-Za-z0-9.-]+$",
      "sort": 1,
      "type": "input",
      "required": true
    },
    {
      "name": "port",
      "labelZh": "端口",
      "labelEn": "Port",
      "regex": "^[0-9]{1,5}$",
      "sort": 2,
      "type": "input"
    },
    {
      "name": "pduSize",
      "labelZh": "PDU 大小",
      "labelEn": "PDU Size",
      "regex": "^(480|960)$",
      "sort": 3,
      "type": "select",
      "options": [
        {"value":"480","labelZh":"480","labelEn":"480"},
        {"value":"960","labelZh":"960","labelEn":"960"}
      ]
    },
    {
      "name": "tcpCount",
      "labelZh": "TCP 连接数",
      "labelEn": "TCP Connections",
      "regex": "^[1-3]$",
      "sort": 4,
      "type": "select",
      "options": [
        {"value":"1","labelZh":"1","labelEn":"1"},
        {"value":"2","labelZh":"2","labelEn":"2"},
        {"value":"3","labelZh":"3","labelEn":"3"}
      ]
    },
    {
      "name": "localTsap",
      "labelZh": "本地 TSAP",
      "labelEn": "Local TSAP",
      "regex": "^0[xX][0-9A-Fa-f]{1,4}$",
      "sort": 5,
      "type": "input"
    },
    {
      "name": "remoteTsap",
      "labelZh": "远程 TSAP",
      "labelEn": "Remote TSAP",
      "regex": "^0[xX][0-9A-Fa-f]{1,4}$",
      "sort": 6,
      "type": "input"
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "VW0 / V0.0 / DB1.DBW0",
    "placeholderEn": "VW0 / V0.0 / DB1.DBW0",
    "descriptionZh": "支持 V0.0、VB0、VW0、VD0；V 区映射到 DB1。也支持 DB/I/Q/M 地址",
    "descriptionEn": "Supports V0.0, VB0, VW0, VD0; V area maps to DB1. DB/I/Q/M addresses are also supported"

  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^((DB[0-9]{1,5}\\.DBX[0-9]{1,5}\\.[0-7])|([IQM][XBWD]?[0-9]{1,5}(\\.[0-7])?)|(V[0-9]{1,5}\\.[0-7]))$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^((DB[0-9]{1,5}\\.DBX[0-9]{1,5}\\.[0-7])|([IQM][XBWD]?[0-9]{1,5}(\\.[0-7])?)|(V[0-9]{1,5}\\.[0-7]))$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5})|(V[BDW][0-9]{1,5}))$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5})|(V[BDW][0-9]{1,5}))$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5})|(V[BDW][0-9]{1,5}))$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5})|(V[BDW][0-9]{1,5}))$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5})|(V[BDW][0-9]{1,5}))$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|([IQM][XBWD]?[0-9]{1,5})|(V[BDW][0-9]{1,5}))$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5}\\.[0-9]{1,5}$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5}\\.[0-9]{1,5}$"}
  ]
}`

const opcUaSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "Endpoint",
      "labelEn": "Endpoint",
      "regex": "^opc\\.tcp://.+$",
      "sort": 1,
      "type": "input",
      "required": true
    },
    {
      "name": "port",
      "labelZh": "端口",
      "labelEn": "Port",
      "regex": "^[0-9]{1,5}$",
      "sort": 2,
      "type": "input",
      "required": true
    },
    {
      "name": "username",
      "labelZh": "用户名",
      "labelEn": "Username",
      "regex": "^.*$",
      "sort": 3,
      "type": "input"
    },
    {
      "name": "password",
      "labelZh": "密码",
      "labelEn": "Password",
      "regex": "^.*$",
      "sort": 4,
      "type": "input"
    },
    {
      "name": "tcpCount",
      "labelZh": "TCP 连接数",
      "labelEn": "TCP Connections",
      "regex": "^[1-3]$",
      "sort": 5,
      "type": "select",
      "options": [
        {"value":"1","labelZh":"1","labelEn":"1"},
        {"value":"2","labelZh":"2","labelEn":"2"},
        {"value":"3","labelZh":"3","labelEn":"3"}
      ]
    }
  ],
  "variableAddress": {
    "labelZh": "NodeID",
    "labelEn": "NodeID",
    "placeholderZh": "ns=2;s=Tag",
    "placeholderEn": "ns=2;s=Tag",
    "descriptionZh": "完整 NodeID，例如 ns=2;s=Tag 或 ns=2;i=1001",
    "descriptionEn": "Full NodeID, for example ns=2;s=Tag or ns=2;i=1001"

  },
  "variableDataTypes": [
    {"value":"NUMBER","labelZh":"number","labelEn":"number","addressRegex":"^(ns=[0-9]{1,5};)?[isgb]=.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^(ns=[0-9]{1,5};)?[isgb]=.+$"}
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
	{BuiltinModbusRTUID, DriverTypeModbus, ModelModbusRTU, DriverModbusRTU, modbusRtuSchema},
	{BuiltinModbusRTUOverTCPID, DriverTypeModbus, ModelModbusRTUOverTCP, DriverModbusRTUOverTCP, modbusRtuOverTcpSchema},
	{BuiltinS71500ID, DriverTypeS7, ModelS71500, DriverS71500, s71500Schema},
	{BuiltinS7200ID, DriverTypeS7, ModelS7200, DriverS7200, s7200Schema},
	{BuiltinOpcUaID, DriverTypeOpcUa, ModelOpcUa, DriverOpcUa, opcUaSchema},
}
