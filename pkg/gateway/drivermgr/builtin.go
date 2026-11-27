package drivermgr

const (
	DriverTypeModbus     = "modbus"
	DriverTypeS7         = "siemens"
	DriverTypeOpcUa      = "opcUa"
	DriverTypeOmron      = "omron"
	DriverTypePanasonic  = "Panasonic"
	DriverTypeIEC        = "IEC"
	DriverTypeABB        = "ABB"
	DriverTypeMitsubishi = "Mitsubishi"

	DriverTypeEthernetIP   = "ethernetIp"
	DriverTypeAllenBradley = "Allen-Bradley"
	DriverTypeEtherCAT     = "ethercat"
	DriverTypeProfinet     = "profinet"
	DriverTypeDNP3         = "dnp3"
	DriverTypeDLT645       = "dlt645"
	DriverTypeCJT188       = "cjt188"
	DriverTypeDLT698       = "dlt698"
	DriverTypeKNX          = "KNX"
	DriverTypeCNC          = "CNC"
	DriverTypeKuka         = "Kuka"
	DriverTypeCodesys      = "Codesys"
	DriverTypeAB5000       = "Allen-Bradley"
	DriverTypeHJ212        = "HJ212"
	DriverTypeBACnet       = "BACnet"
	DriverTypeADS          = "TwinCAT"
	DriverTypeSRTP         = "GE/Emerson"
	DriverTypeSNMP         = "SNMP"

	ModelModbusTCP        = "modbusTcp"
	ModelModbusRTU        = "modbusRtu"
	ModelModbusRTUOverTCP = "modbusRtuOverTcp"
	ModelModbusASCII      = "Modbus Ascii"
	ModelS71500           = "s7-1500"
	ModelS7200            = "s7-200"
	ModelOpcUa            = "opcUa"
	ModelFins             = "fins"
	ModelHostlinkCmode    = "hostlinkCmode"
	ModelOmronCip         = "omronCip"
	ModelMewtocol         = "Mewtocol"
	ModelIEC101           = "IEC101"
	ModelIEC102           = "IEC102"
	ModelIEC103           = "IEC103"
	ModelIEC104           = "IEC104"
	ModelCOMLI            = "COMLI"
	ModelMCFX             = "MC-FX"
	ModelMC1E             = "MC-1E"
	ModelMC3E             = "MC-3E"
	ModelMC4E             = "MC-4E"
	ModelEthernetIP       = "ethernetIp"
	ModelEthernetIP5500   = "EtherNet/IP-ControlLogic 55xx"
	ModelEthernetIP1400   = "EtherNet/IP-MicroLogix 1400"
	ModelEtherCAT         = "ethercat"
	ModelProfinet         = "profinet"
	ModelDNP3             = "dnp3"
	ModelDLT6452007       = "dlt645_2007"
	ModelDLT6451997       = "dlt645_1997"
	ModelCJT1882004       = "cjt188_2004"
	ModelCJT1882018       = "cjt188_2018"
	ModelDLT69845         = "dlt698_45"
	ModelMPI              = "mpi"
	ModelPPI              = "ppi"
	ModelFetchWrite       = "FetchWrite"
	ModelKNXnetIP         = "KNXnet/IP"
	ModelKND              = "KND"
	ModelHeidenhain       = "heidenhain"
	ModelFanucFocas       = "fanuc focas"
	ModelMTConnect        = "MTConnect"
	ModelKuka             = "Kuka"
	ModelCodesysV3        = "codesys-v3"
	ModelAB5000           = "ab5000"
	ModelDF1              = "DF1"
	ModelHJ212            = "HJ212"
	ModelBACnet           = "BACnet"
	ModelBeckhoffADS      = "Beckhoff ADS"
	ModelSRTP             = "SRTP"
	ModelSNMPV2           = "SNMP-V2"
	ModelIEC61850         = "IEC61850"

	DriverModbusTCP        = "modbus-tcp"
	DriverModbusRTU        = "modbus-rtu"
	DriverModbusRTUOverTCP = "modbus-rtu-over-tcp"
	DriverModbusASCII      = "modbus-ascii"
	DriverS71500           = "s7-1500"
	DriverS7200            = "s7-200"
	DriverOpcUa            = "opcua"
	DriverFins             = "omron-fins"
	DriverHostlinkCmode    = "omron-hostlink-cmode"
	DriverOmronCip         = "omron-cip"
	DriverMewtocol         = "panasonic-mewtocol"
	DriverIEC101           = "iec101"
	DriverIEC102           = "iec102"
	DriverIEC103           = "iec103"
	DriverIEC104           = "iec104"
	DriverCOMLI            = "comli"
	DriverFX               = "fx"
	DriverA1E              = "a1e"
	DriverQnA3E            = "qna3e"
	DriverQnA4E            = "qna4e"
	DriverEthernetIP       = "ethernet-ip"
	DriverEthernetIP5500   = "ethernet-ip-5500"
	DriverEthernetIP1400   = "ethernet-ip-1400"
	DriverEtherCAT         = "ethercat"
	DriverProfinet         = "profinet"
	DriverDNP3             = "dnp3"
	DriverDLT6452007       = "dlt645-2007"
	DriverDLT6451997       = "dlt645-1997"
	DriverCJT1882004       = "cjt188-2004"
	DriverCJT1882018       = "cjt188-2018"
	DriverDLT69845         = "dlt698-45"
	DriverMPI              = "mpi"
	DriverPPI              = "ppi"
	DriverFetchWrite       = "siemens-fetch-write"
	DriverKNXnetIP         = "knxnet-ip"
	DriverKND              = "knd"
	DriverHeidenhain       = "heidenhain"
	DriverFanucFocas       = "fanuc-focas"
	DriverMTConnect        = "mtconnect"
	DriverKuka             = "kuka"
	DriverCodesysV3        = "codesys-v3"
	DriverAB5000           = "ab5000"
	DriverDF1              = "df1"
	DriverHJ212            = "hj212"
	DriverBACnet           = "bacnet"
	DriverADS              = "ads"
	DriverSRTP             = "srtp"
	DriverSNMP             = "snmp"
	DriverIEC61850         = "iec61850"

	BuiltinModbusTCPID        = "01M706HSSR3Q8KY52DPGDWG9KE"
	BuiltinModbusRTUID        = "01M706HSSR3Q8KY52DPJ6XTA17"
	BuiltinModbusRTUOverTCPID = "01M706HSSR3Q8KY52DPNYR9MW5"
	BuiltinModbusASCIIID      = "01M7Z9X3Q8KY52DPGDWG9KEXYZ"
	BuiltinS71500ID           = "01M706HSSR3Q8KY52DPQZHT18E"
	BuiltinS7200ID            = "01M706HSSR3Q8KY52DPQZHT18F"
	BuiltinOpcUaID            = "01M706HSSR3Q8KY52DPSV88HRY"
	BuiltinFinsID             = "01M706HSSR3Q8KY52DPSV88HRZ"
	BuiltinHostlinkCmodeID    = "01M706HSSR3Q8KY52DPSV88HS1"
	BuiltinMewtocolID         = "01M8C0MEWT0C0L1A2B3C4D5E6F"
	BuiltinOmronCipID         = "01M706HSSR3Q8KY52DPSV88HS2"
	BuiltinIEC101ID           = "01M7Z9X3Q8KY52DPGDWG9KEX01"
	BuiltinIEC102ID           = "01M7Z9X3Q8KY52DPGDWG9KEX02"
	BuiltinIEC103ID           = "01M7Z9X3Q8KY52DPGDWG9KEX03"
	BuiltinIEC104ID           = "01M7Z9X3Q8KY52DPGDWG9KEX04"
	BuiltinEthernetIPID       = "01M7Z9X3Q8KY52DPGDWG9KEX05"
	BuiltinEthernetIP5500ID   = "01M7Z9X3Q8KY52DPGDWG9KEX08"
	BuiltinEthernetIP1400ID   = "01M7Z9X3Q8KY52DPGDWG9KEX09"
	BuiltinEtherCATID         = "01M7Z9X3Q8KY52DPGDWG9KEX06"
	BuiltinProfinetID         = "01M7Z9X3Q8KY52DPGDWG9KEX07"
	BuiltinDNP3ID             = "01M7Z9X3Q8KY52DPGDWG9KEX10"
	BuiltinDLT6452007ID       = "01M7Z9X3Q8KY52DPGDWG9KEX11"
	BuiltinDLT6451997ID       = "01M7Z9X3Q8KY52DPGDWG9KEX12"
	BuiltinCJT1882004ID       = "01M7Z9X3Q8KY52DPGDWG9KEX13"
	BuiltinCJT1882018ID       = "01M7Z9X3Q8KY52DPGDWG9KEX14"
	BuiltinDLT69845ID         = "01M7Z9X3Q8KY52DPGDWG9KEX15"
	BuiltinMPIID              = "01M7Z9X3Q8KY52DPGDWG9KEX16"
	BuiltinPPIID              = "01M7Z9X3Q8KY52DPGDWG9KEX17"
	BuiltinFetchWriteID       = "01M7Z9X3Q8KY52DPGDWG9KEX38"
	BuiltinKNXID              = "01M7Z9X3Q8KY52DPGDWG9KEX18"
	BuiltinKNDID              = "01M7Z9X3Q8KY52DPGDWG9KEX19"
	BuiltinHeidenhainID       = "01M7Z9X3Q8KY52DPGDWG9KEX20"
	BuiltinFanucFocasID       = "01M7Z9X3Q8KY52DPGDWG9KEX21"
	BuiltinMTConnectID        = "01M7Z9X3Q8KY52DPGDWG9KEX22"
	BuiltinKukaID             = "01M7Z9X3Q8KY52DPGDWG9KEX23"
	BuiltinCodesysV3ID        = "01M7Z9X3Q8KY52DPGDWG9KEX24"
	BuiltinAB5000ID           = "01M7Z9X3Q8KY52DPGDWG9KEX25"
	BuiltinDF1ID              = "01M7Z9X3Q8KY52DPGDWG9KEX26"
	BuiltinHJ212ID            = "01M7Z9X3Q8KY52DPGDWG9KEX27"
	BuiltinBACnetID           = "01M7Z9X3Q8KY52DPGDWG9KEX28"
	BuiltinADSID              = "01M7Z9X3Q8KY52DPGDWG9KEX29"
	BuiltinSRTPID             = "01M7Z9X3Q8KY52DPGDWG9KEX30"
	BuiltinSNMPID             = "01M7Z9X3Q8KY52DPGDWG9KEX31"
	BuiltinIEC61850ID         = "01M7Z9X3Q8KY52DPGDWG9KEX37"
	BuiltinCOMLIID            = "01M7Z9X3Q8KY52DPGDWG9KEX32"
	BuiltinFXID               = "01M7Z9X3Q8KY52DPGDWG9KEX33"
	BuiltinA1EID              = "01M7Z9X3Q8KY52DPGDWG9KEX34"
	BuiltinQnA3EID            = "01M7Z9X3Q8KY52DPGDWG9KEX35"
	BuiltinQnA4EID            = "01M7Z9X3Q8KY52DPGDWG9KEX36"
)

const emptySchema = `{"fields":[]}`

const modbusTcpSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "IP:PORT",
      "labelEn": "IP:PORT",
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
      "options": [
        {
          "value": "CDAB",
          "labelZh": "CDAB",
          "labelEn": "CDAB"
        },
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
      "type": "select",
      "options": [
        {"value":"0","labelZh":"0","labelEn":"0"},
        {"value":"1","labelZh":"1","labelEn":"1"}
      ]
    },
    {
      "name": "concurrency",
      "labelZh": "并发数",
      "labelEn": "Concurrency",
      "regex": "^[1-3]$",
      "sort": 5,
      "type": "select",
      "options": [
        {"value": "1", "labelZh": "1", "labelEn": "1"},
        {"value": "2", "labelZh": "2", "labelEn": "2"},
        {"value": "3", "labelZh": "3", "labelEn": "3"}
      ]
    },
    {
      "name": "timeoutMs",
      "labelZh": "超时(ms)",
      "labelEn": "Timeout (ms)",
      "regex": "^[1-9][0-9]{0,5}$",
      "placeholderZh": "3000",
      "placeholderEn": "3000",
      "sort": 6,
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

const modbusRtuSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "串口设备",
      "labelEn": "Serial Port",
      "regex": "^[^\\s]+$",
      "placeholderZh": "ttyUSB0",
      "placeholderEn": "ttyUSB0",
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
      "options": [
        {"value":"CDAB","labelZh":"CDAB","labelEn":"CDAB"},
        {"value":"ABCD","labelZh":"ABCD","labelEn":"ABCD"},
        {"value":"BADC","labelZh":"BADC","labelEn":"BADC"},
        {"value":"DCBA","labelZh":"DCBA","labelEn":"DCBA"}
      ]
    },
    {
      "name": "positionAddress",
      "labelZh": "起始地址",
      "labelEn": "Position Address",
      "regex": "^(0|1)$",
      "sort": 7,
      "type": "select",
      "options": [
        {"value":"0","labelZh":"0","labelEn":"0"},
        {"value":"1","labelZh":"1","labelEn":"1"}
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

const modbusAsciiSchema = modbusRtuSchema

const modbusRtuOverTcpSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "IP:PORT",
      "labelEn": "IP:PORT",
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
      "options": [
        {"value":"CDAB","labelZh":"CDAB","labelEn":"CDAB"},
        {"value":"ABCD","labelZh":"ABCD","labelEn":"ABCD"},
        {"value":"BADC","labelZh":"BADC","labelEn":"BADC"},
        {"value":"DCBA","labelZh":"DCBA","labelEn":"DCBA"}
      ]
    },
    {
      "name": "positionAddress",
      "labelZh": "起始地址",
      "labelEn": "Position Address",
      "regex": "^(0|1)$",
      "sort": 4,
      "type": "select",
      "options": [
        {"value":"0","labelZh":"0","labelEn":"0"},
        {"value":"1","labelZh":"1","labelEn":"1"}
      ]
    },
    {
      "name": "concurrency",
      "labelZh": "并发数",
      "labelEn": "Concurrency",
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
      "name": "concurrency",
      "labelZh": "并发数",
      "labelEn": "Concurrency",
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
      "name": "concurrency",
      "labelZh": "并发数",
      "labelEn": "Concurrency",
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
      "name": "concurrency",
      "labelZh": "并发数",
      "labelEn": "Concurrency",
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

const finsSchema = `{
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
      "name": "concurrency",
      "labelZh": "并发数",
      "labelEn": "Concurrency",
      "regex": "^[1-3]$",
      "sort": 3,
      "type": "select",
      "options": [
        {"value":"1","labelZh":"1","labelEn":"1"},
        {"value":"2","labelZh":"2","labelEn":"2"},
        {"value":"3","labelZh":"3","labelEn":"3"}
      ]
    },
    {
      "name": "transport",
      "labelZh": "传输方式",
      "labelEn": "Transport",
      "regex": "^(tcp|udp)$",
      "sort": 4,
      "type": "select",
      "options": [
        {"value":"tcp","labelZh":"TCP","labelEn":"TCP"},
        {"value":"udp","labelZh":"UDP","labelEn":"UDP"}
      ]
    },
    {
      "name": "type",
      "labelZh": "PLC 型号",
      "labelEn": "PLC Type",
      "regex": "^[0-7]$",
      "sort": 5,
      "type": "select",
      "options": [
        {"value":"0","labelZh":"Other","labelEn":"Other"},
        {"value":"1","labelZh":"CP","labelEn":"CP"},
        {"value":"2","labelZh":"CJ","labelEn":"CJ"},
        {"value":"3","labelZh":"CS","labelEn":"CS"},
        {"value":"4","labelZh":"C","labelEn":"C"},
        {"value":"5","labelZh":"NJ","labelEn":"NJ"},
        {"value":"6","labelZh":"NX","labelEn":"NX"},
        {"value":"7","labelZh":"NY","labelEn":"NY"}
      ]
    },
    {
      "name": "expires",
      "labelZh": "超时(ms)",
      "labelEn": "Timeout (ms)",
      "regex": "^[1-9][0-9]{0,5}$",
      "sort": 6,
      "type": "input"
    },
    {
      "name": "maxSize",
      "labelZh": "单次最大字节数",
      "labelEn": "Max Size",
      "regex": "^(64|128|256|512|1024)$",
      "sort": 7,
      "type": "select",
      "options": [
        {"value":"64","labelZh":"64","labelEn":"64"},
        {"value":"128","labelZh":"128","labelEn":"128"},
        {"value":"256","labelZh":"256","labelEn":"256"},
        {"value":"512","labelZh":"512","labelEn":"512"},
        {"value":"1024","labelZh":"1024","labelEn":"1024"}
      ]
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "D100 / CIO0.0 / EM0W0",
    "placeholderEn": "D100 / CIO0.0 / EM0W0",
    "descriptionZh": "支持 CIO0、W0、H0、A0、D100、P0、F0、EM0W0、E1W0；位地址 D0.0；字序后缀 #B/#L",
    "descriptionEn": "Supports CIO0, W0, H0, A0, D100, P0, F0, EM0W0, E1W0; bit D0.0; byte order #B/#L"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^(CIO|W|H|A|D|F|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(\\.[0-9]{1,2})?$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^(CIO|W|H|A|D|F|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(\\.[0-9]{1,2})?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^(CIO|W|H|A|D|P|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(#[BL]{1,2})?$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^(CIO|W|H|A|D|P|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(#[BL]{1,2})?$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^(CIO|W|H|A|D|P|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(#[BL]{1,2})?$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^(CIO|W|H|A|D|P|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(#[BL]{1,2})?$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^(CIO|W|H|A|D|P|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(#[BL]{1,2})?$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^(CIO|W|H|A|D|P|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(#[BL]{1,2})?$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^(CIO|W|H|A|D|P|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(#[BL]{1,2})?$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^(CIO|W|H|A|D|P|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}(#[BL]{1,2})?$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^(CIO|W|H|A|D|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}\\.[0-9]{1,3}[HL]$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^(CIO|W|H|A|D|EM[0-9A-C]W|E[0-9]W)[0-9]{1,5}\\.[0-9]{1,3}[HL]$"}
  ]
}`

const hostlinkCmodeSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "串口设备",
      "labelEn": "Serial Port",
      "regex": "^[^\\s]+$",
      "placeholderZh": "ttyUSB0",
      "placeholderEn": "ttyUSB0",
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
      "name": "timeoutMs",
      "labelZh": "超时(ms)",
      "labelEn": "Timeout (ms)",
      "regex": "^[1-9][0-9]{0,5}$",
      "sort": 6,
      "type": "input"
    },
    {
      "name": "intervalMs",
      "labelZh": "命令间隔(ms)",
      "labelEn": "Interval (ms)",
      "regex": "^[0-9]{1,5}$",
      "sort": 7,
      "type": "input"
    },
    {
      "name": "maxReadWords",
      "labelZh": "单次最大字数",
      "labelEn": "Max Read Words",
      "regex": "^[1-9][0-9]{0,2}$",
      "sort": 8,
      "type": "input"
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "00!CIO0000 / 00!D0000",
    "placeholderEn": "00!CIO0000 / 00!D0000",
    "descriptionZh": "格式：站号!区域地址，例如 00!CIO0000、00!D0000；TS 区位地址；字符串使用 .lenH/.lenL",
    "descriptionEn": "Format: station!areaAddress, e.g. 00!CIO0000, 00!D0000; TS bits; strings use .lenH/.lenL"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]{1,2}!TS[0-9]{1,4}$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[0-9]{1,2}!TS[0-9]{1,4}$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}\\.[0-9]{1,2}[HL]$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^[0-9]{1,2}!(CIO|A|D|LR|HR|TD)[0-9]{1,4}\\.[0-9]{1,2}[HL]$"}
  ]
}`

const omronCipSchema = `{
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
      "name": "concurrency",
      "labelZh": "并发数",
      "labelEn": "Concurrency",
      "regex": "^[1-3]$",
      "sort": 3,
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
    "placeholderZh": "D100 / MyArray[3] / MyStruct.Member",
    "placeholderEn": "D100 / MyArray[3] / MyStruct.Member",
    "descriptionZh": "CIP 符号名或对象路径；数组使用 [n]；BIT/BOOL 可将位号写在最后一段",
    "descriptionEn": "CIP symbol or object path; arrays use [n]; BIT/BOOL may use a trailing bit number"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^.+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^.+$"}
  ]
}`

const iec104Schema = `{
  "fields": [
    {"name":"location","labelZh":"IP:PORT","labelEn":"IP:PORT","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9]):(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3})$","placeholderZh":"127.0.0.1:2404","placeholderEn":"127.0.0.1:2404","sort":1,"type":"input","required":true},
    {"name":"ca","labelZh":"公共地址","labelEn":"Common Address","regex":"^[0-9]{1,5}$","sort":2,"type":"input"},
    {"name":"interval","labelZh":"总招周期(s)","labelEn":"Interrogation Interval (s)","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"timeout","labelZh":"超时(s)","labelEn":"Timeout (s)","regex":"^[1-9][0-9]{0,3}$","sort":4,"type":"input"},
    {"name":"active","labelZh":"主动总召","labelEn":"Active Interrogation","regex":"^(true|false)$","sort":5,"type":"select","options":[{"value":"true","labelZh":"是","labelEn":"Yes"},{"value":"false","labelZh":"否","labelEn":"No"}]}
  ],
  "variableAddress": {"labelZh":"信息对象地址 IOA","labelEn":"Information Object Address","placeholderZh":"1","placeholderEn":"1","descriptionZh":"十进制 IOA","descriptionEn":"Decimal IOA"},
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[0-9]+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9]+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]+$"}
  ]
}`

const iec101Schema = `{
  "fields": [
    {"name":"link","labelZh":"链路类型","labelEn":"Link Type","regex":"^[01]$","sort":1,"type":"select","required":true,"options":[{"value":"0","labelZh":"串口","labelEn":"Serial"},{"value":"1","labelZh":"TCP","labelEn":"TCP"}]},
    {"name":"location","labelZh":"地址","labelEn":"Address","regex":"^[^\\s]+$","sort":2,"type":"input","required":true,"variants":[
      {"when":{"field":"link","equals":"1"},"labelZh":"IP:PORT","labelEn":"IP:PORT","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9]):(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3})$","placeholderZh":"127.0.0.1:2404","placeholderEn":"127.0.0.1:2404"},
      {"when":{"field":"link","equals":"0"},"labelZh":"设备地址","labelEn":"Device Address","regex":"^[^\\s]+$","placeholderZh":"ttyS0","placeholderEn":"ttyS0"}
    ]},
    {"name":"ca","labelZh":"公共地址","labelEn":"Common Address","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"caLength","labelZh":"CA 长度","labelEn":"CA Length","regex":"^[12]$","sort":4,"type":"input"},
    {"name":"cotLength","labelZh":"COT 长度","labelEn":"COT Length","regex":"^[12]$","sort":5,"type":"input"},
    {"name":"ioaLength","labelZh":"IOA 长度","labelEn":"IOA Length","regex":"^[123]$","sort":6,"type":"input"},
    {"name":"baud","labelZh":"波特率","labelEn":"Baud Rate","regex":"^[0-9]{4,6}$","sort":7,"type":"input"},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^[012]$","sort":8,"type":"input"},
    {"name":"data","labelZh":"数据位","labelEn":"Data Bits","regex":"^[5-8]$","sort":9,"type":"input"},
    {"name":"stop","labelZh":"停止位","labelEn":"Stop Bits","regex":"^[01]$","sort":10,"type":"input"},
    {"name":"timeout","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[1-9][0-9]{0,5}$","sort":11,"type":"input"}
  ],
  "variableAddress": {"labelZh":"信息对象地址 IOA","labelEn":"Information Object Address","placeholderZh":"100 / 2000","placeholderEn":"100 / 2000","descriptionZh":"十进制 IOA","descriptionEn":"Decimal IOA"},
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[0-9]+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9]+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]+$"}
  ]
}`

const iec102Schema = `{
  "fields": [
    {"name":"link","labelZh":"链路类型","labelEn":"Link Type","regex":"^[01]$","sort":1,"type":"select","required":true,"options":[{"value":"0","labelZh":"串口","labelEn":"Serial"},{"value":"1","labelZh":"TCP","labelEn":"TCP"}]},
    {"name":"location","labelZh":"地址","labelEn":"Address","regex":"^[^\\s]+$","sort":2,"type":"input","required":true,"variants":[
      {"when":{"field":"link","equals":"1"},"labelZh":"IP:PORT","labelEn":"IP:PORT","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9]):(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3})$","placeholderZh":"127.0.0.1:2404","placeholderEn":"127.0.0.1:2404"},
      {"when":{"field":"link","equals":"0"},"labelZh":"设备地址","labelEn":"Device Address","regex":"^[^\\s]+$","placeholderZh":"ttyS0","placeholderEn":"ttyS0"}
    ]},
    {"name":"la","labelZh":"链路地址","labelEn":"Link Address","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"ca","labelZh":"公共地址","labelEn":"Common Address","regex":"^[0-9]{1,5}$","sort":4,"type":"input"},
    {"name":"timeout","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[1-9][0-9]{0,5}$","sort":5,"type":"input"},
    {"name":"baud","labelZh":"波特率","labelEn":"Baud Rate","regex":"^[0-9]{4,6}$","sort":6,"type":"input"},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^[012]$","sort":7,"type":"input"},
    {"name":"data","labelZh":"数据位","labelEn":"Data Bits","regex":"^[5-8]$","sort":8,"type":"input"},
    {"name":"stop","labelZh":"停止位","labelEn":"Stop Bits","regex":"^[01]$","sort":9,"type":"input"}
  ],
  "variableAddress": {"labelZh":"记录地址.IOA","labelEn":"RA.IOA","placeholderZh":"1.0","placeholderEn":"1.0","descriptionZh":"格式 RA.IOA，范围 0..255","descriptionEn":"RA.IOA, 0..255"},
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]+\\.[0-9]+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]+\\.[0-9]+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]+\\.[0-9]+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]+\\.[0-9]+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9]+\\.[0-9]+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]+\\.[0-9]+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]+\\.[0-9]+$"}
  ]
}`

const iec103Schema = `{
  "fields": [
    {"name":"link","labelZh":"链路类型","labelEn":"Link Type","regex":"^[01]$","sort":1,"type":"select","required":true,"options":[{"value":"0","labelZh":"串口","labelEn":"Serial"},{"value":"1","labelZh":"TCP","labelEn":"TCP"}]},
    {"name":"location","labelZh":"地址","labelEn":"Address","regex":"^[^\\s]+$","sort":2,"type":"input","required":true,"variants":[
      {"when":{"field":"link","equals":"1"},"labelZh":"IP:PORT","labelEn":"IP:PORT","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9]):(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3})$","placeholderZh":"127.0.0.1:2404","placeholderEn":"127.0.0.1:2404"},
      {"when":{"field":"link","equals":"0"},"labelZh":"设备地址","labelEn":"Device Address","regex":"^[^\\s]+$","placeholderZh":"ttyS0","placeholderEn":"ttyS0"}
    ]},
    {"name":"timeout","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[1-9][0-9]{0,5}$","sort":3,"type":"input"},
    {"name":"baud","labelZh":"波特率","labelEn":"Baud Rate","regex":"^[0-9]{4,6}$","sort":4,"type":"input"},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^[012]$","sort":5,"type":"input"},
    {"name":"data","labelZh":"数据位","labelEn":"Data Bits","regex":"^[5-8]$","sort":6,"type":"input"},
    {"name":"stop","labelZh":"停止位","labelEn":"Stop Bits","regex":"^[01]$","sort":7,"type":"input"}
  ],
  "variableAddress": {"labelZh":"CA.FUN.NUM","labelEn":"CA.FUN.NUM","placeholderZh":"1.16.1","placeholderEn":"1.16.1","descriptionZh":"CA、FUN、NUM 范围 0..255","descriptionEn":"CA, FUN, NUM 0..255"},
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]+\\.[0-9]+\\.[0-9]+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]+\\.[0-9]+\\.[0-9]+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]+\\.[0-9]+\\.[0-9]+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]+\\.[0-9]+\\.[0-9]+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9]+\\.[0-9]+\\.[0-9]+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]+\\.[0-9]+\\.[0-9]+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]+\\.[0-9]+\\.[0-9]+$"}
  ]
}`

const ethernetIPSchema = `{
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
      "name": "slot",
      "labelZh": "槽号",
      "labelEn": "Slot",
      "regex": "^[0-9]{1,3}$",
      "sort": 3,
      "type": "input"
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "Tag / Program:MainProgram.Tag",
    "placeholderEn": "Tag / Program:MainProgram.Tag",
    "descriptionZh": "CIP 符号标签，点号分隔层级，例如 Program:MainProgram.Counter；不支持数组下标或末尾位号",
    "descriptionEn": "CIP symbolic tag with dot-separated members, for example Program:MainProgram.Counter; array indexes and trailing bit numbers are not supported"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^.+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"}
  ]
}`

const etherCATSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "TwinCAT 地址",
      "labelEn": "TwinCAT Address",
      "regex": "^[A-Za-z0-9.-]+$",
      "sort": 1,
      "type": "input",
      "required": true
    },
    {
      "name": "port",
      "labelZh": "TCP 端口",
      "labelEn": "TCP Port",
      "regex": "^[0-9]{1,5}$",
      "sort": 2,
      "type": "input"
    },
    {
      "name": "amsNetId",
      "labelZh": "目标 AMS Net ID",
      "labelEn": "Target AMS Net ID",
      "regex": "^([0-9]{1,3}\\.){5}[0-9]{1,3}$",
      "sort": 3,
      "type": "input"
    },
    {
      "name": "amsPort",
      "labelZh": "目标 ADS 端口",
      "labelEn": "Target ADS Port",
      "regex": "^8[0-9]{2}$",
      "sort": 4,
      "type": "input"
    },
    {
      "name": "localAmsNetId",
      "labelZh": "本地 AMS Net ID",
      "labelEn": "Local AMS Net ID",
      "regex": "^([0-9]{1,3}\\.){5}[0-9]{1,3}$",
      "sort": 5,
      "type": "input"
    },
    {
      "name": "localAmsPort",
      "labelZh": "本地 ADS 端口",
      "labelEn": "Local ADS Port",
      "regex": "^8[0-9]{2}$",
      "sort": 6,
      "type": "input"
    },
    {
      "name": "timeoutMs",
      "labelZh": "超时(ms)",
      "labelEn": "Timeout (ms)",
      "regex": "^[1-9][0-9]{0,5}$",
      "sort": 7,
      "type": "input"
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "0xF020,0 / 0x4020,0.80",
    "placeholderEn": "0xF020,0 / 0x4020,0.80",
    "descriptionZh": "ADS 索引组,索引偏移，支持十进制或 0x 前缀；STRING 必须追加 .长度，例如 0x4020,0.80",
    "descriptionEn": "ADS index group,index offset in decimal or 0x form; STRING requires a .length suffix, for example 0x4020,0.80"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^.+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"}
  ]
}`

const ethernetIP5500Schema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "PLC IP 地址",
      "labelEn": "PLC IP Address",
      "regex": "^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$",
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
      "name": "slot",
      "labelZh": "CPU 槽号",
      "labelEn": "CPU Slot",
      "regex": "^[0-9]{1,3}$",
      "sort": 3,
      "type": "input"
    },
    {
      "name": "interval",
      "labelZh": "指令间隔(ms)",
      "labelEn": "Request Interval (ms)",
      "regex": "^[0-9]{1,5}$",
      "sort": 4,
      "type": "input"
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "Tag / Program:MainProgram.Tag / Tag[3]",
    "placeholderEn": "Tag / Program:MainProgram.Tag / Tag[3]",
    "descriptionZh": "CIP 标签名；支持点号成员、数组 [n]、程序域 Program:MainProgram.Tag；BIT/BOOL 按 BOOL 处理，整型位请使用位成员或设备侧映射",
    "descriptionEn": "CIP tag name; supports dot members, arrays [n], Program:MainProgram.Tag; BIT/BOOL is treated as BOOL, use a bit member or device mapping for integer bits"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^.+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^.+$"},
    {"value":"WORD","labelZh":"word","labelEn":"word","addressRegex":"^.+$"},
    {"value":"DWORD","labelZh":"dword","labelEn":"dword","addressRegex":"^.+$"},
    {"value":"LWORD","labelZh":"lword","labelEn":"lword","addressRegex":"^.+$"}
  ]
}`

const ethernetIP1400Schema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "PLC IP 地址",
      "labelEn": "PLC IP Address",
      "regex": "^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$",
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
      "name": "cipSize",
      "labelZh": "CIP 大小",
      "labelEn": "CIP Size",
      "regex": "^[0-9]{1,5}$",
      "sort": 3,
      "type": "input"
    },
    {
      "name": "logixArraySize",
      "labelZh": "Logix 数组大小",
      "labelEn": "Logix Array Size",
      "regex": "^[0-9]{1,5}$",
      "sort": 4,
      "type": "input"
    }
  ],
  "variableAddress": {
    "labelZh": "PCCC 地址",
    "labelEn": "PCCC Address",
    "placeholderZh": "N7:0 / F8:0 / B3:0/0 / T4:0",
    "placeholderEn": "N7:0 / F8:0 / B3:0/0 / T4:0",
    "descriptionZh": "PCCC 文件地址：N/F/B/T/C/I/O/S；位地址使用 /位号；T/C 支持 .PRE/.ACC 读写，ACC 写入使用 read-modify-write",
    "descriptionEn": "PCCC file address: N/F/B/T/C/I/O/S; use /bit for bit files; T/C supports .PRE/.ACC reads and writes, ACC writes use read-modify-write"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"}
  ]
}`

const dnp3Schema = `{
  "fields": [
    {"name":"location","labelZh":"从站 IP","labelEn":"Outstation IP","regex":"^[A-Za-z0-9.-]+$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"端口","labelEn":"Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input"},
    {"name":"masterId","labelZh":"主站地址","labelEn":"Master ID","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"slaveId","labelZh":"从站地址","labelEn":"Slave ID","regex":"^[0-9]{1,5}$","sort":4,"type":"input"},
    {"name":"class0123Interval","labelZh":"Class0123 周期(ms)","labelEn":"Class0123 Interval (ms)","regex":"^[0-9]{1,8}$","sort":5,"type":"input"},
    {"name":"class123Interval","labelZh":"Class123 周期(ms)","labelEn":"Class123 Interval (ms)","regex":"^[0-9]{1,8}$","sort":6,"type":"input"},
    {"name":"timeSync","labelZh":"时间同步","labelEn":"Time Sync","regex":"^[01]$","sort":7,"type":"select","options":[{"value":"0","labelZh":"否","labelEn":"No"},{"value":"1","labelZh":"是","labelEn":"Yes"}]}
  ],
  "variableAddress": {
    "labelZh": "DNP3 地址",
    "labelEn": "DNP3 Address",
    "placeholderZh": "30.1.5.value / CROB.0:1:1000:0",
    "placeholderEn": "30.1.5.value / CROB.0:1:1000:0",
    "descriptionZh": "格式 group.variation.index[.value]；CROB 控制点使用 CROB.index:counter:on-time:off-time",
    "descriptionEn": "Format group.variation.index[.value]; CROB points use CROB.index:counter:on-time:off-time"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"}
  ]
}`

const dlt645Schema = `{
  "fields": [
    {"name":"link","labelZh":"连接方式","labelEn":"Link","regex":"^[01]$","sort":1,"type":"select","required":true,"options":[{"value":"0","labelZh":"串口","labelEn":"Serial"},{"value":"1","labelZh":"TCP","labelEn":"TCP"}]},
    {"name":"location","labelZh":"串口设备 / IP","labelEn":"Serial Port / IP","regex":"^[^\\s]+$","sort":2,"type":"input","required":true,"variants":[{"when":{"field":"link","equals":"1"},"labelZh":"IP 地址","labelEn":"IP Address","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$","placeholderZh":"192.168.1.10","placeholderEn":"192.168.1.10"},{"when":{"field":"link","equals":"0"},"labelZh":"串口设备","labelEn":"Serial Port","regex":"^[^\\s]+$","placeholderZh":"ttyUSB0","placeholderEn":"ttyUSB0"}]},
    {"name":"port","labelZh":"TCP 端口","labelEn":"TCP Port","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"baudRate","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(1200|2400|4800|9600|19200|38400|57600|115200)$","sort":4,"type":"select","options":[{"value":"1200","labelZh":"1200","labelEn":"1200"},{"value":"2400","labelZh":"2400","labelEn":"2400"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"},{"value":"57600","labelZh":"57600","labelEn":"57600"},{"value":"115200","labelZh":"115200","labelEn":"115200"}]},
    {"name":"dataBits","labelZh":"数据位","labelEn":"Data Bits","regex":"^(5|6|7|8)$","sort":5,"type":"select","options":[{"value":"5","labelZh":"5","labelEn":"5"},{"value":"6","labelZh":"6","labelEn":"6"},{"value":"7","labelZh":"7","labelEn":"7"},{"value":"8","labelZh":"8","labelEn":"8"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^(noParity|oddParity|evenParity|none|odd|even|0|1|2)$","sort":6,"type":"select","options":[{"value":"evenParity","labelZh":"偶校验","labelEn":"Even"},{"value":"noParity","labelZh":"无校验","labelEn":"None"},{"value":"oddParity","labelZh":"奇校验","labelEn":"Odd"}]},
    {"name":"stopBits","labelZh":"停止位","labelEn":"Stop Bits","regex":"^(1|1\\.5|2)$","sort":7,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"1.5","labelZh":"1.5","labelEn":"1.5"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":8,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":9,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "DL/T 645 地址",
    "labelEn": "DL/T 645 Address",
    "placeholderZh": "000000000001#00-00-00-00",
    "placeholderEn": "000000000001#00-00-00-00",
    "descriptionZh": "格式：表号#DI。2007 使用 DI3-DI2-DI1-DI0，1997 使用 DI1-DI0",
    "descriptionEn": "Format: meter#DI. 2007 uses DI3-DI2-DI1-DI0; 1997 uses DI1-DI0"
  },
  "variableDataTypes": [
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]{1,12}#[0-9A-Fa-f]{2}(-[0-9A-Fa-f]{2}){0,3}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]{1,12}#[0-9A-Fa-f]{2}(-[0-9A-Fa-f]{2}){0,3}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]{1,12}#[0-9A-Fa-f]{2}(-[0-9A-Fa-f]{2}){0,3}$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9]{1,12}#[0-9A-Fa-f]{2}(-[0-9A-Fa-f]{2}){0,3}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[0-9]{1,12}#[0-9A-Fa-f]{2}(-[0-9A-Fa-f]{2}){0,3}$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[0-9]{1,12}#[0-9A-Fa-f]{2}(-[0-9A-Fa-f]{2}){0,3}$"}
  ]
}`

const cjt188Schema = `{
  "fields": [
    {"name":"link","labelZh":"连接方式","labelEn":"Link","regex":"^[01]$","sort":1,"type":"select","required":true,"options":[{"value":"0","labelZh":"串口","labelEn":"Serial"},{"value":"1","labelZh":"TCP","labelEn":"TCP"}]},
    {"name":"location","labelZh":"串口设备 / IP","labelEn":"Serial Port / IP","regex":"^[^\\s]+$","sort":2,"type":"input","required":true,"variants":[{"when":{"field":"link","equals":"1"},"labelZh":"IP 地址","labelEn":"IP Address","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$","placeholderZh":"192.168.1.10","placeholderEn":"192.168.1.10"},{"when":{"field":"link","equals":"0"},"labelZh":"串口设备","labelEn":"Serial Port","regex":"^[^\\s]+$","placeholderZh":"ttyUSB0","placeholderEn":"ttyUSB0"}]},
    {"name":"port","labelZh":"TCP 端口","labelEn":"TCP Port","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"baudRate","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(1200|2400|4800|9600|19200|38400|57600|115200)$","sort":4,"type":"select","options":[{"value":"1200","labelZh":"1200","labelEn":"1200"},{"value":"2400","labelZh":"2400","labelEn":"2400"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"},{"value":"57600","labelZh":"57600","labelEn":"57600"},{"value":"115200","labelZh":"115200","labelEn":"115200"}]},
    {"name":"dataBits","labelZh":"数据位","labelEn":"Data Bits","regex":"^(5|6|7|8)$","sort":5,"type":"select","options":[{"value":"5","labelZh":"5","labelEn":"5"},{"value":"6","labelZh":"6","labelEn":"6"},{"value":"7","labelZh":"7","labelEn":"7"},{"value":"8","labelZh":"8","labelEn":"8"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^(noParity|oddParity|evenParity|none|odd|even|0|1|2)$","sort":6,"type":"select","options":[{"value":"evenParity","labelZh":"偶校验","labelEn":"Even"},{"value":"noParity","labelZh":"无校验","labelEn":"None"},{"value":"oddParity","labelZh":"奇校验","labelEn":"Odd"}]},
    {"name":"stopBits","labelZh":"停止位","labelEn":"Stop Bits","regex":"^(1|1\\.5|2)$","sort":7,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"1.5","labelZh":"1.5","labelEn":"1.5"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":8,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":9,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "CJ/T 188 地址",
    "labelEn": "CJ/T 188 Address",
    "placeholderZh": "10#0000000001#901F",
    "placeholderEn": "10#0000000001#901F",
    "descriptionZh": "格式：仪表类型#表号#DI，仪表类型与 DI 使用十六进制；表号 14 位十进制",
    "descriptionEn": "Format: meter type#meter number#DI; type and DI are hexadecimal; meter number has 14 decimal digits"
  },
  "variableDataTypes": [
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9A-Fa-f]{1,2}#[0-9]{1,14}#[0-9A-Fa-f]{4}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9A-Fa-f]{1,2}#[0-9]{1,14}#[0-9A-Fa-f]{4}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9A-Fa-f]{1,2}#[0-9]{1,14}#[0-9A-Fa-f]{4}$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9A-Fa-f]{1,2}#[0-9]{1,14}#[0-9A-Fa-f]{4}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[0-9A-Fa-f]{1,2}#[0-9]{1,14}#[0-9A-Fa-f]{4}$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[0-9A-Fa-f]{1,2}#[0-9]{1,14}#[0-9A-Fa-f]{4}$"}
  ]
}`

const dlt698Schema = `{
  "fields": [
    {"name":"link","labelZh":"连接方式","labelEn":"Link","regex":"^[01]$","sort":1,"type":"select","required":true,"options":[{"value":"0","labelZh":"串口","labelEn":"Serial"},{"value":"1","labelZh":"TCP","labelEn":"TCP"}]},
    {"name":"location","labelZh":"串口设备 / IP","labelEn":"Serial Port / IP","regex":"^[^\\s]+$","sort":2,"type":"input","required":true,"variants":[{"when":{"field":"link","equals":"1"},"labelZh":"IP 地址","labelEn":"IP Address","regex":"^[A-Za-z0-9.-]+$","placeholderZh":"192.168.1.10","placeholderEn":"192.168.1.10"},{"when":{"field":"link","equals":"0"},"labelZh":"串口设备","labelEn":"Serial Port","regex":"^[^\\s]+$","placeholderZh":"ttyUSB0","placeholderEn":"ttyUSB0"}]},
    {"name":"port","labelZh":"TCP 端口","labelEn":"TCP Port","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"meterAddress","labelZh":"表地址","labelEn":"Meter Address","regex":"^[0-9A-Fa-f]{2,16}$","sort":4,"type":"input","required":true},
    {"name":"clientAddress","labelZh":"客户机地址","labelEn":"Client Address","regex":"^[0-9]{1,3}$","sort":5,"type":"input"},
    {"name":"baudRate","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(1200|2400|4800|9600|19200|38400|57600|115200)$","sort":6,"type":"select","options":[{"value":"1200","labelZh":"1200","labelEn":"1200"},{"value":"2400","labelZh":"2400","labelEn":"2400"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"},{"value":"57600","labelZh":"57600","labelEn":"57600"},{"value":"115200","labelZh":"115200","labelEn":"115200"}]},
    {"name":"dataBits","labelZh":"数据位","labelEn":"Data Bits","regex":"^(5|6|7|8)$","sort":7,"type":"select","options":[{"value":"5","labelZh":"5","labelEn":"5"},{"value":"6","labelZh":"6","labelEn":"6"},{"value":"7","labelZh":"7","labelEn":"7"},{"value":"8","labelZh":"8","labelEn":"8"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^(noParity|oddParity|evenParity|none|odd|even|0|1|2)$","sort":8,"type":"select","options":[{"value":"evenParity","labelZh":"偶校验","labelEn":"Even"},{"value":"noParity","labelZh":"无校验","labelEn":"None"},{"value":"oddParity","labelZh":"奇校验","labelEn":"Odd"}]},
    {"name":"stopBits","labelZh":"停止位","labelEn":"Stop Bits","regex":"^(1|1\\.5|2)$","sort":9,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"1.5","labelZh":"1.5","labelEn":"1.5"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":10,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":11,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "OAD 地址",
    "labelEn": "OAD Address",
    "placeholderZh": "00000200",
    "placeholderEn": "00000200",
    "descriptionZh": "4 字节 OAD 十六进制，例如 00000200",
    "descriptionEn": "4-byte OAD in hexadecimal, for example 00000200"
  },
  "variableDataTypes": [
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9A-Fa-f]{8}$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9A-Fa-f]{8}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[0-9A-Fa-f]{8}$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[0-9A-Fa-f]{8}$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9A-Fa-f]{8}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9A-Fa-f]{8}$"}
  ]
}`

const mpiSchema = `{
  "fields": [
    {"name":"location","labelZh":"串口设备","labelEn":"Serial Port","regex":"^[^\\s]+$","placeholderZh":"ttyUSB0","placeholderEn":"ttyUSB0","sort":1,"type":"input","required":true},
    {"name":"baudRate","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(1200|2400|4800|9600|19200|38400|57600|115200)$","sort":2,"type":"select","options":[{"value":"1200","labelZh":"1200","labelEn":"1200"},{"value":"2400","labelZh":"2400","labelEn":"2400"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"},{"value":"57600","labelZh":"57600","labelEn":"57600"},{"value":"115200","labelZh":"115200","labelEn":"115200"}]},
    {"name":"dataBits","labelZh":"数据位","labelEn":"Data Bits","regex":"^(5|6|7|8)$","sort":3,"type":"select","options":[{"value":"5","labelZh":"5","labelEn":"5"},{"value":"6","labelZh":"6","labelEn":"6"},{"value":"7","labelZh":"7","labelEn":"7"},{"value":"8","labelZh":"8","labelEn":"8"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^(noParity|oddParity|evenParity)$","sort":4,"type":"select","options":[{"value":"oddParity","labelZh":"奇校验","labelEn":"Odd"},{"value":"noParity","labelZh":"无校验","labelEn":"None"},{"value":"evenParity","labelZh":"偶校验","labelEn":"Even"}]},
    {"name":"stopBits","labelZh":"停止位","labelEn":"Stop Bits","regex":"^(1|1\\.5|2)$","sort":5,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"1.5","labelZh":"1.5","labelEn":"1.5"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"station","labelZh":"MPI 站号","labelEn":"MPI Station","regex":"^[0-9]{1,3}$","sort":6,"type":"input","required":true},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":7,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":8,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "MPI 地址",
    "labelEn": "MPI Address",
    "placeholderZh": "DB1.DBB0 / M0.0 / I0",
    "placeholderEn": "DB1.DBB0 / M0.0 / I0",
    "descriptionZh": "支持 DB、I、Q、M、S、SM、AI、AQ、C、T、IECC、IECT、IECHC、P、DI 区域；位地址加 .0-.7，字符串/字节数组加长度后缀",
    "descriptionEn": "Supports DB, I, Q, M, S, SM, AI, AQ, C, T, IECC, IECT, IECHC, P and DI areas; add .0-.7 for bits and a length suffix for string/byte_array"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^((DB[0-9]{1,5}\\.DBX[0-9]{1,5}\\.[0-7])|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}\\.[0-7]))$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^((DB[0-9]{1,5}\\.DBX[0-9]{1,5}\\.[0-7])|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}\\.[0-7]))$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}))$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}))$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}))$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}))$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}))$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}))$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}))$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}))$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^(DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5}\\.[0-9]{1,5}|(SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}\\.[0-9]{1,5})$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^(DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5}\\.[0-9]{1,5}|(SM|S|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI)[0-9]{1,5}\\.[0-9]{1,5})$"}
  ]
}`

const ppiSchema = `{
  "fields": [
    {"name":"link","labelZh":"连接方式","labelEn":"Link","regex":"^[01]$","sort":1,"type":"select","required":true,"options":[{"value":"0","labelZh":"串口","labelEn":"Serial"},{"value":"1","labelZh":"TCP","labelEn":"TCP"}]},
    {"name":"location","labelZh":"串口设备 / IP","labelEn":"Serial Port / IP","regex":"^[^\\s]+$","sort":2,"type":"input","required":true,"variants":[{"when":{"field":"link","equals":"1"},"labelZh":"IP 地址","labelEn":"IP Address","regex":"^[A-Za-z0-9.-]+$","placeholderZh":"192.168.1.10","placeholderEn":"192.168.1.10"},{"when":{"field":"link","equals":"0"},"labelZh":"串口设备","labelEn":"Serial Port","regex":"^[^\\s]+$","placeholderZh":"ttyUSB0","placeholderEn":"ttyUSB0"}]},
    {"name":"port","labelZh":"TCP 端口","labelEn":"TCP Port","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"baudRate","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(1200|2400|4800|9600|19200|38400|57600|115200)$","sort":4,"type":"select","options":[{"value":"1200","labelZh":"1200","labelEn":"1200"},{"value":"2400","labelZh":"2400","labelEn":"2400"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"},{"value":"57600","labelZh":"57600","labelEn":"57600"},{"value":"115200","labelZh":"115200","labelEn":"115200"}]},
    {"name":"dataBits","labelZh":"数据位","labelEn":"Data Bits","regex":"^(5|6|7|8)$","sort":5,"type":"select","options":[{"value":"5","labelZh":"5","labelEn":"5"},{"value":"6","labelZh":"6","labelEn":"6"},{"value":"7","labelZh":"7","labelEn":"7"},{"value":"8","labelZh":"8","labelEn":"8"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^(noParity|oddParity|evenParity)$","sort":6,"type":"select","options":[{"value":"evenParity","labelZh":"偶校验","labelEn":"Even"},{"value":"noParity","labelZh":"无校验","labelEn":"None"},{"value":"oddParity","labelZh":"奇校验","labelEn":"Odd"}]},
    {"name":"stopBits","labelZh":"停止位","labelEn":"Stop Bits","regex":"^(1|1\\.5|2)$","sort":7,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"1.5","labelZh":"1.5","labelEn":"1.5"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"station","labelZh":"PPI 站号","labelEn":"PPI Station","regex":"^[0-9]{1,3}$","sort":8,"type":"input","required":true},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":9,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":10,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "PPI 地址",
    "labelEn": "PPI Address",
    "placeholderZh": "DB1.DBB0 / VW0 / M0.0 / I0.0",
    "placeholderEn": "DB1.DBB0 / VW0 / M0.0 / I0.0",
    "descriptionZh": "支持 DB、V、I、Q、M、S、SM、AI、AQ、AIW、AQW、C、T、IECC、IECT、IECHC、P、DI；位地址加 .0-.7，字符串/字节数组加长度后缀",
    "descriptionEn": "Supports DB, V, I, Q, M, S, SM, AI, AQ, AIW, AQW, C, T, IECC, IECT, IECHC, P and DI; add .0-.7 for bits and a length suffix for string/byte_array"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^((DB[0-9]{1,5}\\.DBX[0-9]{1,5}\\.[0-7])|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}\\.[0-7]))$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^((DB[0-9]{1,5}\\.DBX[0-9]{1,5}\\.[0-7])|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}\\.[0-7]))$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}))$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}))$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}))$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}))$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}))$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}))$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}))$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^((DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5})|((SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}))$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^(DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5}\\.[0-9]{1,5}|(SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}\\.[0-9]{1,5})$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^(DB[0-9]{1,5}\\.(DBB|DBW|DBD)[0-9]{1,5}\\.[0-9]{1,5}|(SM|S|AIW|AQW|AI|AQ|C|T|IECC|IECT|IECHC|P|I|Q|M|DI|V|VB|VW|VD)[0-9]{1,5}\\.[0-9]{1,5})$"}
  ]
}`

const knxSchema = `{
  "fields": [
    {"name":"discoveryIp","labelZh":"发现地址","labelEn":"Discovery IP","regex":"^[A-Za-z0-9.:-]+$","sort":1,"type":"input","required":true},
    {"name":"discoveryPort","labelZh":"发现端口","labelEn":"Discovery Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input","required":true},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":3,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":4,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "KNX 地址",
    "labelEn": "KNX Address",
    "placeholderZh": "1/2/3,1.2.3 或 1/2/3,1.2.3,2",
    "placeholderEn": "1/2/3,1.2.3 or 1/2/3,1.2.3,2",
    "descriptionZh": "格式：组地址,个体地址[,nbit]；组地址 main/middle/sub，个体地址 area.line.device；nbit 仅用于 UINT8 的 2/4 位数据",
    "descriptionEn": "Format: group address,individual address[,nbit]; group is main/middle/sub and individual is area.line.device; nbit is only for 2/4-bit UINT8 data"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}(,[24])?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^[0-9]{1,2}/[0-9]{1,2}/[0-9]{1,3},[0-9]{1,2}\\.[0-9]{1,2}\\.[0-9]{1,3}$"}
  ]
}`

const kndSchema = `{
  "fields": [
    {"name":"location","labelZh":"HTTP 地址","labelEn":"HTTP Address","placeholderZh":"http://192.168.1.10:8000","placeholderEn":"http://192.168.1.10:8000","regex":"^(http://)?[A-Za-z0-9._-]+(:[0-9]{1,5})?$","sort":1,"type":"input","required":true},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":2,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":3,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "KND 地址",
    "labelEn": "KND Address",
    "placeholderZh": "systemStatus.run-status / R100 / vars.0",
    "placeholderEn": "systemStatus.run-status / R100 / vars.0",
    "descriptionZh": "支持 PLC 区 X/Y/F/G/R/S/K/D、T/C.conf/cur、TL，以及 systemInfo/systemStatus/alarms/坐标/进给/倍率/变量等语义地址",
    "descriptionEn": "Supports PLC X/Y/F/G/R/S/K/D, T/C.conf/cur, TL, and semantic systemInfo/systemStatus/alarms/coordinates/feed/override/vars addresses"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"DOUBLE","labelZh":"double","labelEn":"double","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^[A-Za-z0-9_./,:?&=-]+$"}
  ]
}`

const heidenhainSchema = `{
  "fields": [
    {"name":"host","labelZh":"CNC IP","labelEn":"CNC IP","regex":"^[A-Za-z0-9.:-]+$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"LSV2 端口","labelEn":"LSV2 Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input","required":true},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":3,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":4,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "LSV2 地址",
    "labelEn": "LSV2 Address",
    "placeholderZh": "W100 / B0 / S0.10 / machinePosition.X",
    "placeholderEn": "W100 / B0 / S0.10 / machinePosition.X",
    "descriptionZh": "支持 PLC B/W/D/N/U/M/I/O/T/C/S 地址，以及 runState、controlMode、programMain、programCurrent、programLineNo、倍率、刀具和 machinePosition 等语义地址",
    "descriptionEn": "Supports PLC B/W/D/N/U/M/I/O/T/C/S addresses and semantic runState, controlMode, programs, overrides, spindle tool and machinePosition"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"DOUBLE","labelZh":"double","labelEn":"double","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^[A-Za-z0-9_./:-]+$"}
  ]
}`

const focasSchema = `{
  "fields": [
    {"name":"host","labelZh":"CNC IP","labelEn":"CNC IP","regex":"^[A-Za-z0-9.:-]+$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"FOCAS 端口","labelEn":"FOCAS Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input","required":true},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":3,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":4,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "FOCAS 地址",
    "labelEn": "FOCAS Address",
    "placeholderZh": "R100 / absolute.X / runState",
    "placeholderEn": "R100 / absolute.X / runState",
    "descriptionZh": "PMC 地址：A/C/D/E/F/G/K/M/N/R/T/X/Y；CNC 语义地址：absolute、machine、relative、runState、controlMode、program*、刀具和参数等",
    "descriptionEn": "PMC addresses: A/C/D/E/F/G/K/M/N/R/T/X/Y; CNC semantic addresses include absolute, machine, relative, runState, controlMode, programs, tools and parameters"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"DOUBLE","labelZh":"double","labelEn":"double","addressRegex":"^[A-Za-z0-9_./:-]+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^[A-Za-z0-9_./:-]+$"}
  ]
}`

const mtconnectSchema = `{
  "fields": [
    {"name":"host","labelZh":"设备 IP","labelEn":"Device IP","regex":"^[A-Za-z0-9.:-]+$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"HTTP 端口","labelEn":"HTTP Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input","required":true},
    {"name":"nsPrefix","labelZh":"命名空间前缀","labelEn":"Namespace Prefix","regex":"^[A-Za-z_][A-Za-z0-9_-]*$","sort":3,"type":"input"},
    {"name":"nsUri","labelZh":"命名空间 URI","labelEn":"Namespace URI","regex":"^[A-Za-z0-9._:/-]+$","sort":4,"type":"input"},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":5,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":6,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "XPath 地址",
    "labelEn": "XPath Address",
    "placeholderZh": "//DeviceStream/ComponentStream/Samples/Position[@dataItemId=\"Xpos\"]",
    "placeholderEn": "//DeviceStream/ComponentStream/Samples/Position[@dataItemId=\"Xpos\"]",
    "descriptionZh": "MTConnect XML 的 XPath 表达式，常用格式为 //.../Element[@dataItemId=\"id\"]",
    "descriptionEn": "XPath expression into MTConnect XML, commonly //.../Element[@dataItemId=\"id\"]"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^.+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^.+$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^.+$"},
    {"value":"DOUBLE","labelZh":"double","labelEn":"double","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"}
  ]
}`

const kukaSchema = `{
  "fields": [
    {"name":"host","labelZh":"地址","labelEn":"Host","regex":"^[A-Za-z0-9.:-]+$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"TCP 端口","labelEn":"TCP Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input","required":true},
    {"name":"connectionMode","labelZh":"连接模式","labelEn":"Connection Mode","regex":"^[01]$","sort":3,"type":"select","options":[{"value":"1","labelZh":"服务端","labelEn":"Server"},{"value":"0","labelZh":"客户端","labelEn":"Client"}]},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":4,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":5,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "XPath 地址",
    "labelEn": "XPath Address",
    "placeholderZh": "//Robot/Axis / //EKI/Status",
    "placeholderEn": "//Robot/Axis / //EKI/Status",
    "descriptionZh": "EKI 响应 XML 的 XPath，按机器人 KRL 项目定义的 XML 结构填写",
    "descriptionEn": "XPath into the EKI response XML, following the XML structure defined by the robot KRL project"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^.+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^.+$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^.+$"},
    {"value":"DOUBLE","labelZh":"double","labelEn":"double","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"}
  ]
}`

const codesysV3Schema = `{
  "fields": [
    {"name":"host","labelZh":"PLC IP","labelEn":"PLC IP","regex":"^[A-Za-z0-9.:-]+$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"CMP 端口","labelEn":"CMP Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input","required":true},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":3,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":4,"type":"input"},
    {"name":"username","labelZh":"用户名","labelEn":"Username","regex":"^[^\n\r]{0,64}$","sort":5,"type":"input"},
    {"name":"password","labelZh":"密码","labelEn":"Password","regex":"^[^\n\r]{0,64}$","sort":6,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "变量路径",
    "labelEn": "Variable Path",
    "placeholderZh": "PLC_PRG.MyVar / GVL.Array[0]",
    "placeholderEn": "PLC_PRG.MyVar / GVL.Array[0]",
    "descriptionZh": "CODESYS 全局变量或 PLC_PRG 变量路径，需与 PLC 符号完全一致",
    "descriptionEn": "CODESYS global or PLC_PRG variable path; must match the PLC symbol exactly"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"}
  ]
}`

const ab5000Schema = `{
  "fields": [
    {"name":"location","labelZh":"PLC IP","labelEn":"PLC IP","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"端口","labelEn":"Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input","required":true},
    {"name":"slot","labelZh":"Slot","labelEn":"Slot","regex":"^[0-9]{1,3}$","sort":3,"type":"input"},
    {"name":"compatibilityMode","labelZh":"兼容模式","labelEn":"Compatibility Mode","regex":"^(true|false)$","sort":4,"type":"select","options":[{"value":"false","labelZh":"否","labelEn":"No"},{"value":"true","labelZh":"是","labelEn":"Yes"}]},
    {"name":"interval","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":5,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "Logix 标签",
    "labelEn": "Logix Tag",
    "placeholderZh": "MyTag / Program:MainProgram.Counter / MyArray[0]",
    "placeholderEn": "MyTag / Program:MainProgram.Counter / MyArray[0]",
    "descriptionZh": "ControlLogix/CompactLogix 符号标签路径，支持结构体成员和数组下标",
    "descriptionEn": "ControlLogix/CompactLogix symbolic tag path with struct members and array indexes"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"SINT","labelZh":"sint","labelEn":"sint","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^.+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"}
  ]
}`

const df1Schema = `{
  "fields": [
    {"name":"location","labelZh":"串口设备","labelEn":"Serial Port","regex":"^[^\\s]+$","placeholderZh":"ttyUSB0","placeholderEn":"ttyUSB0","sort":1,"type":"input","required":true},
    {"name":"baudRate","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(1200|2400|4800|9600|19200|38400|57600|115200)$","sort":2,"type":"select","options":[{"value":"1200","labelZh":"1200","labelEn":"1200"},{"value":"2400","labelZh":"2400","labelEn":"2400"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"},{"value":"57600","labelZh":"57600","labelEn":"57600"},{"value":"115200","labelZh":"115200","labelEn":"115200"}]},
    {"name":"dataBits","labelZh":"数据位","labelEn":"Data Bits","regex":"^(5|6|7|8)$","sort":3,"type":"select","options":[{"value":"5","labelZh":"5","labelEn":"5"},{"value":"6","labelZh":"6","labelEn":"6"},{"value":"7","labelZh":"7","labelEn":"7"},{"value":"8","labelZh":"8","labelEn":"8"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^(noParity|oddParity|evenParity|none|odd|even|0|1|2)$","sort":4,"type":"select","options":[{"value":"evenParity","labelZh":"偶校验","labelEn":"Even"},{"value":"noParity","labelZh":"无校验","labelEn":"None"},{"value":"oddParity","labelZh":"奇校验","labelEn":"Odd"}]},
    {"name":"stopBits","labelZh":"停止位","labelEn":"Stop Bits","regex":"^(1|1\\.5|2)$","sort":5,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"1.5","labelZh":"1.5","labelEn":"1.5"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":6,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":7,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "DF1 文件地址",
    "labelEn": "DF1 File Address",
    "placeholderZh": "N7:0 / B3:0/0 / T4:0.PRE",
    "placeholderEn": "N7:0 / B3:0/0 / T4:0.PRE",
    "descriptionZh": "格式：文件类型+文件号:元素号[/位号][.PRE|.ACC]，支持 N/F/B/T/C/R/S/A/U/I/O",
    "descriptionEn": "Format: file type + file number:element[/bit][.PRE|.ACC], supporting N/F/B/T/C/R/S/A/U/I/O"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[A-Za-z][0-9]{1,3}:[0-9]{1,3}(/[0-9]{1,2})?$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[A-Za-z][0-9]{1,3}:[0-9]{1,3}(/[0-9]{1,2})?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[A-Za-z][0-9]{1,3}:[0-9]{1,3}(\\.[A-Za-z]+)?$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[A-Za-z][0-9]{1,3}:[0-9]{1,3}(\\.[A-Za-z]+)?$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[A-Za-z][0-9]{1,3}:[0-9]{1,3}(\\.[A-Za-z]+)?$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[A-Za-z][0-9]{1,3}:[0-9]{1,3}(\\.[A-Za-z]+)?$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^[A-Za-z][0-9]{1,3}:[0-9]{1,3}(\\.[A-Za-z]+)?$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^[A-Za-z][0-9]{1,3}:[0-9]{1,3}(\\.[A-Za-z]+)?$"}
  ]
}`

const hj212Schema = `{
  "fields": [
    {"name":"link","labelZh":"链路类型","labelEn":"Link Type","regex":"^[01]$","sort":1,"type":"select","required":true,"options":[{"value":"0","labelZh":"串口","labelEn":"Serial"},{"value":"1","labelZh":"TCP","labelEn":"TCP"}]},
    {"name":"location","labelZh":"串口设备/IP 地址","labelEn":"Serial Port / IP","regex":"^[^\\s]+$","placeholderZh":"ttyUSB0","placeholderEn":"ttyUSB0","sort":2,"type":"input","required":true},
    {"name":"port","labelZh":"TCP 端口","labelEn":"TCP Port","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"baudRate","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(1200|2400|4800|9600|19200|38400|57600|115200)$","sort":4,"type":"select","options":[{"value":"1200","labelZh":"1200","labelEn":"1200"},{"value":"2400","labelZh":"2400","labelEn":"2400"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"},{"value":"57600","labelZh":"57600","labelEn":"57600"},{"value":"115200","labelZh":"115200","labelEn":"115200"}]},
    {"name":"dataBits","labelZh":"数据位","labelEn":"Data Bits","regex":"^(5|6|7|8)$","sort":5,"type":"select","options":[{"value":"5","labelZh":"5","labelEn":"5"},{"value":"6","labelZh":"6","labelEn":"6"},{"value":"7","labelZh":"7","labelEn":"7"},{"value":"8","labelZh":"8","labelEn":"8"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^(noParity|oddParity|evenParity|none|odd|even|0|1|2)$","sort":6,"type":"select","options":[{"value":"noParity","labelZh":"无校验","labelEn":"None"},{"value":"evenParity","labelZh":"偶校验","labelEn":"Even"},{"value":"oddParity","labelZh":"奇校验","labelEn":"Odd"}]},
    {"name":"stopBits","labelZh":"停止位","labelEn":"Stop Bits","regex":"^(1|1\\.5|2)$","sort":7,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"1.5","labelZh":"1.5","labelEn":"1.5"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":8,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":9,"type":"input"},
    {"name":"ST","labelZh":"系统编码(ST)","labelEn":"System Code (ST)","regex":"^[0-9]{1,3}$","sort":10,"type":"input","required":true},
    {"name":"PW","labelZh":"密码(PW)","labelEn":"Password (PW)","regex":"^[^;\\r\\n]{0,9}$","sort":11,"type":"input"},
    {"name":"MN","labelZh":"设备编号(MN)","labelEn":"Device ID (MN)","regex":"^[^;\\r\\n]{1,27}$","sort":12,"type":"input","required":true}
  ],
  "variableAddress": {
    "labelZh": "污染物变量地址",
    "labelEn": "Pollutant Address",
    "placeholderZh": "RT!a34013-Rtd / MIN!w01018-Avg / HOUR!w01018-Cou",
    "placeholderEn": "RT!a34013-Rtd / MIN!w01018-Avg / HOUR!w01018-Cou",
    "descriptionZh": "格式：RT!/MIN!/HOUR!/DAY! + 污染物编码 + 后缀。RT 支持 -Rtd、-ZsRtd、-Flag、-SampleTime、-EFlag；分钟/小时/日支持 -Cou、-Min、-Avg、-Max、-ZsMin、-ZsAvg、-ZsMax、-Flag",
    "descriptionEn": "Format: RT!/MIN!/HOUR!/DAY! + pollutant code + suffix. RT supports -Rtd, -ZsRtd, -Flag, -SampleTime, -EFlag; period data supports -Cou, -Min, -Avg, -Max, -ZsMin, -ZsAvg, -ZsMax, -Flag"
  },
  "variableDataTypes": [
    {"value":"DOUBLE","labelZh":"double","labelEn":"double","addressRegex":"^RT!.*-(Rtd|ZsRtd)$|^(MIN|HOUR|DAY)!.*-(Cou|Min|Avg|Max|ZsMin|ZsAvg|ZsMax)$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^RT!.*-(Flag|SampleTime|EFlag)$|^(MIN|HOUR|DAY)!.*-Flag$"}
  ]
}`

const bacnetSchema = `{
  "fields": [
    {"name":"host","labelZh":"设备/BBMD IP","labelEn":"Device / BBMD IP","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"BACnet 端口","labelEn":"BACnet Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input"},
    {"name":"srcPort","labelZh":"本地端口","labelEn":"Source Port","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"bbmd","labelZh":"BBMD 注册","labelEn":"BBMD Register","regex":"^[01]$","sort":4,"type":"select","options":[{"value":"0","labelZh":"否","labelEn":"No"},{"value":"1","labelZh":"是","labelEn":"Yes"}]},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":5,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":6,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "BACnet 对象地址",
    "labelEn": "BACnet Object Address",
    "placeholderZh": "AI0 / AI0.Present_Value / DEV0.Object_Name / AV1.custom.85",
    "placeholderEn": "AI0 / AI0.Present_Value / DEV0.Object_Name / AV1.custom.85",
    "descriptionZh": "格式：对象前缀+实例号[.属性名|custom.属性ID]。前缀支持 AI/AO/AV/BI/BO/BV/MSI/MSO/MSV/ACC/DEV；省略属性时默认 Present_Value，DEV 必须指定属性",
    "descriptionEn": "Format: object prefix + instance [.property | custom.propertyId]. Prefixes: AI/AO/AV/BI/BO/BV/MSI/MSO/MSV/ACC/DEV; Present_Value is the default, DEV requires an explicit property"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"FLOAT","labelZh":"float","labelEn":"float","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^(AI|AO|AV|BI|BO|BV|MSI|MSO|MSV|ACC|DEV)[0-9]{1,7}(\\.[A-Za-z_][A-Za-z0-9_]*|\\.[Cc]ustom\\.[0-9]{1,7})?$"}
  ]
}`

const beckhoffADSSchema = `{
  "fields": [
    {"name":"host","labelZh":"PLC IP","labelEn":"PLC IP","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"AMS/TCP 端口","labelEn":"AMS/TCP Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input"},
    {"name":"srcAmsNetId","labelZh":"源 AMS Net ID","labelEn":"Source AMS Net ID","regex":"^([0-9]{1,3}\\.){5}[0-9]{1,3}$","sort":3,"type":"input"},
    {"name":"srcAdsPort","labelZh":"源 ADS 端口","labelEn":"Source ADS Port","regex":"^[0-9]{1,5}$","sort":4,"type":"input"},
    {"name":"dstAmsNetId","labelZh":"目标 AMS Net ID","labelEn":"Target AMS Net ID","regex":"^([0-9]{1,3}\\.){5}[0-9]{1,3}$","sort":5,"type":"input"},
    {"name":"dstAdsPort","labelZh":"目标 ADS 端口","labelEn":"Target ADS Port","regex":"^[0-9]{1,5}$","sort":6,"type":"input"},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":7,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":8,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "ADS 索引地址",
    "labelEn": "ADS Index Address",
    "placeholderZh": "0xF020,0 / 0x4020,0.80",
    "placeholderEn": "0xF020,0 / 0x4020,0.80",
    "descriptionZh": "格式：索引组,索引偏移[.字符串长度]，group/offset 支持十进制或 0x 前缀；STRING 必须追加 .长度，例如 0x4020,0.80",
    "descriptionEn": "Format: index group,index offset[.stringLength], group/offset accept decimal or 0x prefix; STRING requires .length, for example 0x4020,0.80"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^.+$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^.+$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^.+$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^.+$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+\\.[0-9]{1,3}$"}
  ]
}`

const srtpSchema = `{
  "fields": [
    {"name":"host","labelZh":"PLC IP","labelEn":"PLC IP","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"SRTP 端口","labelEn":"SRTP Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input"},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":3,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":4,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "SRTP 内存地址",
    "labelEn": "SRTP Memory Address",
    "placeholderZh": "%R1 / %I1 / %AI1 / %R1.3 / %R1#8",
    "placeholderEn": "%R1 / %I1 / %AI1 / %R1.3 / %R1#8",
    "descriptionZh": "1 基地址：%I/%Q/%M/%T/%SA/%SB/%SC/%S/%G/%AI/%AQ/%R + 地址。BIT 可用 %R1.3 取字位；STRING 用 #长度，例如 %R1#8",
    "descriptionEn": "1-based address: %I/%Q/%M/%T/%SA/%SB/%SC/%S/%G/%AI/%AQ/%R + address. BIT can use %R1.3 for a word bit; STRING uses #length, e.g. %R1#8"
  },
  "variableDataTypes": [
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^%[A-Za-z0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^%[A-Za-z0-9]+#[0-9]{1,3}$"}
  ]
}`

const snmpSchema = `{
  "fields": [
    {"name":"host","labelZh":"Agent IP","labelEn":"Agent IP","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"SNMP 端口","labelEn":"SNMP Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input"},
    {"name":"localPort","labelZh":"本地端口","labelEn":"Local Port","regex":"^[0-9]{1,5}$","sort":3,"type":"input"},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":4,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":5,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "Community|OID",
    "labelEn": "Community|OID",
    "placeholderZh": "public|1.3.6.1.2.1.1.1.0",
    "placeholderEn": "public|1.3.6.1.2.1.1.1.0",
    "descriptionZh": "格式：community|OID，例如 public|1.3.6.1.2.1.1.1.0；只支持 SNMPv2c GetRequest，单标签单 OID",
    "descriptionEn": "Format: community|OID, for example public|1.3.6.1.2.1.1.1.0; SNMPv2c GetRequest only, one OID per tag"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[^|\\r\\n]{1,64}\\|[0-9]{1,3}(\\.[0-9]{1,3}){1,31}$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[^|\\r\\n]{1,64}\\|[0-9]{1,3}(\\.[0-9]{1,3}){1,31}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[^|\\r\\n]{1,64}\\|[0-9]{1,3}(\\.[0-9]{1,3}){1,31}$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[^|\\r\\n]{1,64}\\|[0-9]{1,3}(\\.[0-9]{1,3}){1,31}$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[^|\\r\\n]{1,64}\\|[0-9]{1,3}(\\.[0-9]{1,3}){1,31}$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[^|\\r\\n]{1,64}\\|[0-9]{1,3}(\\.[0-9]{1,3}){1,31}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[^|\\r\\n]{1,64}\\|[0-9]{1,3}(\\.[0-9]{1,3}){1,31}$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^[^|\\r\\n]{1,64}\\|[0-9]{1,3}(\\.[0-9]{1,3}){1,31}$"}
  ]
}`

const iec61850Schema = `{
  "fields": [
    {"name":"host","labelZh":"IED IP","labelEn":"IED IP","regex":"^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"MMS 端口","labelEn":"MMS Port","regex":"^[0-9]{1,5}$","sort":2,"type":"input"},
    {"name":"defaultFC","labelZh":"默认功能约束","labelEn":"Default FC","regex":"^[A-Za-z0-9]{2}$","sort":3,"type":"select","options":[{"value":"ST","labelZh":"ST","labelEn":"ST"},{"value":"MX","labelZh":"MX","labelEn":"MX"},{"value":"SP","labelZh":"SP","labelEn":"SP"},{"value":"SV","labelZh":"SV","labelEn":"SV"},{"value":"CF","labelZh":"CF","labelEn":"CF"},{"value":"DC","labelZh":"DC","labelEn":"DC"},{"value":"CO","labelZh":"CO","labelEn":"CO"}]},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[0-9]{1,6}$","sort":4,"type":"input"},
    {"name":"intervalMs","labelZh":"命令间隔(ms)","labelEn":"Command Interval (ms)","regex":"^[0-9]{1,6}$","sort":5,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "IEC 61850 对象引用",
    "labelEn": "IEC 61850 Object Reference",
    "placeholderZh": "LD0/LLN0.Health[ST] / LD0/GGIO1.AnIn1.mag.f[MX]",
    "placeholderEn": "LD0/LLN0.Health[ST] / LD0/GGIO1.AnIn1.mag.f[MX]",
    "descriptionZh": "格式：LD/LN.DO.DA[FC]；省略 [FC] 时使用默认功能约束。例如 LD0/LLN0.Health[ST]",
    "descriptionEn": "Format: LD/LN.DO.DA[FC]; when [FC] is omitted the default functional constraint is used, e.g. LD0/LLN0.Health[ST]"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"INT8","labelZh":"int8","labelEn":"int8","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"UINT8","labelZh":"uint8","labelEn":"uint8","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^[A-Za-z0-9_]+/[A-Za-z0-9_]+(\\.[A-Za-z0-9_]+)*(\\[[A-Za-z0-9]{2}\\])?$"}
  ]
}`

const profinetSchema = s71500Schema

const mcSchema = `{
  "fields": [
    {"name":"transport_mode","labelZh":"传输模式","labelEn":"Transport","regex":"^[01]$","sort":1,"type":"select","options":[{"value":"0","labelZh":"TCP","labelEn":"TCP"},{"value":"1","labelZh":"UDP","labelEn":"UDP"}]},
    {"name":"host","labelZh":"PLC IP","labelEn":"PLC IP","regex":"^[^\\s]+$","placeholderZh":"192.168.1.10","placeholderEn":"192.168.1.10","sort":2,"type":"input","required":true},
    {"name":"port","labelZh":"端口","labelEn":"Port","regex":"^[0-9]{1,5}$","placeholderZh":"2000","placeholderEn":"2000","sort":3,"type":"input"},
    {"name":"src-port","labelZh":"本地端口","labelEn":"Source Port","regex":"^[0-9]{1,5}$","placeholderZh":"52001","placeholderEn":"52001","sort":4,"type":"input"},
    {"name":"timeout","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[1-9][0-9]{0,5}$","placeholderZh":"15000","placeholderEn":"15000","sort":5,"type":"input"},
    {"name":"interval","labelZh":"采集间隔(ms)","labelEn":"Interval (ms)","regex":"^[0-9]{1,5}$","placeholderZh":"20","placeholderEn":"20","sort":6,"type":"input"},
    {"name":"retry","labelZh":"重试次数","labelEn":"Retry Count","regex":"^([0-9]|10)$","sort":7,"type":"input"}
  ],
  "variableAddress": {"labelZh":"软元件地址","labelEn":"Device Address","placeholderZh":"D0 / M0 / X0 / D0.3","placeholderEn":"D0 / M0 / X0 / D0.3","descriptionZh":"三菱软元件地址，例如 D0、M0、X0；字内位写作 D0.3","descriptionEn":"Mitsubishi device address, for example D0, M0, X0; word bit uses D0.3"},
  "variableDataTypes": [
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"}
  ]
}`

const fxSchema = `{
  "fields": [
    {"name":"device","labelZh":"串口设备","labelEn":"Serial Port","regex":"^[^\\s]+$","placeholderZh":"ttyS0","placeholderEn":"ttyS0","sort":1,"type":"input","required":true},
    {"name":"baud","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(4800|9600|19200|38400)$","sort":2,"type":"select","options":[{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"}]},
    {"name":"data","labelZh":"数据位","labelEn":"Data Bits","regex":"^[78]$","sort":3,"type":"select","options":[{"value":"7","labelZh":"7","labelEn":"7"},{"value":"8","labelZh":"8","labelEn":"8"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^[012]$","sort":4,"type":"select","options":[{"value":"2","labelZh":"偶校验","labelEn":"Even"},{"value":"0","labelZh":"无","labelEn":"None"},{"value":"1","labelZh":"奇校验","labelEn":"Odd"}]},
    {"name":"stop","labelZh":"停止位","labelEn":"Stop Bits","regex":"^[12]$","sort":5,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"timeout","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[1-9][0-9]{0,5}$","placeholderZh":"3000","placeholderEn":"3000","sort":6,"type":"input"},
    {"name":"interval","labelZh":"采集间隔(ms)","labelEn":"Interval (ms)","regex":"^[0-9]{1,5}$","placeholderZh":"20","placeholderEn":"20","sort":7,"type":"input"},
    {"name":"retry","labelZh":"重试次数","labelEn":"Retry Count","regex":"^([0-9]|10)$","sort":8,"type":"input"}
  ],
  "variableAddress": {"labelZh":"软元件地址","labelEn":"Device Address","placeholderZh":"D0 / M0 / X0 / D0.3","placeholderEn":"D0 / M0 / X0 / D0.3","descriptionZh":"三菱 FX 软元件地址，例如 D0、M0、X0；字内位写作 D0.3","descriptionEn":"Mitsubishi FX device address, for example D0, M0, X0; word bit uses D0.3"},
  "variableDataTypes": [
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[A-Za-z]+[0-9]+(\\.[0-9]{1,2})?$"}
  ]
}`

const comliSchema = `{
  "fields": [
    {"name":"device","labelZh":"串口设备","labelEn":"Serial Port","regex":"^[^\\s]+$","placeholderZh":"ttyS0","placeholderEn":"ttyS0","sort":1,"type":"input","required":true},
    {"name":"baud","labelZh":"波特率","labelEn":"Baud Rate","regex":"^(1200|2400|4800|9600|19200|38400|57600|115200)$","sort":2,"type":"select","options":[{"value":"9600","labelZh":"9600","labelEn":"9600"},{"value":"1200","labelZh":"1200","labelEn":"1200"},{"value":"2400","labelZh":"2400","labelEn":"2400"},{"value":"4800","labelZh":"4800","labelEn":"4800"},{"value":"19200","labelZh":"19200","labelEn":"19200"},{"value":"38400","labelZh":"38400","labelEn":"38400"},{"value":"57600","labelZh":"57600","labelEn":"57600"},{"value":"115200","labelZh":"115200","labelEn":"115200"}]},
    {"name":"data","labelZh":"数据位","labelEn":"Data Bits","regex":"^[5-8]$","sort":3,"type":"select","options":[{"value":"8","labelZh":"8","labelEn":"8"},{"value":"5","labelZh":"5","labelEn":"5"},{"value":"6","labelZh":"6","labelEn":"6"},{"value":"7","labelZh":"7","labelEn":"7"}]},
    {"name":"parity","labelZh":"校验位","labelEn":"Parity","regex":"^[012]$","sort":4,"type":"select","options":[{"value":"0","labelZh":"无","labelEn":"None"},{"value":"1","labelZh":"奇校验","labelEn":"Odd"},{"value":"2","labelZh":"偶校验","labelEn":"Even"}]},
    {"name":"stop","labelZh":"停止位","labelEn":"Stop Bits","regex":"^[12]$","sort":5,"type":"select","options":[{"value":"1","labelZh":"1","labelEn":"1"},{"value":"2","labelZh":"2","labelEn":"2"}]},
    {"name":"timeout","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[1-9][0-9]{0,5}$","placeholderZh":"3000","placeholderEn":"3000","sort":6,"type":"input"},
    {"name":"interval","labelZh":"采集间隔(ms)","labelEn":"Interval (ms)","regex":"^[0-9]{1,5}$","placeholderZh":"20","placeholderEn":"20","sort":7,"type":"input"},
    {"name":"retry","labelZh":"重试次数","labelEn":"Retry Count","regex":"^([0-9]|10)$","sort":8,"type":"input"}
  ],
  "variableAddress": {"labelZh":"站号!区域.偏移","labelEn":"Identity!Area.Offset","placeholderZh":"1!1.0 / 1!0.3 / 1!1.0.3","placeholderEn":"1!1.0 / 1!0.3 / 1!1.0.3","descriptionZh":"站号 1-247；区域 0=位区，1=寄存器区；寄存器位地址写作 1!1.0.3（位 0-15）","descriptionEn":"Identity 1-247; area 0=bits, 1=registers; register bit address uses 1!1.0.3 (bit 0-15)"},
  "variableDataTypes": [
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^[0-9]{1,3}!(0\\.[0-9]{1,5}|1\\.[0-9]{1,4}\\.(?:[0-9]|1[0-5]))$"},
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^[0-9]{1,3}!(0\\.[0-9]{1,5}|1\\.[0-9]{1,4}\\.(?:[0-9]|1[0-5]))$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^[0-9]{1,3}!1\\.[0-9]{1,4}$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^[0-9]{1,3}!1\\.[0-9]{1,4}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^[0-9]{1,3}!1\\.[0-9]{1,4}$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^[0-9]{1,3}!1\\.[0-9]{1,4}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^[0-9]{1,3}!1\\.[0-9]{1,4}$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^[0-9]{1,3}!1\\.[0-9]{1,4}$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^[0-9]{1,3}!1\\.[0-9]{1,4}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^[0-9]{1,3}!1\\.[0-9]{1,4}$"}
  ]
}`

const mewtocolSchema = `{
  "fields": [
    {"name":"location","labelZh":"PLC 地址","labelEn":"PLC Address","regex":"^[A-Za-z0-9.-]+$","placeholderZh":"192.168.1.10","placeholderEn":"192.168.1.10","sort":1,"type":"input","required":true},
    {"name":"port","labelZh":"端口","labelEn":"Port","regex":"^[0-9]{1,5}$","placeholderZh":"2000","placeholderEn":"2000","sort":2,"type":"input"},
    {"name":"unit","labelZh":"站号","labelEn":"Unit","regex":"^[1-9][0-9]?$","placeholderZh":"1","placeholderEn":"1","sort":3,"type":"input"},
    {"name":"timeoutMs","labelZh":"超时(ms)","labelEn":"Timeout (ms)","regex":"^[1-9][0-9]{0,5}$","placeholderZh":"15000","placeholderEn":"15000","sort":4,"type":"input"}
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "placeholderZh": "X0.0 / DT0 / DT0.10",
    "placeholderEn": "X0.0 / DT0 / DT0.10",
    "descriptionZh": "区域前缀 + 十进制地址；位地址用 X0.0，字符串用 DT0.10 表示字节长度",
    "descriptionEn": "Area prefix + decimal address; bit address X0.0, string DT0.10 for byte length"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}(\\.[0-9]{1,2})?$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}(\\.[0-9]{1,2})?$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^(X|Y|R|L|LD|T|C|DT|FL|IX|IY|ID)[0-9]{1,5}$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^(DT|LD|FL)[0-9]{1,5}\\.[0-9]{1,3}[HL]?$"}
  ]
}`

const fetchWriteSchema = `{
  "fields": [
    {
      "name": "location",
      "labelZh": "PLC 地址",
      "labelEn": "PLC Address",
      "regex": "^\\S+$",
      "sort": 1,
      "type": "input",
      "required": true,
      "placeholderZh": "127.0.0.1",
      "placeholderEn": "127.0.0.1"
    },
    {
      "name": "port",
      "labelZh": "端口",
      "labelEn": "Port",
      "regex": "^(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3})$",
      "sort": 2,
      "type": "input",
      "required": true,
      "placeholderZh": "102",
      "placeholderEn": "102"
    },
    {
      "name": "timeoutMs",
      "labelZh": "超时时间(ms)",
      "labelEn": "Timeout (ms)",
      "sort": 3,
      "type": "input",
      "placeholderZh": "3000",
      "placeholderEn": "3000"
    },
    {
      "name": "concurrency",
      "labelZh": "连接数",
      "labelEn": "Connections",
      "sort": 4,
      "type": "input",
      "placeholderZh": "1",
      "placeholderEn": "1"
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Address",
    "regex": "^(DB[0-9]{1,3}\\.(DBW[0-9]{1,4}(\\.[0-9]{1,3})?|DBD[0-9]{1,4}|DBB[0-9]{1,4}(\\.[0-9]{1,3})?|DBX[0-9]{1,4}\\.[0-7])|(M|I|Q|PEPA)[0-9]{1,4}(\\.[0-9]{1,3})?|(Z|T)[0-9]{1,4}(\\.([0-9]|1[0-5]))?)$",
    "placeholderZh": "例如 DB1.DBW0、M0.1、I0、Q0",
    "placeholderEn": "e.g. DB1.DBW0, M0.1, I0, Q0"
  },
  "variableDataTypes": [
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^(DB[0-9]{1,3}\\.(DBW[0-9]{1,4}(\\.([0-9]|1[0-5]))?|DBB[0-9]{1,4}(\\.[0-7])?|DBX[0-9]{1,4}\\.[0-7])|(M|I|Q|PEPA)[0-9]{1,4}(\\.[0-7])?|(Z|T)[0-9]{1,4}(\\.([0-9]|1[0-5]))?)$"},
    {"value":"BIT","labelZh":"bit","labelEn":"bit","addressRegex":"^(DB[0-9]{1,3}\\.(DBW[0-9]{1,4}(\\.([0-9]|1[0-5]))?|DBB[0-9]{1,4}(\\.[0-7])?|DBX[0-9]{1,4}\\.[0-7])|(M|I|Q|PEPA)[0-9]{1,4}(\\.[0-7])?|(Z|T)[0-9]{1,4}(\\.([0-9]|1[0-5]))?)$"},
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBD|DBB)[0-9]{1,4}|(M|I|Q|PEPA|Z|T)[0-9]{1,4})$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBD|DBB)[0-9]{1,4}|(M|I|Q|PEPA|Z|T)[0-9]{1,4})$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBD|DBB)[0-9]{1,4}|(M|I|Q|PEPA|Z|T)[0-9]{1,4})$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBD|DBB)[0-9]{1,4}|(M|I|Q|PEPA|Z|T)[0-9]{1,4})$"},
    {"value":"INT64","labelZh":"int64","labelEn":"int64","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBD|DBB)[0-9]{1,4}|(M|I|Q|PEPA|Z|T)[0-9]{1,4})$"},
    {"value":"UINT64","labelZh":"uint64","labelEn":"uint64","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBD|DBB)[0-9]{1,4}|(M|I|Q|PEPA|Z|T)[0-9]{1,4})$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBD|DBB)[0-9]{1,4}|(M|I|Q|PEPA|Z|T)[0-9]{1,4})$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBD|DBB)[0-9]{1,4}|(M|I|Q|PEPA|Z|T)[0-9]{1,4})$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBB)[0-9]{1,4}\\.[0-9]{1,3}|(M|I|Q|PEPA)[0-9]{1,4}\\.[0-9]{1,3})$"},
    {"value":"BYTE_ARRAY","labelZh":"byte_array","labelEn":"byte_array","addressRegex":"^(DB[0-9]{1,3}\\.(DBW|DBB)[0-9]{1,4}\\.[0-9]{1,3}|(M|I|Q|PEPA)[0-9]{1,4}\\.[0-9]{1,3})$"}
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
	{BuiltinModbusASCIIID, DriverTypeModbus, ModelModbusASCII, DriverModbusASCII, modbusAsciiSchema},
	{BuiltinIEC101ID, DriverTypeIEC, ModelIEC101, DriverIEC101, iec101Schema},
	{BuiltinIEC104ID, DriverTypeIEC, ModelIEC104, DriverIEC104, iec104Schema},
	{BuiltinIEC102ID, DriverTypeIEC, ModelIEC102, DriverIEC102, iec102Schema},
	{BuiltinIEC103ID, DriverTypeIEC, ModelIEC103, DriverIEC103, iec103Schema},
	{BuiltinS71500ID, DriverTypeS7, ModelS71500, DriverS71500, s71500Schema},
	{BuiltinS7200ID, DriverTypeS7, ModelS7200, DriverS7200, s7200Schema},
	{BuiltinMPIID, DriverTypeS7, ModelMPI, DriverMPI, mpiSchema},
	{BuiltinPPIID, DriverTypeS7, ModelPPI, DriverPPI, ppiSchema},
	{BuiltinFetchWriteID, DriverTypeS7, ModelFetchWrite, DriverFetchWrite, fetchWriteSchema},
	{BuiltinKNXID, DriverTypeKNX, ModelKNXnetIP, DriverKNXnetIP, knxSchema},
	{BuiltinKNDID, DriverTypeCNC, ModelKND, DriverKND, kndSchema},
	{BuiltinHeidenhainID, DriverTypeCNC, ModelHeidenhain, DriverHeidenhain, heidenhainSchema},
	{BuiltinFanucFocasID, DriverTypeCNC, ModelFanucFocas, DriverFanucFocas, focasSchema},
	{BuiltinMTConnectID, DriverTypeCNC, ModelMTConnect, DriverMTConnect, mtconnectSchema},
	{BuiltinKukaID, DriverTypeKuka, ModelKuka, DriverKuka, kukaSchema},
	{BuiltinCodesysV3ID, DriverTypeCodesys, ModelCodesysV3, DriverCodesysV3, codesysV3Schema},
	{BuiltinAB5000ID, DriverTypeAB5000, ModelAB5000, DriverAB5000, ab5000Schema},
	{BuiltinDF1ID, DriverTypeAB5000, ModelDF1, DriverDF1, df1Schema},
	{BuiltinHJ212ID, DriverTypeHJ212, ModelHJ212, DriverHJ212, hj212Schema},
	{BuiltinBACnetID, DriverTypeBACnet, ModelBACnet, DriverBACnet, bacnetSchema},
	{BuiltinADSID, DriverTypeADS, ModelBeckhoffADS, DriverADS, beckhoffADSSchema},
	{BuiltinSRTPID, DriverTypeSRTP, ModelSRTP, DriverSRTP, srtpSchema},
	{BuiltinSNMPID, DriverTypeSNMP, ModelSNMPV2, DriverSNMP, snmpSchema},
	{BuiltinIEC61850ID, DriverTypeIEC, ModelIEC61850, DriverIEC61850, iec61850Schema},
	{BuiltinCOMLIID, DriverTypeABB, ModelCOMLI, DriverCOMLI, comliSchema},
	{BuiltinFXID, DriverTypeMitsubishi, ModelMCFX, DriverFX, fxSchema},
	{BuiltinA1EID, DriverTypeMitsubishi, ModelMC1E, DriverA1E, mcSchema},
	{BuiltinQnA3EID, DriverTypeMitsubishi, ModelMC3E, DriverQnA3E, mcSchema},
	{BuiltinQnA4EID, DriverTypeMitsubishi, ModelMC4E, DriverQnA4E, mcSchema},
	{BuiltinOpcUaID, DriverTypeOpcUa, ModelOpcUa, DriverOpcUa, opcUaSchema},
	{BuiltinFinsID, DriverTypeOmron, ModelFins, DriverFins, finsSchema},
	{BuiltinHostlinkCmodeID, DriverTypeOmron, ModelHostlinkCmode, DriverHostlinkCmode, hostlinkCmodeSchema},
	{BuiltinMewtocolID, DriverTypePanasonic, ModelMewtocol, DriverMewtocol, mewtocolSchema},
	{BuiltinOmronCipID, DriverTypeOmron, ModelOmronCip, DriverOmronCip, omronCipSchema},
	{BuiltinEthernetIPID, DriverTypeEthernetIP, ModelEthernetIP, DriverEthernetIP, ethernetIPSchema},
	{BuiltinEthernetIP5500ID, DriverTypeAllenBradley, ModelEthernetIP5500, DriverEthernetIP5500, ethernetIP5500Schema},
	{BuiltinEthernetIP1400ID, DriverTypeAllenBradley, ModelEthernetIP1400, DriverEthernetIP1400, ethernetIP1400Schema},
	{BuiltinEtherCATID, DriverTypeEtherCAT, ModelEtherCAT, DriverEtherCAT, etherCATSchema},
	{BuiltinProfinetID, DriverTypeProfinet, ModelProfinet, DriverProfinet, profinetSchema},
	{BuiltinDNP3ID, DriverDNP3, ModelDNP3, DriverDNP3, dnp3Schema},
	{BuiltinDLT6452007ID, DriverTypeDLT645, ModelDLT6452007, DriverDLT6452007, dlt645Schema},
	{BuiltinDLT6451997ID, DriverTypeDLT645, ModelDLT6451997, DriverDLT6451997, dlt645Schema},
	{BuiltinCJT1882004ID, DriverTypeCJT188, ModelCJT1882004, DriverCJT1882004, cjt188Schema},
	{BuiltinCJT1882018ID, DriverTypeCJT188, ModelCJT1882018, DriverCJT1882018, cjt188Schema},
	{BuiltinDLT69845ID, DriverTypeDLT698, ModelDLT69845, DriverDLT69845, dlt698Schema},
}
