# 架构说明

本文描述仓库的运行时拓扑、目录职责、启动链、请求链路与数据流。规范约束见根目录 `AGENTS.md`、`pkg/platform/platform.md` 与 `ui/platform-ui/AGENTS.md`。

## 1. 定位与运行时拓扑

```
                    ┌──────────────────────┐
   浏览器 ─────────▶│ ui/platform-ui       │  React 19 + Vite 7
                    └──────────┬───────────┘
                               │ /api/v1/*
                ┌──────────────┴──────────────┐
                ▼                             ▼
      ┌──────────────────┐          ┌──────────────────┐
      │ gateway :32200   │          │ platform :32300  │
      │ 采集/反控/网关元数据│          │ 认证/身份/审计/OpenAPI│
      └────┬──────┬──────┘          └───┬────┬────┬────┘
           │      │                     │    │    │
     Modbus/OPCUA/S7  MQTT          MySQL  Redis MinIO
```

- **gateway**：南向通过 `pkg/gateway/protocol/` 采集 Modbus（TCP/RTU/RTU-over-TCP）、OPC UA、S7-1500，北向以 MQTT 发布数据。状态持久化在本地文件（`pkg/gateway/storage`）。
- **platform**：账号与权限、组织、审计、OpenAPI 文档；认证依赖 Redis 会话与 MinIO 上的 RSA 密钥。
- **platform-ui**：管理后台，开发期通过 Vite proxy 分流到后端；`VITE_USE_MOCK=true` 时完全走内存 mock。

## 2. 目录与包职责

```
AI-harns-platform
├── cmd/
│   ├── gateway/            gateway 入口：app / config / options
│   └── platform/           platform 入口：app / config / options
├── pkg/
│   ├── generic/            通用 HTTP 骨架（Default()、Trace、CORS、BodyLimit、Server）
│   ├── apis/               Meta、PageQuery/PageResp、ETag 常量
│   │   └── response/       Result、错误码、i18n 消息映射、WriteOK/WriteError
│   ├── platform/
│   │   ├── identity/       用户/角色/组织：model、store、service、server、v1
│   │   ├── auth/           JWT、Redis 会话、RSA 解密、验证码、Auth 中间件
│   │   ├── auditapi/       审计查询 handler、平台审计中间件、指标适配
│   │   ├── httpx/          BindJSON/BindQuery、WithOperation、Recovery、CORS
│   │   ├── principal/      请求上下文中的当前用户（UserID/UserName/FromContext）
│   │   ├── migrate/        golang-migrate + 内嵌 migrations
│   │   ├── thingmodel/     物模型与属性：model、store、service、server、tsl、v1
│   │   ├── things/         设备（Thing）：凭证、MQTT 配置、在线状态
│   │   └── web/            路由装配、health/ready、OpenAPI/Swagger
│   ├── gateway/
│   │   ├── device/         设备 CRUD、启停、反控、采集调度、MQTT 发布
│   │   ├── registry/       deviceType → v1 model / runtime device / broker 工厂
│   │   ├── protocol/       modbus、opcua、s7，各自含 model 与 runtime
│   │   ├── runtime/        设备模型、点位、序列化、校验、常量
│   │   ├── storage/        本地文件存储（按 OS 区分根目录）
│   │   ├── v1/             HTTP DTO（ModBusDevice / OpcUaDevice / S7Device）
│   │   └── web/            gateway 路由装配与优雅退出
│   ├── audit/              共享审计：Event/Recorder/Store、脱敏、清理、gin 中间件
│   ├── emqx/               共享 EMQX HTTP API 客户端（反控发布）
│   ├── tsdb/               Prometheus TSDB 裁剪副本（chunkenc、chunks、fileutil、tsdbutil）
│   ├── i18n/               语言解析中间件与 en-US / zh-CN 文案
│   ├── logging/            slog 门面、V(level)、Trace、文件切割
│   ├── metrics/            Prometheus 指标与 /metrics
│   ├── telemetry/          OpenTelemetry 初始化与 gin 中间件
│   ├── host/               主机信息（CPU/内存/磁盘，按 OS 实现）
│   ├── utils/              binutil、crcutil、randutil、uuidutil、fileutil
│   └── version/            版本信息与 -version flag
├── apis/                   OpenAPI YAML：gateway/*.yaml、platform/*.yaml（embed）
├── migrations/             6 组 up/down SQL + embed.go
├── etc/                    配置模板：gateway、platform、platform.docker
├── hack/                   build / package / lint / version 脚本
├── test/                   协议模拟器：modbus、modbusrtu、s7
├── deploy/                 k8s 清单（platform）
├── cert/                   示例 TLS 证书（仅本地开发）
└── ui/platform-ui/         前端工程（其余 ui/* 为占位）
```

## 3. 进程启动链

两个服务的启动结构一致：

```
main → app.NewXxxCmd()（cobra + pflag，DisableFlagParsing）
     → options.NewDefaultOptions()
     → ParseAndApplyConfigFile()       # CLI flag > 配置文件 > 默认值
     → options.Validate()
     → options.Config(stopCh)          # 建立外部依赖
     → web.NewServer(generic.Default(), o, cfg)
     → server.Serve()                  # 后台监听
     → 等待 SIGINT/SIGTERM → 优雅退出
```

**platform 的 `Config()`**（`cmd/platform/options/options.go:135`）依次：

1. 构造 Redis 客户端并 Ping（5s 超时），失败即启动失败。
2. 打开 MySQL 并 Ping，设置连接池（`max-open-conns` / `max-idle-conns` / `conn-max-lifetime-minutes`）。
3. `migrate: true` 时执行内嵌迁移（`pkg/platform/migrate`，版本表 `schema_migrations`）。
4. 初始化 MinIO 客户端与 OpenTelemetry。

**gateway 的 `Config()`**（`cmd/gateway/options/options.go:74`）：

1. `gateway.NewGatewayManager` 加载/生成网关身份（写入本地存储）。
2. `device.NewStore` 从存储恢复设备。
3. 连接 MQTT broker（ClientID 为 `gateway-id-<gatewayId>`），失败即启动失败。
4. `device.NewManager` 并恢复设备采集。

## 4. HTTP 请求链路

`generic.Default()` 注册的全局中间件顺序：

```
i18n → Trace → accessLogger → securityHeaders → metrics → gzip
```

随后各服务追加：

| 服务 | 额外中间件 |
|---|---|
| gateway | `generic.CORS` → `gin.Recovery` |
| platform | 可选 `telemetry` → `generic.CORS` → `httpx.Recovery` |

`/api/v1` 路由组统一挂 `generic.BodyLimit(max-body-bytes)`。

**platform 业务链路**

```
/api/v1/auth/login          audit → WithOperation → handler
/api/v1/auth/logout         audit → WithOperation → handler
/api/v1/iot/**              Auth → audit → handler
/api/v1/auth/index          Auth → handler
```

- `Trace` 负责生成/透传 `X-Trace-Id`，访问日志读取同一 trace。
- `Auth` 校验 JWT 与 Redis 会话，把完整 `UserVo` 注入 `context`（`principal`）。
- 审计中间件只处理写方法，排除 `health`、`auth/publicKey`、`auth/verificationCode`。

**platform 路由表**

| 路径 | 方法 | 说明 |
|---|---|---|
| `/api/v1/health` | GET | 存活探针，恒 200 |
| `/api/v1/ready` | GET | 依赖就绪（DB/Redis/MinIO），失败 503 |
| `/api/v1/openapi` | GET | OpenAPI 索引 |
| `/api/v1/openapi/:file` | GET | 单个规范文件 |
| `/swagger` | GET | Swagger UI |
| `/api/v1/auth/login` | POST | 登录，返回 token |
| `/api/v1/auth/logout` | POST | 登出（幂等） |
| `/api/v1/auth/verificationCode` | POST | 数学验证码 |
| `/api/v1/auth/publicKey` | GET | RSA 公钥 |
| `/api/v1/auth/index` | GET | 当前用户（可 `?refresh=true`） |
| `/api/v1/iot/users` | GET/POST | 用户分页 / 创建 |
| `/api/v1/iot/users/:id` | GET/PUT/DELETE | 用户详情 / 更新 / 删除 |
| `/api/v1/iot/users/:id/password` | PUT | 重置密码 |
| `/api/v1/iot/roles` | GET/POST | 角色分页 / 创建 |
| `/api/v1/iot/roles/:id` | GET/PUT/DELETE | 角色详情 / 更新 / 删除 |
| `/api/v1/iot/orgs` | POST | 创建组织 |
| `/api/v1/iot/orgs/:id` | GET/PUT/DELETE | 组织详情 / 更新 / 删除 |
| `/api/v1/iot/orgs/tree` | GET | 组织树 |
| `/api/v1/iot/auditLogs` | GET | 审计日志分页查询 |
| `/metrics` | GET | Prometheus 指标 |
| `/debug/pprof/*any` | GET | pprof |

**gateway 路由表**

| 路径 | 方法 | 说明 |
|---|---|---|
| `/api/v1/devices` | GET/POST | 设备列表 / 创建 |
| `/api/v1/devices/:id` | GET/PUT/PATCH/DELETE | 详情 / 全量更新 / 局部更新 / 删除 |
| `/api/v1/devices/:id/:status` | PUT | 启停（start/stop/restart） |
| `/api/v1/devices/:id/action` | PUT | 变量反控 |
| `/api/v1/gatewayMeta` | GET | 网关身份与版本（带 ETag） |
| `/api/v1/gatewayCpu|Mem|Disk` | GET | 网关资源使用 |

## 5. 网关数据流

```
HTTP 创建/启动设备
   → registry.DeviceTypeMap      按 deviceType 反序列化 v1 DTO
   → registry.DeviceTypeObjectMap 生成 runtime.Device
   → device.Manager.readyCollect
        → registry.DeviceTypeBrokerMap[deviceType](device)  建立 broker
        → broker.Collect(ctx)                              周期采集
        → results channel                                  解析后的点位
   → json.Marshal(runtime.PublishData)
   → MQTT Publish(topic = data/<gatewayId>/v1/<deviceId>, QoS 1)
```

- 设备未显式设置 `topic` 时按上述规则生成并回写设备。
- 采集异常会把设备状态置为 `CollectingError`，连接失败为 `Unconnected`，无点位为 `EmptyVariable`。
- 反控通过 `PUT /api/v1/devices/:id/action` 进入对应 protocol 的写点位逻辑（OPC UA 当前不支持反控）。
- 设备与网关元数据持久化在本地文件，按 OS 选择目录（`storage/store_{darwin,linux,windows}.go`）。

## 6. 数据模型与迁移

- 所有实体内嵌 `apis.Meta`：`id`、`version`、`createdUser`、`updatedUser`、`createdTime`、`updatedTime`。
- `id` 为 ULID（32 列宽），`version` 每次更新 +1，`ETag` 仅作参考，不做 If-Match 校验。
- 关联表 `user_roles`、`user_orgs` 不设外键，删除实体时同事务清理。
- 迁移文件 `migrations/NNNNNN_name.{up,down}.sql`，通过 `migrations/embed.go` 内嵌，`pkg/platform/migrate.Run` 用 golang-migrate 执行。
- 当前迁移：users、roles、orgs、user_relations、audit_logs、seed_admin、thing_models、things。
- `000006_seed_admin` 创建超级管理员角色与账号 `admin`，密码 `harns@iot`。首次登录后必须改密。

## 7. 配置与部署产物

| 文件 | 用途 |
|---|---|
| `etc/gateway/config.yaml` | 网关默认配置（端口、MQTT、TLS、CORS、日志） |
| `etc/platform/config.yaml` | 平台默认配置（端口、Redis、MySQL、MinIO、安全、CORS、审计） |
| `etc/platform/config.docker.yaml` | docker compose 使用的容器内配置 |

配置加载优先级：CLI flag > 配置文件 > `NewDefaultOptions`。平台默认配置路径 `/etc/platform/config.yaml`，可用 `-c` 覆盖。

构建与发布：

- `make` → `_output/bin/{gateway,platform}`。
- `make package` → `_output/release/AI-harns-platform-<version>-<os>_<arch>.zip`，内含 `gateway|platform|eam|ems` 的 `bin/etc/log` 与 `ui/<module>` 构建产物。
- `Dockerfile` 通过 `ARG SERVICE` 选择 `./cmd/${SERVICE}` 构建，最终镜像仅含单个二进制。
- `docker-compose.yml` 启动 MySQL、Redis、MinIO、platform。

## 8. 已知边界

- 仓库没有 `cmd/data`（端口 32201）服务；`ui/platform-ui` 只对接 gateway 与 platform。
- `ui/eam-ui`、`ui/ems-ui`、`ui/gateway-ui` 仅有 `.gitkeep`；`make package` 会跳过没有 `package.json` 的 UI 模块。
- `pkg/tsdb` 为 Prometheus TSDB 的裁剪副本，无业务逻辑，不应按业务代码修改。
- CI 的 vet/test 只覆盖 `cmd/...`、`pkg/platform/...`、`pkg/generic/...`、`pkg/metrics/...`、`pkg/telemetry/...`、`apis/...`；`pkg/gateway`、`pkg/audit`、`pkg/utils` 需本地自行验证。
- `cert/` 下为本地开发的示例证书。
