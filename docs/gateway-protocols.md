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

| 协议 | 完成状态 | 验证状态 |
|---|---|------|
| Modbus TCP（`modbus` / `modbusTcp`） | 已完成 | 已验证  |
| Modbus RTU（`modbus` / `modbusRtu`） | 已完成 | 已验证  |
| Modbus RTU over TCP（`modbus` / `modbusRtuOverTcp`） | 已完成 | 已验证  |
| Modbus ASCII（`modbus` / `Modbus Ascii`） | 已完成 | 已验证  |
| IEC 60870-5-101（`IEC` / `IEC101`） | 已完成 | 未验证  |
| IEC 60870-5-102（`IEC` / `IEC102`） | 已完成 | 未验证  |
| IEC 60870-5-103（`IEC` / `IEC103`） | 已完成 | 未验证  |
| IEC 60870-5-104（`IEC` / `IEC104`） | 已完成 | 已验证  |
| Siemens S7-1500（`s7` / `s71500`） | 已完成 | 已验证  |
| Siemens S7-200（`s7` / `s7-200`） | 已完成 | 已验证  |
| OPC UA（`opcUa` / `opcUa`） | 已完成 | 已验证  |
| Omron FINS（`omron` / `fins`） | 已完成 | 已验证  |
| Omron HostLink C-Mode（`omron` / `hostlinkCmode`） | 已完成 | 未验证  |
| Omron EtherNet/IP CIP（`omron` / `omronCip`） | 已完成 | 已验证  |
| EtherNet/IP CIP（`ethernetIp` / `ethernetIp`） | 已完成 | 已验证  |
| EtherNet/IP ControlLogix 5500（`ethernetIp` / `ethernetIp5500`） | 已完成 | 未验证 |
| EtherCAT（`ethercat` / `ethercat`，经 TwinCAT ADS） | 已完成 | 已验证  |
| PROFINET（`profinet` / `profinet`，经 S7 通信） | 已完成 | 已验证  |

## 推送协议

| 协议 | 完成状态 | 验证状态 |
|---|---|---|
| MQTT（发布 + 命令订阅） | 已完成 | 已验证 |
| Kafka（发布 + 命令消费） | 已完成 | 未验证 |

> 报告按仓库约定不自动提交；需要时由维护者单独归档。

## 维护约定

- 新增协议时先在“南向采集协议”或“北向推送协议”表中追加一行，默认标记为 `未完成 / 未验证`。
- 协议实现并注册后，将完成状态改为 `已完成`。
- 通过模拟器或真实设备完成端到端采集与反控验证后，再将验证状态改为 `已验证`，并在“现有验证记录”中补充报告路径和结论。
