# Timeseries 存储规范

本文定义设备时序数据在 InfluxDB 3 Core 中的存储结构、database 划分、批量写入链路与配置。

## 1. 概述

- 数据库：InfluxDB 3 Core
- 接入方式：InfluxDB 3 官方 HTTP API（`/api/v3/configure/database`、`/api/v3/write_lp`）
- 写入方式：schemaless line protocol，表在首次写入时自动创建
- 不引入官方 Go 客户端：当前客户端依赖要求 Go 1.25，本项目保持 Go 1.24，因此直接使用官方 HTTP API
- 代码位置：`pkg/timeseries/`

## 2. 两个 database

InfluxDB 3 Core 的 retention 是 **database 级**，创建后不可修改，因此按保留期拆成两个库：

| database | 用途 | retention 配置 |
|---|---|---|
| `harns_timeseries` | 属性、事件、控制记录 | `timeseries_data_retention`，默认 `730d` |
| `harns_timeseries_log` | 属性日志、事件日志 | `timeseries_log_retention`，默认 `1d` |

启动时通过管理 API 创建：

```http
POST /api/v3/configure/database
Authorization: Bearer <token>
Content-Type: application/json

{ "db": "harns_timeseries", "retention_period": "730d" }
```

- database 已存在时记录 warning，retention 以服务端为准。
- Core 不支持修改 retention；需要变更只能新建 database 并迁移。
- 不显式建表，首次写入自动创建。

## 3. 表结构

设备类表列名使用 snake_case，包含 `model_id`、`device_id`、`device_key` 三个 tag；`gateway_property_log` 使用 `gateway_type`、`gateway_id`；时间列 `time` 使用毫秒精度。

### 3.1 `device_property_<modelKey>`

| 角色 | 列 |
|---|---|
| tags | `model_id`, `device_id`, `device_key` |
| time | `time` |
| fields | 动态属性列，键为物模型属性 identifier |

- 每个属性上报 item 写一条 point。
- 数组/结构体属性存 JSON 字符串。
- 保留：`harns_timeseries`

### 3.2 `device_property_log_<modelKey>`

| 角色 | 列 |
|---|---|
| tags | `model_id`, `device_id`, `device_key` |
| time | `time` |
| fields | `success`, `payload`, `error_msg` |

- 按请求一条，`payload` 为本次请求原始 JSON。
- 保留：`harns_timeseries_log`

### 3.2.1 `gateway_property_log`

| 角色 | 列 |
|---|---|
| tags | `gateway_type`, `gateway_id` |
| time | `time` |
| fields | `success`, `payload`, `error_msg` |

- 网关属性报文每次请求一条。
- `payload` 为整条网关请求原始 JSON。
- 保留：`harns_timeseries_log`

### 3.3 `device_event_<modelKey>`

| 角色 | 列 |
|---|---|
| tags | `model_id`, `device_id`, `device_key`, `event_identify` |
| time | `time` |
| fields | `data` (JSON 字符串) |

- 每个事件上报 item 写一条 point。
- 保留：`harns_timeseries`

### 3.4 `device_event_log_<modelKey>`

| 角色 | 列 |
|---|---|
| tags | `model_id`, `device_id`, `device_key` |
| time | `time` |
| fields | `success`, `payload`, `error_msg` |

- 按请求一条。
- 保留：`harns_timeseries_log`

### 3.5 `device_command_record_<modelKey>`

| 角色 | 列 |
|---|---|
| tags | `model_id`, `device_id`, `device_key` |
| time | `time` |
| fields | `success`, `command_payload`, `command_reply_payload`, `message_id` |

- 一次请求一条记录。
- `command_payload` / `command_reply_payload` 为原始 JSON 字符串。
- 保留：`harns_timeseries`

## 4. 类型映射

InfluxDB 3 没有 int32，整数统一为 64 位。

| 业务类型 | InfluxDB 3 |
|---|---|
| int / int32 | `int64` |
| long / int64 | `int64` |
| float / double | `float64` |
| bool | `bool` |
| text / time | `string` |
| enum | `int64` 或 `string`，按物模型定义 |
| array / struct | JSON `string` |

## 5. 写入链路

```
Processor
  → Service.WriteProperty / WriteEvent / ...
      → 组装 timeseries.Point
      → Store.EnqueueData / EnqueueLog
          → batcher(channel)
              → 满足阈值后 WritePoints
```

Service 一次调用对应一台设备；Store 会把多次调用的 point 合并成一次写入。

## 6. 批量与背压

两个独立 batcher：data 和 log。

| 参数 | 默认 | 说明 |
|---|---|---|
| `batch-size` | 10000 | point 数量阈值 |
| `flush-interval-ms` | 3000 | 时间阈值 |
| `queue-size` | 20000 | channel 容量 |

- 任一阈值满足即 flush。
- 队列满时阻塞等待，直到有空间或请求 ctx 取消。
- 写入失败只记录日志，不重试。
- 服务关闭时关闭 channel、flush 剩余数据并等待 goroutine 退出。

## 7. 配置

```yaml
timeseries:
  enabled: false
  url: "http://127.0.0.1:8181"
  token: ""
  database: "harns_timeseries"
  log-database: "harns_timeseries_log"
  timeseries_data_retention: "730d"
  timeseries_log_retention: "1d"
  batch-size: 10000
  flush-interval-ms: 3000
  queue-size: 20000
  write-timeout-ms: 10000
```

- `enabled=true` 时启动阶段会创建 database 并启动批量写入 goroutine。
- `token` 启用时必填。

## 8. 包结构

```
pkg/timeseries/
├── config.go      连接、database、retention、批量参数
├── model.go       常量、表名函数、Point 结构
├── batcher.go     chan + goroutine 批量管道
├── store.go       InfluxDB 客户端、EnsureDatabase、Enqueue
├── service.go     业务组装与写入
└── *_test.go
```

## 9. 注意事项

- InfluxDB 3 Core 的 retention 不可修改。
- 同名字段在同一表中类型必须一致；物模型字段类型变更需要谨慎。
- 日志表只保留 1 天，高吞吐场景需要评估存储与写入能力。
- `time` 使用数据时间，毫秒精度。
- 表名依赖 `modelKey`，`modelKey` 变更会变成新表。
