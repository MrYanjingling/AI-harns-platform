# Thing 规范

本文定义 AI-harns-platform 设备（Thing）的结构、凭证生成、接口与在线状态。功能条目见 `pkg/platform/platform.md` 的 F10–F13。

## 1. 概念与范围

- 一个 Thing 归属一个物模型（`thing_model_id`）。
- Thing 保存身份、凭证、MQTT Topic、静态特征与启用状态；属性值是时序数据，不存 MySQL；特征值是静态 JSON。
- 在线状态与上次上报时间由 Redis 全局活跃时间索引（`iot:thing:last-active`）提供，不落 MySQL。
- 接口前缀：`/api/v1/iot/things`。

## 2. 表结构

```sql
CREATE TABLE IF NOT EXISTS `things` (
  `id`              CHAR(32)     NOT NULL COMMENT 'ULID',
  `version`         BIGINT       NOT NULL DEFAULT 0,
  `thing_model_id`  CHAR(32)     NOT NULL COMMENT '所属物模型 ID',
  `device_name`     VARCHAR(64)  NOT NULL COMMENT '设备名称，同模型内唯一',
  `device_key`      VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT '设备 Key，全局唯一',
  `device_secret`   VARCHAR(64)  NOT NULL COMMENT '设备密钥',
  `token`           VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT 'HTTP 上报 Token，全局唯一',
  `up_topic`        VARCHAR(255) COLLATE utf8mb4_bin NOT NULL COMMENT '上行 Topic，iot/<deviceKey>/up',
  `down_topic`      VARCHAR(255) COLLATE utf8mb4_bin NOT NULL COMMENT '下行 Topic，iot/<deviceKey>/down',
  `device_desc`     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '设备描述',
  `features`        JSON         NULL COMMENT '设备静态特征值',
  `image`           VARCHAR(255) NOT NULL DEFAULT '' COMMENT '设备图片 MinIO key',
  `mqtt_client_id`  VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT '默认等于 id',
  `mqtt_username`   VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT '默认等于 deviceKey',
  `mqtt_password`   VARCHAR(64)  NOT NULL COMMENT '默认等于 deviceSecret',
  `enable`          TINYINT      NOT NULL DEFAULT 1 COMMENT '1 启用，0 禁用',
  `created_user`    VARCHAR(64)  NOT NULL DEFAULT '',
  `updated_user`    VARCHAR(64)  NOT NULL DEFAULT '',
  `created_time`    DATETIME     NOT NULL,
  `updated_time`    DATETIME     NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_things_device_key` (`device_key`),
  UNIQUE KEY `uk_things_token` (`token`),
  UNIQUE KEY `uk_things_up_topic` (`up_topic`),
  UNIQUE KEY `uk_things_down_topic` (`down_topic`),
  UNIQUE KEY `uk_things_model_name` (`thing_model_id`, `device_name`),
  UNIQUE KEY `uk_things_mqtt_client_id` (`mqtt_client_id`),
  UNIQUE KEY `uk_things_mqtt_username` (`mqtt_username`),
  KEY `idx_things_mqtt_password` (`mqtt_password`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='设备 Thing';
```

## 3. 凭证与生成规则

| 字段 | 来源 | 规则 |
|---|---|---|
| id | 后端 | ULID |
| deviceName | 入参 | 必填，≤64，同一物模型内唯一 |
| deviceKey | 入参/后端 | 可选；`^[a-zA-Z][a-zA-Z0-9_-]{3,63}$`，全局唯一；为空生成 `dk_` + 16 位随机小写字母数字，且至少包含一个字母和一个数字 |
| deviceSecret | 后端 | 32 字节 `crypto/rand` → hex 64 位；不可修改 |
| token | 后端 | 32 字节 `crypto/rand` → hex 64 位，全局唯一；不可修改 |
| upTopic | 后端 | `iot/` + deviceKey + `/up`（上行），不可覆盖、不可修改 |
| downTopic | 后端 | `iot/` + deviceKey + `/down`（下行），不可覆盖、不可修改 |
| mqtt_client_id | 后端 | 创建时等于 `id`，可修改 |
| mqtt_username | 后端 | 创建时等于 `deviceKey`，可修改 |
| mqtt_password | 后端 | 创建时等于 `deviceSecret`，可修改 |
| enable | 后端 | 创建固定 1，可通过更新接口切换 |

> `deviceKey`、`deviceSecret`、`token` 创建后不可修改。

## 4. 创建（批量）

`POST /api/v1/iot/things`

```json
{
  "thingModelId": "01k8z3m4q7x9v2b6n1c5d0e8fg",
  "things": [
    {
      "deviceName": "设备1",
      "deviceKey": "device_001",
      "deviceDesc": "一号设备"
    },
    {
      "deviceName": "设备2"
    }
  ]
}
```

带特征：

```json
{
  "thingModelId": "01k8...",
  "things": [
    {
      "deviceName": "设备1",
      "features": {
        "ratedVoltage": 380,
        "ratedPower": "5.5kW"
      },
      "image": "images/2026/11/01k8z3m4q7x9v2b6n1c5d0e8fg.png"
    }
  ]
}
```

- 单批 1~100 条，单事务，全部成功或全部回滚。
- `thingModelId` 对应的物模型必须存在。
- 批内与库内 `deviceName` 不可重复（同模型），`deviceKey` 全局不可重复。
- 未传 `deviceKey` 时自动生成。
- `features` 可选；key 必须在物模型中定义，值只校验合法 JSON，不校验 dataType 范围。
- `image` 可选；非空时必须以 `images/` 开头，来源于 `/api/v1/iot/files` 返回的 key。
- 响应返回创建结果，包含 `deviceSecret`、`token`、`mqttPassword` 和 `features`。

## 5. 查询与在线状态

### 5.1 接口

```
GET /api/v1/iot/things?page=1&pageSize=10&keyword=设备
GET /api/v1/iot/things/:id
```

- `keyword` 同时模糊匹配 `deviceName` 与 `deviceKey`。
- 按 `created_time DESC` 排序。
- 列表不返回 `deviceSecret`、`token`、`mqttPassword`、`features`、`image`；详情返回完整字段。

### 5.2 Redis 活跃时间索引

HTTP 与 MQTT 共用同一个全局结构，避免逐设备读取：

```
key: iot:thing:last-active
type: ZSET
member: thing id
score: 最后上报数据时间（Unix 毫秒）
```

写入：

| 事件 | 命令 |
|---|---|
| HTTP 上报 | `ZADD iot:thing:last-active GT CH <score> <id>` |
| MQTT 上报（含网关代传子设备） | 对每个子设备 `ZADD ... GT CH <score> <subThingId>` |
| 删除设备 | `ZREM iot:thing:last-active <id>` |

- `score = min(dataTime, now)`：数据时间超前时以服务器时间为准。
- `GT`：只允许时间前进，防止网关补传历史数据把上次上报时间改小。
- MQTT 断开不删除成员，由 TTL 判定离线。

读取：

| 场景 | 命令 |
|---|---|
| 分页列表 | `ZMSCORE iot:thing:last-active id1 id2 ... idN`（每页 1 条命令） |
| 详情 | `ZSCORE iot:thing:last-active <id>` |

在线计算：

```
online = enable == 1 && score != null && now - score <= onlineTTL
```

- `onlineTTL` 来自配置 `things.online-ttl-seconds`，默认 60 秒，可在 `etc/platform/config.yaml` 修改。
- `lastActiveTime` 直接取 score。
- 索引中不存在：`onlineStatus=0`、`lastActiveTime=null`。
- 不做 retention 清理，成员数受 thing 总数约束。
- Redis 不可用：降级为离线并记录日志，不影响列表返回。
- `enable=0`：强制离线。
- 设备影子 `iot:thing:shadow:<id>` 不参与在线判定，Hash 字段如下：
  - `reported`：已上报属性，JSON object，按属性 map 合并；
  - `desired`：期望属性，JSON object；
  - `version`：影子变更版本；
  - `timestamp`：最新已应用的 `property item.time`，不是平台接收时间。
- `TouchThingOnline` 只维护 `iot:thing:last-active` ZSET，不写影子。
- 上报时间早于影子 `timestamp` 的批次整批跳过；较新批次合并 `reported`，并把与 `reported` 相等的 `desired` 属性删除。

响应示例：

```json
{
  "id": "01k8...",
  "thingModelId": "01k8...",
  "deviceName": "设备1",
  "deviceKey": "device_001",
  "upTopic": "iot/device_001/up",
  "downTopic": "iot/device_001/down",
  "mqttClientId": "01k8...",
  "mqttUsername": "device_001",
  "enable": 1,
  "onlineStatus": 1,
  "lastActiveTime": 1760000000000
}
```

## 6. 更新

`PUT /api/v1/iot/things/:id`

```json
{
  "deviceName": "新名称",
  "deviceDesc": "新描述",
  "mqttClientId": "new-client-id",
  "mqttUsername": "new-username",
  "mqttPassword": "new-password",
  "enable": 0,
  "features": {
    "ratedVoltage": 380
  },
  "image": "images/2026/11/01k8z3m4q7x9v2b6n1c5d0e8fg.png"
}
```

- 全部字段可选，只更新传入项。
- `deviceName`：同模型内唯一。
- `mqttClientId` / `mqttUsername`：全局唯一、非空、≤64。
- `mqttPassword`：非空、≤64，不要求唯一。
- `enable`：仅 `0` 或 `1`。
- `features`：提供时整体替换，`{}` 表示清空；key 必须在物模型中定义。
- `image`：提供时更新，`""` 清空；非空时必须以 `images/` 开头。
- `deviceKey`、`deviceSecret`、`token`、`upTopic`、`downTopic` 不可修改。
- 更新后 `version` +1。

## 7. 删除

`DELETE /api/v1/iot/things?cascade=false`

```json
["01k8aaa", "01k8bbb"]
```

- body 为纯 id 数组。
- `cascade` 默认 false，非 `true` 按 false 处理。
- 幂等：不存在的 id 忽略，返回 `{"deleted": n}`。
- 当前删除 things，并 `ZREM iot:thing:last-active` 清理活跃索引；`cascade` 预留给后续影子、时序数据清理。
- `ZREM` 失败只记录日志，不影响删除结果。

## 8. 设备认证（EMQX）

EMQX 通过 HTTP 认证器连接平台，接口：

```
POST /api/v1/iot/thingsAuth
X-Things-Auth-Secret: <things.auth-secret>
Content-Type: application/json
```

- 该接口不挂平台 Auth 中间件，由 `things.auth-secret` 保护。
- `auth-secret` 为空时不校验 Header；生产环境应设置为 ≥16 字符的随机串。

请求：

```json
{
  "clientid": "01k8...",
  "username": "device_001",
  "password": "64位hex"
}
```

认证 SQL：

```sql
SELECT t.*, m.thing_model_key
FROM things t
JOIN thing_models m ON m.id = t.thing_model_id
WHERE t.mqtt_client_id = ?
  AND t.mqtt_username = ?
  AND t.mqtt_password = ?
  AND t.enable = 1
LIMIT 1
```

允许（HTTP 200）：

```json
{
  "result": "allow",
  "is_superuser": false,
  "client_attrs": {
    "thing_id": "01k8...",
    "thing_model_id": "01k8...",
    "thing_model_key": "tm_...",
    "device_key": "device_001",
    "device_name": "设备1"
  },
  "acl": [
    { "permission": "allow", "action": "all", "topic": "iot/device_001/up", "qos": [0, 1, 2], "retain": false },
    { "permission": "allow", "action": "all", "topic": "iot/device_001/down", "qos": [0, 1, 2], "retain": false }
  ]
}
```

拒绝（HTTP 401）：

```json
{ "result": "deny", "is_superuser": false }
```

- 任一凭据为空、查不到、`enable=0`、共享密钥不匹配、DB 异常 → 401 deny。
- EMQX 会把 401 当作 `ignore` 继续认证链，与现有 Java 行为一致。
- 日志不会记录 password。

## 9. EMQX 消息回调

EMQX 通过 webhook 把设备消息回调到平台：

```
POST /api/v1/iot/thingsPubMessage
X-Things-Auth-Secret: <things.auth-secret>
```

- 与 `/thingsAuth` 同一组，不经过平台用户认证。
- 只处理 `event == "message.publish"`，且 `client_attrs.device_key` 或 `client_attrs.gateway_id` 非空的事件；其余返回 204。
- 网关连接需在 `client_attrs` 中携带 `gateway_type` / `gateway_id`；`/thingsAuth` 当前只返回 thing 的属性。
- 入口把 EMQX 事件转换为平台 `DeviceMessage`，再交给 Router。

### 9.1 处理链路

```
EMQX 事件
  -> 校验 event / device_key
  -> 按 topic 分类
  -> 转换 DeviceMessage
  -> 刷新在线状态
  -> Kafka 关闭：Router.Handle
  -> Kafka 开启：Kafka(topic, key=deviceKey 或 gatewayId) -> consumer 批量消费 -> Router.Handle
```

`DeviceMessage`：

```go
type DeviceMessage struct {
    Kind          MessageKind
    Topic         string
    DeviceKey     string
    ThingID       string
    ThingModelID  string
    ThingModelKey string
    DeviceName    string
    GatewayType   string
    GatewayID     string
    Payload       string
    Timestamp     int64
    MessageID     string
    Source        string
    QoS           int
}
```

Handler 与业务 Processor 只接收 `DeviceMessage`，不再包含 EMQX 字段。

### 9.2 消息分类

分类由 payload 的 `type` 字段决定，与 topic 无关，且 `type` 必填：

```
property       -> property
event          -> event
command_reply  -> command_reply
command        -> 下行类型，入站忽略
缺失/其他      -> 忽略并记录日志
```

只使用两个 topic：

```
iot/<deviceKey>/up    设备 -> 平台
iot/<deviceKey>/down  平台 -> 设备
```

业务处理由 `PropertyProcessor` / `EventProcessor` / `CommandReplyProcessor` 分别实现；
生产环境使用 `DefaultPropertyProcessor`，`NewDefaultMessageRouter()` 仅用于测试和兜底占位。

## 10. HTTP 设备数据上报

设备使用创建设备时生成的 token 上报属性数据：

```
POST /api/v1/iot/thingsData
X-Things-Token: <token>
Content-Type: application/json

{ "temperature": 25.6 }
```

- 不经过平台用户认证，也不使用 EMQX 共享密钥。
- Body 直接是原始 JSON，按 payload `type` 分类：`property`、`event`、`command_reply`；`type` 缺失、`command` 或未知类型返回 400。
- 与 MQTT 路径共用同一个 `DeviceMessage` 和 Router。

### 10.1 Token 查询与缓存

```
key   = iot:thing:token:<sha256(token) hex>
value = JSON(ThingIdentity)
TTL   = 30 分钟，命中时刷新
```

- 先查 Redis，未命中再查 MySQL（`token=? AND enable=1`），然后回填。
- 命中时使用 `GETEX` 刷新 TTL，实现滑动过期。
- 缓存只包含非敏感字段：id、thing model、deviceKey、deviceName、mqtt 字段、两个 topic。
- 更新设备、删除设备后直接删除对应 key，下次重新查库。
- Redis 异常时回落 MySQL。

### 10.2 处理链路

```
token -> ThingIdentity -> 校验 JSON -> 刷新在线状态
  -> 构造 DeviceMessage(Kind=取自 payload type，Source=http, Topic=upTopic)
  -> Kafka 关闭：Router.Handle
  -> Kafka 开启：Kafka(topic, key=deviceKey 或 gatewayId) -> consumer -> Router.Handle
```

响应使用平台风格：

```json
{ "code": 200, "message": "success" }
```

| 场景 | HTTP | code |
|---|---|---|
| token 为空/无效 | 401 | 10031 |
| body 空或非法 JSON | 400 | 10001 |
| 处理/Kafka 失败 | 500 | 10017 |

## 11. 错误码

| 业务码 | 语义 key | 说明 |
|---|---|---|
| 10048 | `error.thing_not_found` | 设备不存在 |
| 10049 | `error.thing_device_name_exists` | 设备名称已存在 |
| 10050 | `error.thing_device_key_exists` | 设备 Key 已存在 |
| 10051 | `error.thing_topic_exists` | Topic 冲突（预留，topic 当前不可修改） |
| 10052 | `error.thing_mqtt_client_id_exists` | MQTT clientId 已存在 |
| 10053 | `error.thing_mqtt_username_exists` | MQTT username 已存在 |
| 10054 | `error.thing_invalid` | 设备校验失败 |

## 12. 相关文件

- 功能说明：`pkg/platform/platform.md` F10–F13
- 代码：`pkg/platform/things/`
- 迁移：`migrations/000008_create_things.{up,down}.sql`
- OpenAPI：`apis/platform/things.yaml`
