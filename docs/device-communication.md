# 设备通信报文规范

本文定义设备与平台之间的上下行报文结构，包括设备上报的 property、event、command_reply，以及平台下发的 command。

## 1. 通用信封

所有上报数据使用同一层信封：

```json
{
  "type": "property",
  "payload": []
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| type | string | 是 | 消息类型，见第 2 节；缺失时平台不处理 |
| payload | array/object | 是 | 具体类型的数据，由 type 决定 |

## 2. type 枚举

| type | 方向 | 状态 | 说明 |
|---|---|---|---|
| `property` | 设备 → 平台 | 已实现 | 属性数据上报 |
| `event` | 设备 → 平台 | 已实现 | 事件上报 |
| `command_reply` | 设备 → 平台 | 已实现 | 命令执行结果 |
| `command` | 平台 → 设备 | 下行 | 命令下发，结构已定义，下发待实现 |

- `type` 缺失、为空或未知：平台记录日志后不处理。
- 设备上报中出现 `command`：平台拒绝。

## 3. property 上报

```json
{
  "type": "property",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "time": 1773827321000,
      "property": {
        "temperature": 25.6,
        "status": "running",
        "running": true
      }
    }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| collectionId | string | 否 | 采集通道 ID，预留字段，当前允许为空 |
| time | int64 | 否 | 数据采集时间，Unix 毫秒；缺失或 <= 0 时使用服务器当前时间 |
| property | object | 是 | 属性键值对，键为物模型中的属性 identifier，不能为空 |

`property` 的值支持 JSON 原始类型：

- number
- string
- boolean
- null
- object / array（由物模型结构决定）

平台按 JSON 原始值接收，后续根据物模型的 `dataType` 进行解析和校验：

- `required` 属性必须出现且不能为 `null`，否则整条请求失败。
- 物模型中不存在的属性 identifier 直接忽略。
- 值支持字符串形式（如 `"1"`、`"25.5"`）和 JSON 原生类型。
- `int`/`long` 校验整数和 `min`/`max`；`float`/`double` 校验数值和 `min`/`max`。
- `bool` 支持 `true/false`、`1/0`、`"1"/"0"` 以及物模型的 `trueValue`/`falseValue`。
- `enum` 校验值在 `enumSpec.values` 中。
- `text` 按字符数校验 `length`；`time` 按字符串存储，不校验格式。
- `array` 的 `size` 为最大元素个数，并递归校验元素。
- `struct` 只处理已定义字段，未知字段忽略。
- 校验失败写 `device_property_log_<modelKey>`，`success=false`，不写时序数据。
- 校验通过后先更新设备影子，再提交时序服务，最后写 `success=true`。
- 设备影子 `reported` 按属性合并；`item.time` 早于影子 `timestamp` 的批次整批跳过。
- 如果 `desired` 中的属性值与 `reported` 相等，则从 `desired` 中删除。
- 影子更新后，每个 property item 先判断是否存在告警和规则条件，只对命中的目标异步分发；Kafka 开启时分别投递到告警/规则触发 topic，否则调用对应本地 handler。

## 4. event 上报

```json
{
  "type": "event",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "time": 1773827321000,
      "eventIdentify": {
        "alarmCode": 1001,
        "level": "high",
        "detail": {
          "temperature": 88.5
        }
      }
    }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| collectionId | string | 否 | 采集通道 ID，预留字段，当前允许为空 |
| time | int64 | 是 | 事件发生时间，Unix 毫秒 |
| eventIdentify | object | 是 | 事件标识数据，非空；键与物模型事件定义对应，不是固定结构 |

- `eventIdentify` 的字段由物模型的事件定义决定，平台本期只做结构解析。
- 值支持 JSON 原始类型，允许嵌套对象/数组。
- 每个上报项的 `eventIdentify` 至少包含一个字段。

## 5. command（下行）

```json
{
  "type": "command",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "messageId": "7d2c3b4a-1234-5678-9abc-def012345678",
      "property": {
        "a": 1
      }
    }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| collectionId | string | 否 | 采集通道 ID，预留字段，当前允许为空 |
| messageId | string | 是 | 命令唯一标识，用于匹配 command_reply |
| property | object | 是 | 非空；键由物模型服务定义 |

- 下行方向：平台 → 设备，走 `down` topic。
- `messageId` 由平台生成并保证唯一。
- 本期只定义结构，不实现下发逻辑。

## 6. command_reply（上行）

```json
{
  "type": "command_reply",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "messageId": "7d2c3b4a-1234-5678-9abc-def012345678",
      "property": {
        "a": 1
      },
      "result": true,
      "time": 1773232324230
    }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| collectionId | string | 否 | 采集通道 ID，预留字段，当前允许为空 |
| messageId | string | 是 | 对应 command 的 messageId |
| property | object | 是 | 非空；执行结果附带的数据 |
| result | boolean | 是 | 执行成功/失败；必须出现 |
| time | int64 | 是 | 执行时间，Unix 毫秒 |

- 上行方向：设备 → 平台，走 `up` topic。
- `messageId` 必须与下发的 command 一致。
- `result` 缺失视为非法报文。

## 7. 批量与断点续传

- `payload` 是数组，可一次上报多条。
- 建议按 `time` 升序排列。
- 设备断点续传时允许重发旧批次。
- 同一设备、同一时间的数据可能重复；平台不做去重，时序数据库按时间覆盖。
- 不限制业务条数，但受请求体大小限制（`max-body-bytes`，默认 1MB）。

## 8. 上报入口

### MQTT

设备发布到上行 topic：

```
iot/<deviceKey>/up
```

EMQX 通过 webhook 回调平台，平台解析 payload。

### HTTP

```
POST /api/v1/iot/thingsData
X-Things-Token: <创建设备时生成的 token>
Content-Type: application/json
```

Body 即第 1 节的通用信封。

## 9. 完整示例

```json
{
  "type": "property",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "time": 1773827321000,
      "property": {
        "a": 1,
        "b": "string",
        "d": 12.3
      }
    },
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "time": 1773827322000,
      "property": {
        "a": 2,
        "b": "string",
        "d": 12.4
      }
    }
  ]
}
```

## 10. 错误与日志

| 场景 | 行为 |
|---|---|
| `type` 缺失 | 不处理，记录上报日志 |
| `type` 未知 | 不处理，记录日志 |
| `type=command` 出现在上报中 | 拒绝，记录日志 |
| payload 不是合法 JSON | HTTP 400；MQTT 忽略并记录日志 |
| property payload 不是数组 | 处理失败并记录日志 |

## 11. 扩展

当前已定义 `property`、`event`、`command_reply`、`command` 四种结构。

后续新增类型时：

1. 新增对应的 payload 结构；
2. 实现统一的 `GetType()` 接口；
3. 在解码入口按 `type` 分发。

现有结构不会变化。
