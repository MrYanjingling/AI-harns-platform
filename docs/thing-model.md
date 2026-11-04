# 物模型规范

本文定义 AI-harns-platform 物模型（Thing Model）的结构、数据类型、校验规则与导入导出格式。功能条目见 `pkg/platform/platform.md` 的 F6–F9。

## 1. 概念与范围

- 物模型描述一个设备拥有的属性和特征。
- 属性是动态数据通道，特征是静态属性定义。
- 本版本不包含事件、服务、产品、品类。
- 物模型通过 `thingModelKey` 在全局唯一标识，`name` 全局唯一。
- 属性以 `identifier` 在物模型内唯一标识，且创建后不可修改。
- 后端接口前缀：`/api/v1/iot/thingModels`。

### 1.1 存储与关联

物模型与属性分两张表存放：

| 表 | 作用 | 关联 |
|---|---|---|
| `thing_models` | 物模型主表 | — |
| `thing_model_properties` | 属性表 | `thing_model_id` → `thing_models.id` |

- 属性通过 `thing_model_id`（物模型的 ULID 主键）关联，**不使用 `thing_model_key`**。key 是面向用户的业务标识，仅用于展示、查询和导入导出；关系键统一用主键 id，与 `user_roles`、`user_orgs` 等关联表保持一致。
- `thing_model_id` 是内部列，由 URL 中的 `:id` 决定，不出现在请求体或响应 Vo 中，避免 path 与 body 不一致。
- `thing_model_properties` 以 `id`（ULID）为主键；`(thing_model_id, identifier)` 与 `(thing_model_id, name)` 为模型内唯一索引，因此不同物模型可以使用相同的 identifier/name。
- 特征定义不单独建表，存在 `thing_models.features` JSON 列。
- 删除物模型时在同一事务内按 `thing_model_id` 删除全部属性。
- 删除接口支持 `?cascade=false`（默认 false）。该参数为预留项，后续与 thing 联动时生效；当前属性始终随物模型删除，非 `true` 的取值一律按 false 处理。

## 2. JSON 结构

```json
{
  "thingModelKey": "tm_01k8z3m4q7x9v2b6n1c5d0e8fg",
  "name": "温控器",
  "description": "示例物模型",
  "properties": [
    {
      "identifier": "temperature",
      "name": "温度",
      "dataType": {
        "type": "double",
        "unit": "℃",
        "unitName": "摄氏度",
        "min": "0",
        "max": "100",
        "step": "0.1"
      },
      "rwFlag": "r",
      "required": true,
      "description": "环境温度",
      "custom": {},
      "sort": 0
    },
    {
      "identifier": "mode",
      "name": "模式",
      "dataType": {
        "type": "enum",
        "enumSpec": {
          "values": [
            { "value": 0, "desc": "自动" },
            { "value": 1, "desc": "手动" }
          ]
        }
      },
      "rwFlag": "rw",
      "required": false,
      "sort": 1
    },
    {
      "identifier": "history",
      "name": "历史值",
      "dataType": {
        "type": "array",
        "arraySpec": {
          "size": "10",
          "element": { "type": "double", "unit": "℃" }
        }
      },
      "rwFlag": "r",
      "sort": 2
    },
    {
      "identifier": "config",
      "name": "配置",
      "dataType": {
        "type": "struct",
        "structSpec": {
          "fields": [
            {
              "identifier": "high",
              "name": "上限",
              "dataType": { "type": "double", "min": "0", "max": "100" }
            }
          ]
        }
      },
      "rwFlag": "rw",
      "sort": 3
    }
  ],
  "features": [
    {
      "identifier": "ratedVoltage",
      "name": "额定电压",
      "dataType": { "type": "double", "unit": "V" },
      "description": "设备额定电压",
      "sort": 0
    }
  ]
}
```

字段说明：

| 字段 | 类型 | 说明 |
|---|---|---|
| thingModelKey | string | 全局唯一；创建时可传，不传由后端生成；创建后不可修改 |
| name | string | 必填，全局唯一，≤64 |
| description | string | 可空，≤255 |
| properties | array | 属性列表，见下 |
| features | array | 特征定义列表，见下 |

属性字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| identifier | string | 必填，同模型内唯一，创建后不可修改 |
| name | string | 必填，同模型内唯一，≤64 |
| dataType | object | 必填，见第 3 节 |
| rwFlag | string | `r` / `w` / `rw` |
| required | bool | 是否必填，默认 false |
| description | string | 可空，≤255 |
| custom | object | 自定义信息，可空 |
| sort | int | 展示顺序，默认 0 |

特征字段（纯静态，无 `rwFlag` / `required`）：

| 字段 | 类型 | 说明 |
|---|---|---|
| identifier | string | 必填，features 内唯一，可与属性 identifier 重名 |
| name | string | 必填，features 内唯一，≤64 |
| dataType | object | 必填，见第 3 节 |
| description | string | 可空，≤255 |
| custom | object | 自定义信息，可空 |
| sort | int | 展示顺序，默认 0 |

> 系统字段（`id`、`version`、`createdUser` 等）由后端维护，导入时忽略。

## 3. dataType

`type` 是唯一判别字段，复杂规格各自独立：

```go
type DataTypeDetail struct {
    Type DataType `json:"type"` // int|long|float|double|bool|enum|text|time|array|struct

    Unit, UnitName, Min, Max, Step *string
    TrueValue, FalseValue          *string
    Length                         *string

    EnumSpec   *EnumSpec   `json:"enumSpec,omitempty"`
    ArraySpec  *ArraySpec  `json:"arraySpec,omitempty"`
    StructSpec *StructSpec `json:"structSpec,omitempty"`
}

type EnumSpec   struct { Values []EnumValue `json:"values"` }
type ArraySpec  struct { Size *string `json:"size,omitempty"`; Element DataTypeDetail `json:"element"` }
type StructSpec struct { Fields []StructField `json:"fields"` }
type StructField struct { Identifier, Name string; DataType DataTypeDetail `json:"dataType"` }
type EnumValue   struct { Value int `json:"value"`; Desc string `json:"desc"` }
```

| type | 允许字段 | 必填校验 |
|---|---|---|
| int / long / float / double | unit、unitName、min、max、step | min ≤ max（可解析时） |
| bool | trueValue、falseValue | 无 |
| enum | enumSpec.values[] | values 非空，value 不重复 |
| text | length | 正整数 |
| time | 无 | 无 |
| array | arraySpec.element、arraySpec.size | element 必填；size 可选，1~512 |
| struct | structSpec.fields[] | fields 非空，field identifier 唯一 |

**宽松忽略**：与 `type` 不匹配的字段在解析时直接丢弃，落库只保存归一化后的结构。例如 `type=int` 携带 `enumSpec` 时，`enumSpec` 会被清空；必填项仍严格校验。

## 4. 校验规则

属性级：

- `identifier`：`^[A-Za-z][A-Za-z0-9_]*$`，≤64，不能是 `time`、`device_id`，不能是 SQL 关键字，同模型内唯一。
- `name`：非空，必须以字母开头，≤64，同模型内唯一。
- `rwFlag`：`r`、`w`、`rw`。
- `sort`：非负整数。

错误以字段路径返回：

```json
{
  "code": 10045,
  "message": "物模型校验失败。",
  "errors": [
    {
      "code": 10045,
      "field": "properties[0].identifier",
      "message": "属性标识符只能由字母、数字和下划线组成，且必须以字母开头。"
    },
    {
      "code": 10045,
      "field": "properties[2].dataType.arraySpec.size",
      "message": "数组元素个数必须是 1~512 的整数。"
    }
  ]
}
```

字段路径格式：

| 场景 | 路径示例 |
|---|---|
| 物模型名称 | `name` |
| 物模型 key | `thingModelKey` |
| 属性字段 | `properties[0].name` |
| 数据类型 | `properties[1].dataType.type` |
| 枚举规格 | `properties[1].dataType.enumSpec.values` |
| 数组规格 | `properties[2].dataType.arraySpec.element.type` |
| 结构体字段 | `properties[3].dataType.structSpec.fields[1].dataType.min` |

## 5. 导入导出

| 操作 | 接口 | 说明 |
|---|---|---|
| 导出 | `GET /api/v1/iot/thingModels/:id/tsl` | 导出 JSON，含 `thingModelKey` |
| 导入 | `PUT /api/v1/iot/thingModels/:id/tsl` | 仅 JSON，更新名称/描述并整体替换属性；features 提供时整体替换 |

### 5.1 导入结构

```json
{
  "thingModelKey": "tm_01k8z3m4q7x9v2b6n1c5d0e8fg",
  "name": "温控器",
  "description": "示例物模型",
  "properties": [
    {
      "identifier": "temperature",
      "name": "温度",
      "rwFlag": "r",
      "required": true,
      "dataType": { "type": "double", "unit": "℃", "min": "0", "max": "100" }
    }
  ],
  "features": [
    {
      "identifier": "ratedVoltage",
      "name": "额定电压",
      "dataType": { "type": "double", "unit": "V" }
    }
  ]
}
```

| 字段 | 必填 | 行为 |
|---|---|---|
| `thingModelKey` | 否 | 非空时必须与当前物模型一致；key 不可修改 |
| `name` | 否 | 非空时更新，需满足名称规则且全局唯一；省略则保持不变 |
| `description` | 否 | 提供时更新，`""` 表示清空；省略则保持不变 |
| `properties` | 是 | `[]` 表示清空全部属性；否则整体替换 |
| `features` | 否 | 省略则保留原特征；提供时整体替换，`[]` 表示清空 |

- 导入仅接受 JSON。
- `properties` 缺失 → 字段级错误 `properties`。
- JSON 语法错误 → `10046 error.thing_model_import_invalid`。
- 语义校验失败 → `10045 error.thing_model_invalid`，`errors[].field` 指出具体字段。
- 名称、描述、属性、特征在同一事务内更新；失败整体回滚。

## 6. 错误码

| 业务码 | 语义 key | 说明 |
|---|---|---|
| 10042 | `error.thing_model_not_found` | 物模型不存在 |
| 10043 | `error.thing_model_name_exists` | 物模型名称已存在 |
| 10044 | `error.thing_model_property_not_found` | 属性不存在 |
| 10045 | `error.thing_model_invalid` | 物模型校验失败 |
| 10046 | `error.thing_model_import_invalid` | 导入内容无法解析 |
| 10047 | `error.thing_model_key_exists` | 物模型 Key 已存在 |

## 7. 相关文件

- 功能说明：`pkg/platform/platform.md` F6–F9
- 代码：`pkg/platform/thingmodel/`
- 迁移：`migrations/000007_create_thing_models.{up,down}.sql`
- OpenAPI：`apis/platform/thingmodel.yaml`
