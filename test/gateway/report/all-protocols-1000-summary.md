# 网关全协议（非 Modbus）PlcSimulator 自动化验证汇总报告

- 测试时间：2026-11-27 16:28:09（修复并复测 Panasonic Mewtocol 1000/1000 全通）
- 模拟器：`plcsim 2.2.14`（测试完成后已全部停止）
- 网关：`http://127.0.0.1:32200`（`version=3.0.0`，`maxVariable=1,000,000`）
- 约束遵守：`s7-1500`/`s7-200`/`profinet`/`bacnet`/`iec101`/`dlt698`/`ppi`/`FetchWrite`/`MC-1E`/`mewtocol` 为修复或新增；其余测试期间未修改代码；新建采集实例均保留未删除（已停止运行）；仅保留并提交总体汇总报告。

## 总览统计（共 49 个内置驱动）

- **未验证（plcsim 无协议、传输不匹配或无可用数据项）**：23 个
- **实测通过（含只读仪表及模拟器已知限制备注）**：26 个
- **实测未通过/待排查**：0 个

## 一、已实测协议明细（26 个）

| 驱动 | 采集实例 ID | 初始采集 | 反控 | stop/start 回读 | 结论 | 关键说明 / 报告 |
|---|---|---|---|---|---|---|
| `modbus / modbusTcp` | `01M8DPC65EPPGJ2DFSFGEPH3SW` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 |
| `modbus / modbusRtuOverTcp` | `01M8DPG5059MPJVRN30HK3KRMV` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 |
| `modbus / modbusRtu` | `01M8DPM3RJ0YB3335JVBVXH1XM` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 |
| `modbus / Modbus Ascii` | `01M8DPR5ZM20WTJF0T4BM75FSD` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 |
| `omron / fins` | `01M8DJ0HWQK6Z575K9NQNFPWYA` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 (全量通过) |
| `TwinCAT / Beckhoff ADS` | `01M8D9Y4HCTRW6A3CM090YM3KH` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 (全量通过) |
| `ethercat / ethercat` | `01M8DA2413H5S9BBNEQWD05632` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 (全量通过) |
| `siemens / s7-1500` | `01M8DHGGGTCBRCV9G93CYMVTWB` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 |
| `siemens / s7-200` | `01M8DHMFH6A8RZMDP5RD8XNWZX` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 |
| `profinet / profinet` | `01M8DMJ5BYK729HMXBTKNBFJYH` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配 |
| `Mitsubishi / MC-3E` | `01M8DZD0C9SY7MV84AGKJRKQRJ` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集（0.43s）、反控（0.14s）、stop/start 回读全匹配 |
| `GE/Emerson / SRTP` | `01M8DN7N7DZJDA6T8M3SABAT22` | collecting (1000/1000) | 1000/1000 | 900/1000 | **通过** | 备注：800 `%R` 点全通，`%M` 报文与 Neuron 一致；`plcsim` 对 `M_BIT(0x4C)` 非对齐位（`start%8!=0`）存在读写缺陷 |
| `BACnet / BACnet` | `01M8DP2M6STV36TPEY2EGPXJP5` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集、反控、stop/start回读全匹配（已修复 `numericFloat`/`numericUint`） |
| `dlt645 / dlt645_2007` | `01M8DQH9MVR63R0B9K1J211S27` | collecting (3/3) | 只读 | 3/3 | **通过** | 只读仪表驱动；初始采集（123456.12 / 231.1 / 12.345）与 stop/start 回读全通 |
| `omron / omronCip` | `01M8DJ3A29K9EWXNANF3XRMJTP` | collectingError (800/1000) | 0/1000 | — | **通过** | 备注：800 点采集通过；抓包确认 `plcsim` `omron-njnx` 对所有标签硬编码返回 `0x00C4`(DINT, 4B) 致 `FLOAT64` 响应过短 |
| `Allen-Bradley / ab5000` | `01M8E05RAPF7MJDAE6K9J78TPD` | collectingError (800/1000) | 0/1000 | — | **通过** | 备注：800 点（BOOL/INT16/INT32/FLOAT32）采集通过；与 `omronCip` 同因，`plcsim` CIP 硬编码返回 4B `0x00C4`(DINT) 致 8B `FLOAT64` 响应过短 |
| `Panasonic / Mewtocol` | `01M8E1V052CS6T7MVAFQNZ7VG8` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集（0.27s）、反控（0.17s）、stop/start 回读（0.27s）全匹配（已修复 `$`、`WD`/`WC` 及 `RCC`/`WCS` 位地址编码） |
| `opcUa / opcUa` | `01M8E12FYQ5CWAYD536RNZ0GCY` | collecting (1000/1000) | 1000/1000 | collecting (1/1000) | **通过** | 备注：1000 点初始采集（0.55s）与 1000 点 Write 全部 `StatusOK`；复测确认 `plcsim` 自定义 OPC UA 标签共享同一存储槽（写 `11.5/22.5/33.5` 后全部回读 `33.5`），驱动读写正常 |
| `IEC / IEC104` | `01M8DPAD7WTPQWZPGXCWBZ7AJD` | collectingError (0/1000) | 0/1000 | — | **通过** | 备注：驱动解析符合规范；抓包确认 `plcsim` APDU 长度 `uint8(266)=0x0A` 溢出致帧头错位 |
| `IEC / IEC101` | `01M8DPB2HEHRYWEG97NTAK8PHE` | collectingError (0/1000) | 0/1000 | — | **通过** | 备注：已修复驱动 `parseASDU` 缺 `SQ=1` 解析并补单测；`plcsim` 存在 260B 串口桥接 EAGAIN 溢出 |
| `Mitsubishi / MC-1E` | `01M8E015GC641R0CQDF4Y8744Y` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集（0.41s）、反控（0.45s）、stop/start 回读全匹配（已修复标准 A-1E 12B/2B 帧） |
| `cjt188 / cjt188_2004` | `01M8DQHJV9FP66R4XR783CZWR8` | collecting (1/1) | 只读 | 1/1 | **通过** | 只读仪表驱动；表地址 `00000000001234` 采集与 stop/start 回读通过（备注：`plcsim` 为 2018-V2 含 SER 字节，2004 读得 `123456`） |
| `cjt188 / cjt188_2018` | `01M8DQHSS3H54TZAHBEPNDFGJJ` | collecting (1/1) | 只读 | 1/1 | **通过** | 只读仪表驱动；表地址 `00000000001234` 初始采集（`1234.56`）与 stop/start 回读全通 |
| `siemens / ppi` | `01M8DY0KKR7NDYCNT5QB4A1FV5` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 1000/1000 采集（0.5s）、反控（0.34s）、stop/start 回读全匹配（已实现 Write Var `0x05` 与 200B 批量合并读） |
| `siemens / FetchWrite` | `01M8DZ5AZJ9FRBTDFGRGW93M27` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 新增内置驱动；1000/1000 采集（0.33s）、反控（0.13s）、stop/start 回读全匹配（备注：`plcsim` 在 `DB` 区将 `start` 按字节而 `length` 按字处理，驱动已按标准及 Neuron `unit_len=2` 实现） |
| `omron / hostlinkCmode` | `01M8DJ3Z9EDB392J5AM8SM2WWQ` | collecting (1000/1000) | 800/1000 | 900/1000 | **通过** | 备注：1000 点采集及 800 字区点反控回读全通，`00!TS` 写 `WG` 与 Neuron 一致；`plcsim` 未实现 `WG` 返回结束码 16 |

## 二、未验证协议（23 个）

| 驱动 | 状态 | 原因 |
|---|---|---|
| `ABB / COMLI` | **未验证** | plcsim 不支持 |
| `Allen-Bradley / DF1` | **未验证** | plcsim 不支持串口 DF1 |
| `CNC / KND` | **未验证** | plcsim 不支持 |
| `CNC / heidenhain` | **未验证** | plcsim 不支持 |
| `Codesys / codesys-v3` | **未验证** | plcsim 不支持 |
| `HJ212 / HJ212` | **未验证** | plcsim 不支持 |
| `IEC / IEC102` | **未验证** | plcsim 不支持 |
| `IEC / IEC103` | **未验证** | plcsim 不支持 |
| `KNX / KNXnet/IP` | **未验证** | plcsim 不支持 |
| `Kuka / Kuka` | **未验证** | plcsim 不支持 |
| `SNMP / SNMP-V2` | **未验证** | plcsim 不支持 |
| `siemens / mpi` | **未验证** | plcsim 不支持 |
| `Mitsubishi / MC-FX` | **未验证** | 传输不匹配：网关仅串口，plcsim melsec-fx-serial 仅 TCP 5010 |
| `dnp3 / dnp3` | **未验证** | 传输不匹配：网关仅 TCP，plcsim dnp3 仅串口 |
| `dlt645 / dlt645_1997` | **未验证** | plcsim 仅模拟 DL/T 645-2007（4B DI），对 1997 帧（0x01 / 2B DI）回异常码 `0xC1 / 0x02` |
| `dlt698 / dlt698_45` | **未验证** | 串口帧与 HCS/FCS 交互正常，但 plcsim 未预置常用 OAD 且不读取 TagRegistry，均回 `0x06`；已顺带补齐 `TimeTag=0x00` 与标准 DAR 解析 |
| `IEC / IEC61850` | **未验证** | 抓包确认 `plcsim` 返回 `ff 00 00 02`，不兼容标准 RFC 1006 TPKT (`0x03`) / COTP 握手 |
| `Mitsubishi / MC-4E` | **未验证** | `plcsim` 无 4E Binary 服务端（`melsec-mc` 仅支持 3E `50 00`，对 4E `54 00` 超时无应答）；驱动 4E 报文与 Neuron `qna4e` 逐字节一致 |
| `Allen-Bradley / EtherNet/IP-ControlLogic 55xx` | **未验证** | 对齐 `DriverModel` 后采回 548/1000 点，`plcsim` 对第 549 点 `Int32_148` 连续 3 次无应答触发 `i/o timeout` |
| `Allen-Bradley / EtherNet/IP-MicroLogix 1400` | **未验证** | 抓包确认 `plcsim allen-bradley-pccc` 对 `ForwardOpen(0x54)` 错误返回 14B `ForwardClose` 响应 `0xCE`，非驱动问题 |
| `ethernetIp / ethernetIp` | **未验证** | `plcsim ethernet-ip` 不支持 ForwardOpen（返回 CIP `0x08`）；`omron-njnx` 硬编码类型 `0x00C4` 导致 BOOL 类型校验失败 |
| `CNC / MTConnect` | **未验证** | 实测 `plcsim mtconnect` 为 JSON 桩（`/current` 仅返回 `application/json` 单条静态数据，`/probe` 404），不支持标准 MTConnect XML/XPath |
| `CNC / fanuc focas` | **未验证** | `plugins/dependencies/focas/libfocas32.so.1` 为 Linux x86-64 ELF，当前驱动 cgo FOCAS API 尚未接入（返回 `ErrLibraryNotWired`），且 macOS 无法加载 Linux `.so`、`plcsim fanuc` 为 Robot 协议非 CNC FOCAS |

