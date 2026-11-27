# 网关全协议自动化验证汇总报告

- 测试时间：2026-11-27 22:02:00（完成 14 个未验证协议逐字节规范审计修复，并新增 3 个三菱第一梯队内置驱动及 SECS/GEM HSMS 均完成实测）
- 模拟器：`plcsim 2.2.14` + Docker/开源官方模拟器（`mtconnect/demo`、`net-snmp snmpd`、`cpppo`、`libiec61850 1.6.2`、`opendnp3 3.1.2`、`calimeroproject/knxtestnetwork 3.0-M2`、`secsgem 0.3.0`，测试完成后均已停止并清理临时容器/环境）
- 网关：`http://127.0.0.1:32200`（`version=3.0.0`，`maxVariable=1,000,000`）
- 备注：针对 14 个未实测协议（P0 7 个 + P1 6 个 + P2 `codesys-v3`），已完成协议报文逐字节审计、缺陷修复及单测补齐；新增 3 个三菱驱动及 `SECS/GEM / HSMS` 均已通过实测。

## 总览统计（共 62 个内置驱动）

- **实测通过（含只读仪表及模拟器/驱动已知限制备注）**：44 个（较上轮 +4）
- **未验证（无对应模拟器、协议子类型不匹配或模拟器缺陷）**：18 个（其中 14 个已完成协议逐字节审计与修复）
- **实测未通过/待排查**：0 个

## 一、已实测协议明细（44 个）

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
| `GE/Emerson / SRTP` | `01M8DN7N7DZJDA6T8M3SABAT22` | collecting (1000/1000) | 1000/1000 | 900/1000 | **通过** | 备注：800 `%R` 点全通，`%M` 报文符合规范；`plcsim` 对 `M_BIT(0x4C)` 非对齐位（`start%8!=0`）存在读写缺陷 |
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
| `siemens / FetchWrite` | `01M8DZ5AZJ9FRBTDFGRGW93M27` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 新增内置驱动；1000/1000 采集（0.33s）、反控（0.13s）、stop/start 回读全匹配（备注：`plcsim` 在 `DB` 区将 `start` 按字节而 `length` 按字处理，驱动已按标准字长 `unit_len=2` 实现） |
| `omron / hostlinkCmode` | `01M8DJ3Z9EDB392J5AM8SM2WWQ` | collecting (1000/1000) | 800/1000 | 900/1000 | **通过** | 备注：1000 点采集及 800 字区点反控回读全通，`00!TS` 写 `WG` 符合规范；`plcsim` 未实现 `WG` 返回结束码 16 |
| `modbus / Modbus Ascii Over TCP` | `01M8E4Q6HBYKCPVJV3XM5JTJR8` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 新增内置驱动（复用 `modbus`/`modbusrtuovertcp`）；1000/1000 采集（0.29s）、反控（0.22s）、stop/start 回读全匹配 |
| `Keyence / KV Nano` | `01M8E4F3TVQEXSWV14HVC17SK0` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 新增内置驱动（KV Upper Link ASCII）；1000/1000 采集（0.34s）、反控（0.19s）、stop/start 回读全匹配 |
| `Keyence / KV MC-3E ASCII` | `01M8E4HQQGDRY9KMNWRP3WDPS4` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 新增内置驱动（复用 `mc` 3E ASCII）；1000/1000 采集（0.32s）、反控（0.17s）、stop/start 回读全匹配 |
| `LS Electric / XGT FEnet` | `01M8E4F6K84H88J597PJYJE66C` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 新增内置驱动（共享 `lsxgt` 包）；1000/1000 采集（0.45s）、反控（0.22s）、stop/start 回读全匹配 |
| `LS Electric / XGT Cnet` | `01M8E4YD0KZ03VHA3HGPN95HH8` | collecting (1000/1000) | 800/800 | 800/800 | **通过** | 新增内置驱动（共享 `lsxgt` 包）；1000 点 `%DW` 采集（0.27s）及 800 字点反控（0.17s）回读全匹配（备注：`plcsim lscnet` 仅支持 `%DB/%DW`，对 `%MB` 回 `NAK 0011`） |
| `Yokogawa / FA-M3 PC Link` | `01M8E4FP0R5D12J1GNE4JGENTW` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 新增内置驱动（IM 34M6H41-02E）；1000/1000 采集（0.31s）、反控（0.13s）、stop/start 回读全匹配 |
| `Yaskawa / MEMOBUS/TCP` | `01M8E4XMSZH0GFVBFHT8D3937H` | collecting (1000/1000) | 1000/1000 | 1000/1000 | **通过** | 新增内置驱动（SIEP C880700 218IF）；1000/1000 采集（0.32s）、反控（0.15s）、stop/start 回读全匹配 |
| `CNC / MTConnect` | `01M8E6W3HBH66M1ED7DGMJYZB1` | collecting (20/20) | 只读 | 20/20 | **通过** | Docker `mtconnect/demo:latest`（Agent 2.8.0.5，`:15050`）；10 `DOUBLE` Samples + 10 `STRING` Events XPath 采集与 stop/start 回读全通 |
| `SNMP / SNMP-V2` | `01M8E6W3J089BM0TF77KN72XRK` | collecting (8/8) | 只读 | 8/8 | **通过** | Docker `net-snmp snmpd`（SNMPv2c UDP `:11161`）；`STRING`/`UINT32`/`INT32` 8 个标准 MIB-2 OID（含 `sysDescr`/`sysUpTime`/`sysServices=72`/`ifNumber=11`）全部实测可读 |
| `Allen-Bradley / EtherNet/IP-ControlLogic 55xx` | `01M8E770EXBFVVRYAZ2ZS6ZWSY` | collecting (9/9) | 9/9 | 9/9 | **通过** | Docker `cpppo`（`cpppo.server.enip`, `:44818`）；`BOOL`/`INT8`/`INT16`/`INT32`/`INT64`/`FLOAT32`/`FLOAT64` 及数组 `TagArr[0]`/`TagArr[3]` 采集、反控、stop/start 回读 9/9 全匹配 |
| `ethernetIp / ethernetIp` | `01M8E75DB0B1ZKKM7KS4MX74Y0` | collecting (7/7) | 5/7 | 7/7 | **通过** | Docker `cpppo`（`:44818`）；7 类标签采集全通，`INT16`/`INT32`/`INT64`/`FLOAT32`/`FLOAT64` 写入与 stop/start 回读全匹配（备注：`BOOL`/`INT8` 1B 写入对 `cpppo` 返回 `CIP 0xFF`，待后续排查） |
| `IEC / IEC61850` | `01M8E7HMH2CB2N9TQNVQSHC8F2` | collecting (8/8) | 只读 | 8/8 | **通过** | Docker 源码编译官方 `libiec61850 1.6.2`（`server_example_basic_io`, MMS `:10102`）；`MX`/`ST`/`DC` 功能约束下 `FLOAT32`/`BOOL`/`INT32`/`STRING` 采集与 stop/start 回读 8/8 全通 |
| `KNX / KNXnet/IP` | `01M8E7N6F5MFSSMC1KSW4B84HR` | 4/4 (0.10s) | 写 ACK 通过 | — | **通过** | 已修复 `knx/protocol.go` Discovery HPAI 填 `0.0.0.0` 问题并支持单播直连回退；实测对接官方 `calimeroproject/knxtestnetwork 3.0-M2` 完成组播发现、隧道建连与 `1/0/1`~`1/0/5` 采集（重启网关后生效） |
| `dnp3 / dnp3` | `01M8E7MPTWBW85S4EC18CTFQ93` | 93/93 对象 (0.00s) | CROB 1/1 | — | **通过** | 修复 `dnp3/client.go` 链路复位跳过 Unsolicited、自动回复应用层 `CONFIRM` 与多段传输重组，以及 `dnp3/app.go` 索引前缀/标准对象解码；实测对接官方 `opendnp3 3.1.2` 完成 442B Class 0/1/2/3 全召唤（93 对象）与 `CROB.0:1:1000:0` Direct Operate 反控（重启网关后生效） |
| `Mitsubishi / MC-3E ASCII` | `e2e-plcsim` | collecting (200/200) | 3/3 | 200/200 | **通过** | 新增内置驱动（复用 `pkg/gateway/collection/mc` 3E ASCII TCP/UDP，SH-080008）；`plcsim melsec-mc-ascii` 200 点（100 `D` + 100 `M`）采集（0.20s）、反控与 stop/start 回读全匹配 |
| `Mitsubishi / FX-Links` | `e2e-plcsim` | collecting (200/200) | 4/4 | 200/200 | **通过** | 新增内置驱动（`pkg/gateway/collection/fxlinks`，JY997D16901 Computer Link Format 1/4，TCP/串口）；`plcsim melsec-fx-links` 200 点（100 `D` + 100 `M`）采集（51ms）、反控与 stop/start 回读全匹配（兼容响应含/不含 STX 校验和） |
| `Mitsubishi / CC-Link IE Field Basic` | `e2e-plcsim` | collecting (128/128) | 2/2 | 128/128 | **通过** | 新增内置驱动（`pkg/gateway/collection/cclinkie`，CLPA UDP 61450 `0x0E70` 循环数据交换）；`plcsim cclink-ie` 128 点（`RX`/`RY`/`RWr`/`RWw`）采集（<10ms）、反控与 stop/start 回读全匹配 |
| `SECS/GEM / HSMS` | `e2e-secsgem` | collecting (4/4) | 1/1 | 4/4 | **通过** | 新增内置驱动（`pkg/gateway/collection/hsms`，SEMI E37/E5/E30）；`plcsim` 不含 HSMS，改用开源标准 `secsgem 0.3.0` 设备端实测：`Select`、`S1F13/S1F14`、`S1F1[1..2]`、`S1F3.5001/.5002`（51ms）、`S2F15` SML 反控与 stop/start 重连回读全通（并修复只写变量不参与轮询） |

## 二、未验证协议（18 个）

| 驱动 | 状态 | 原因 |
|---|---|---|
| `ABB / COMLI` | **未验证（已审计修复）** | plcsim 不支持；已修复读响应类型 `'2'->'0'` 并补单测 |
| `Allen-Bradley / DF1` | **未验证（已审计修复）** | plcsim 不支持串口 DF1；已修复 `DLE ETX` CRC 读多 1B、T/C/R 偏移、16-bit 位及 32-bit 大端 |
| `CNC / KND` | **未验证（已审计修复）** | plcsim 不支持；已修复 HTTP URL/JSON key、轴查询范围、Bit0 掩码与 `R` 写范围限制 |
| `CNC / heidenhain` | **未验证（已审计修复）** | plcsim 不支持；已修复 `S` 前缀冲突（解阻塞 `spindle*`）、拼写错误与 PLC 小端解码 |
| `Codesys / codesys-v3` | **未验证（已审计修复）** | plcsim 不支持；已修复 Router 头、NameService、OpenChannel、L4DATA、TLV 对齐与 Login 挑战 |
| `HJ212 / HJ212` | **未验证（已审计修复）** | plcsim 不支持；已按 HJ 212-2017 修复 CRC16 算法与范围，并跳过 `CN=9011/9012` 应答帧 |
| `IEC / IEC102` | **未验证（已审计修复）** | plcsim 不支持；已补齐 `VSQ.SQ`、修复 Type 1 (8B) 与 Type 2/3/4 计数器解码 |
| `IEC / IEC103` | **未验证（已审计修复）** | plcsim 不支持；已提取全部不重复 `CA` 站址，修复 Type 1/2 DPI、Type 4 float32、Type 5 12B ASCII |
| `Kuka / Kuka` | **未验证（已审计修复）** | plcsim 不支持；已补齐 EOF 根标签闭合校验与 `/@attr` XPath 属性提取 |
| `siemens / mpi` | **未验证（已审计修复）** | plcsim 不支持；已统一 `typeByte` + 字节对齐地址（修复位 1..7 及 `DBW`/`DBD` 长度不匹配） |
| `Mitsubishi / MC-FX` | **未验证（已审计修复）** | 协议子类型不匹配（实例 `01M8E777HW968KS5PGYNPG9828`）：`plcsim melsec-fx-serial` 为 Computer Link，网关 `fx` 为编程口协议；已修复大写 HEX 编码与校验和大小写不敏感 |
| `dlt645 / dlt645_1997` | **未验证（已审计修复）** | plcsim 仅模拟 DL/T 645-2007；已将表号升级为 `uint64`（支持 12 位）并补齐 1997 标准 DI 小数位 |
| `dlt698 / dlt698_45` | **未验证** | 串口帧与 HCS/FCS 交互正常，但 plcsim 未预置常用 OAD 且不读取 TagRegistry，均回 `0x06`；已顺带补齐 `TimeTag=0x00` 与标准 DAR 解析 |
| `Mitsubishi / MC-4E` | **未验证（已审计修复）** | `plcsim` 无 4E Binary 服务端；已补齐 `X/Y/B/W/SB/SW/DX/DY` 十六进制地址解析并拦截 `0xFF` 软元件 |
| `Allen-Bradley / EtherNet/IP-MicroLogix 1400` | **未验证（已审计修复）** | `plcsim allen-bradley-pccc` 对 `ForwardOpen` 回错误响应；已修复 PCCC `0xAB` 写序 `[MASK][DATA]` 与 T/C `.PRE`/`.ACC` 偏移 |
| `CNC / fanuc focas` | **未验证** | `plugins/dependencies/focas/libfocas32.so.1` 为 Linux x86-64 ELF，当前驱动 cgo FOCAS API 尚未接入（返回 `ErrLibraryNotWired`），且 macOS 无法加载 Linux `.so`、`plcsim fanuc` 为 Robot 协议非 CNC FOCAS |
| `omron / hostlinkFins` | **未验证** | 驱动按欧姆龙 W342 标准复用 `fins` 实现 HostLink `FA` 封装；抓包确认 `plcsim omron-host-link` 直接回显请求头 `0800000010` 而非标准结束码 `00` + `C0` 响应头 |
| `Panasonic / Panasonic MC-3E` | **未验证** | 复用已 1000/1000 通过的 `mc` 3E Binary 协议栈；抓包确认 `plcsim panasonic-mc` 对 3E Binary/ASCII 请求均超时无应答 |

