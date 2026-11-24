# Gateway 协议支持清单

用于跟踪网关各协议驱动的实现与验证状态。

## 状态定义

| 状态 | 含义 |
|---|---|
| 已完成 | 驱动代码、Schema、驱动注册、采集创建/更新链路均已实现 |
| 未完成 | 上述任一环节缺失 |
| 已验证 | 已通过 `PlcSimulator` 或真实设备完成采集与反控端到端验证，并留有报告 |
| 未验证 | 仅完成代码与单元测试，或端到端测试被阻塞 |

> 单元测试、编译通过不计入“已验证”。验证报告默认放在 `test/gateway/report/`，按仓库规范不自动提交。

## 采集协议

| 协议 | 完成状态 | 验证状态 | 适用于 |
|---|---|---|---|
| Modbus TCP（`modbus` / `modbusTcp`） | 已完成 | 已验证 | 通用 Modbus 设备 |
| Modbus RTU（`modbus` / `modbusRtu`） | 已完成 | 已验证 | 通用 Modbus 设备 |
| Modbus RTU over TCP（`modbus` / `modbusRtuOverTcp`） | 已完成 | 已验证 | 通用 Modbus 设备 |
| Modbus ASCII（`modbus` / `Modbus Ascii`） | 已完成 | 已验证 | 通用 Modbus 设备 |
| IEC 60870-5-101（`IEC` / `IEC101`） | 已完成 | 未验证 | 电力 IEC 60870-5-101 设备 |
| IEC 60870-5-102（`IEC` / `IEC102`） | 已完成 | 未验证 | 电能计量 IEC 60870-5-102 设备 |
| IEC 60870-5-103（`IEC` / `IEC103`） | 已完成 | 未验证 | 保护装置 IEC 60870-5-103 设备 |
| IEC 60870-5-104（`IEC` / `IEC104`） | 已完成 | 已验证 | 电力调度 IEC 60870-5-104 设备 |
| Siemens S7-1500（`siemens` / `s7-1500`） | 已完成 | 已验证 | 西门子 S7-1500 PLC |
| Siemens S7-200（`siemens` / `s7-200`） | 已完成 | 已验证 | 西门子 S7-200 PLC |
| Siemens MPI（`siemens` / `mpi`） | 已完成 | 未验证 | 西门子 S7-300/400 MPI 串口 |
| Siemens PPI（`siemens` / `ppi`） | 已完成 | 已验证（TCP 读） | 西门子 S7-200 PPI 串口/串口服务器 |
| OPC UA（`opcUa` / `opcUa`） | 已完成 | 已验证 | 支持 OPC UA 的 PLC / 系统 |
| Omron FINS（`omron` / `fins`） | 已完成 | 已验证 | 欧姆龙 PLC |
| Omron HostLink C-Mode（`omron` / `hostlinkCmode`） | 已完成 | 未验证 | 欧姆龙 PLC |
| Omron EtherNet/IP CIP（`omron` / `omronCip`） | 已完成 | 已验证 | 欧姆龙 PLC |
| EtherNet/IP CIP（`ethernetIp` / `ethernetIp`） | 已完成 | 已验证 | 罗克韦尔 PLC / 通用 EtherNet/IP 设备 |
| EtherNet/IP ControlLogix 5500（`ethernetIp` / `ethernetIp5500`） | 已完成 | 未验证 | 罗克韦尔 ControlLogix / CompactLogix |
| EtherNet/IP MicroLogix 1400（`ethernetIp` / `ethernetIp1400`） | 已完成 | 未验证 | 罗克韦尔 MicroLogix 1400 |
| EtherCAT（`ethercat` / `ethercat`，经 TwinCAT ADS） | 已完成 | 已验证 | 倍福 TwinCAT / EtherCAT 设备 |
| PROFINET（`profinet` / `profinet`，经 S7 通信） | 已完成 | 已验证 | 西门子 PROFINET PLC |
| DNP 3.0（`dnp3` / `dnp3`） | 已完成 | 未验证 | 电力/水务/能源 DNP3 从站 |
| DL/T 645-2007（`dlt645` / `dlt645_2007`） | 已完成 | 已验证（TCP 读） | 多功能电能表 |
| DL/T 645-1997（`dlt645` / `dlt645_1997`） | 已完成 | 未验证 | 老式多功能电能表 |
| CJ/T 188-2004（`cjt188` / `cjt188_2004`） | 已完成 | 未验证 | 水表、热量表、燃气表、电表 |
| CJ/T 188-2018（`cjt188` / `cjt188_2018`） | 已完成 | 已验证（TCP 读） | 水表、热量表、燃气表、电表 |
| DL/T 698.45（`dlt698` / `dlt698_45`） | 已完成 | 未验证 | 支持 698.45 GET 单对象读的电表 |
| KNXnet/IP（`KNX` / `KNXnet/IP`） | 已完成 | 未验证 | KNX 楼宇自动化网关 |
| KND CNC（`CNC` / `KND`） | 已完成 | 未验证 | KND 数控系统 HTTP 接口 |
| HEIDENHAIN LSV2（`CNC` / `heidenhain`） | 已完成 | 未验证 | HEIDENHAIN TNC 数控系统 |
| Fanuc FOCAS（`CNC` / `fanuc focas`） | 框架已完成 | 未验证 | FANUC CNC，需 `libfocas32.so.1` |

## 推送协议

| 协议 | 完成状态 | 验证状态 | 适用于 |
|---|---|---|---|
| MQTT（发布 + 命令订阅） | 已完成 | 已验证 | MQTT Broker / 消息平台 |
| Kafka（发布 + 命令消费） | 已完成 | 未验证 | Kafka 集群 / 消息平台 |

> 报告按仓库约定不自动提交；需要时由维护者单独归档。

## 维护约定

- 新增协议时先在“南向采集协议”或“北向推送协议”表中追加一行，默认标记为 `未完成 / 未验证`。
- 协议实现并注册后，将完成状态改为 `已完成`。
- 通过模拟器或真实设备完成端到端采集与反控验证后，再将验证状态改为 `已验证`，并在“现有验证记录”中补充报告路径和结论。

- `CNC` / `fanuc focas`：FOCAS 驱动框架已注册，库路径预留为 `plugins/dependencies/focas/`，缺少 `libfocas32.so.1` 时连接保持未连接并重试
- `CNC` / `heidenhain`：HEIDENHAIN LSV2 TCP 采集，登录后读取 PLC 区和 CNC 语义项；当前只读
- `CNC` / `KND`：KND 数控系统 HTTP/JSON 采集与 R 区/变量写入，默认端口 8000；URL/JSON 字段随固件版本变化
- `KNX` / `KNXnet/IP`：KNXnet/IP 组对象读写采集，UDP 3671，先发现网关再建立隧道；支持 1 位、1/2/4 字节整数和 4 字节浮点，读失败会断开并走统一重连指数退避
- `siemens` / `ppi`：S7-200 PPI 主站，默认 9600/8/even/1、站号 2；支持串口和 TCP 透传、V/I/Q/M/SM/AI/AQ/DB 地址，读失败断开并走统一重连指数退避；当前仅实现读
- `siemens` / `mpi`：西门子 MPI 串口主站，默认 38400/8/odd/1，站号 2；链路帧包含 S7 PDU、LRC 和 `10 10` 转义，读失败会断开串口并走统一重连指数退避；当前实现只读
