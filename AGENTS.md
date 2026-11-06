# AGENTS.md

本文件是仓库的统一入口：说明项目结构、常用命令和必须遵守的规范。改动代码前先读这里，再按需阅读两份细则。

| 文档 | 适用范围 |
|---|---|
| `AGENTS.md`（本文件） | 全仓总纲与红线 |
| `docs/ARCHITECTURE.md` | 目录、启动链、请求链路、数据流 |
| `docs/thing-model.md` | 物模型结构、dataType、JSON 规范 |
| `docs/things.md` | 设备（Thing）结构、凭证、在线状态与接口 |
| `docs/device-communication.md` | 设备上下行报文书信与各 type 结构（对外） |
| `docs/timeseries.md` | InfluxDB 3 时序存储、批量写入与配置 |
| `pkg/platform/platform.md` | `platform` 模块唯一事实来源（分层、响应、错误码、配置、日志、存储） |
| `ui/platform-ui/AGENTS.md` | 前端约定 |

冲突时以更具体的那份为准：平台后端看 `pkg/platform/platform.md`，前端看 `ui/platform-ui/AGENTS.md`。

## 1. 项目概览

AI-harns-platform 是工业物联网平台，Go module `harns-platform`，Go 1.24。

| 服务 | 入口 | 默认端口 | 职责 |
|---|---|---|---|
| gateway | `cmd/gateway` | 32200 | 南向采集 Modbus / OPC UA / S7，北向发布 MQTT |
| platform | `cmd/platform` | 32300 | 认证、用户/角色/组织、审计、OpenAPI |
| platform-ui | `ui/platform-ui` | 5173（dev） | React 管理后台 |

- `ui/eam-ui`、`ui/ems-ui`、`ui/gateway-ui` 目前只有 `.gitkeep`，不可构建。
- 仓库内没有 `cmd/data` 服务，前端只对接 gateway 与 platform。

## 2. 常用命令

```bash
make                      # 构建所有服务到 _output/bin
make test                 # go test -vet=off ./...
make lint                 # gofmt -l + go vet（tsdb 排除）
make fmt                  # gofmt -w cmd pkg migrations
make package              # 二进制 + UI dist + etc 配置打包
make package UI_BUILD=0   # 跳过前端构建

go build ./...            # 全量编译
go run ./cmd/gateway      # 本地启动网关
go run ./cmd/platform     # 本地启动平台

docker compose up --build # MySQL + Redis + MinIO + platform
```

前端（要求 Node ≥ 20，pnpm）：

```bash
cd ui/platform-ui
pnpm install
VITE_USE_MOCK=true pnpm dev   # 纯前端 mock
VITE_USE_MOCK=false pnpm dev  # 连接真实后端
pnpm typecheck
pnpm build
```

## 3. 目录地图

```
cmd/            进程入口（cobra：options → config → app → main）
pkg/generic/    通用 HTTP 骨架：Server、CORS、BodyLimit、Trace、options
pkg/apis/       Meta、PageQuery/PageResp、response、错误码
pkg/platform/   平台业务：identity、auth、auditapi、httpx、principal、migrate、thingmodel、things、web
pkg/gateway/    网关领域：device、registry、protocol/{modbus,opcua,s7}、runtime、storage、v1、web
pkg/audit/      共享审计模块（异步批量落库、脱敏、清理）
pkg/emqx/       共享 EMQX HTTP API 客户端（反控发布）
pkg/timeseries/ InfluxDB 3 时序存储（批量写入、database 初始化）
pkg/tsdb/       Prometheus TSDB 精简内嵌（Apache-2.0，勿按业务代码修改）
pkg/i18n/       多语言 en-US / zh-CN
pkg/logging/    slog 门面 + 日志切割
pkg/metrics/    Prometheus 指标
pkg/telemetry/  OpenTelemetry
pkg/host/       平台相关系统信息
pkg/utils/      通用工具
apis/           OpenAPI YAML（embed 后对外提供）
migrations/     SQL 迁移（go:embed，golang-migrate）
etc/            配置模板
hack/           构建/打包/校验脚本
test/           协议模拟器（modbus、modbusrtu、s7）
ui/platform-ui/ 前端
deploy/         k8s 清单
```

更细的包职责见 `docs/ARCHITECTURE.md`。

## 4. 分层与红线

- 通用层 `apis`、`generic`、`host`、`i18n`、`logging`、`tsdb`、`utils`、`version` **禁止反向依赖**模块层。
- 平台业务只能放 `pkg/platform/`，入口在 `cmd/platform/`；不得写进 `pkg/gateway` 或通用层。
- 模块内按 `model` / `store` / `service` / `server` / `v1` 单包内聚，优先同包直接调用，避免循环依赖。
- 跨模块调用由消费方定义接口，在组合根注入；禁止全局单例。
- 禁止让通用层 import 模块层；禁止在业务层直接 `c.JSON`；禁止硬编码文案、魔法数、明文密钥。
- 日志统一 `pkg/logging`，禁止 `fmt.Println`、标准库 `log`、klog / component-base。
- 新增第三方依赖需说明理由，且兼容 Go 1.24。

## 5. 后端规范摘要

**API 响应**

- 成功 `{"code":200,"message":"success","data":...}`，用 `response.WriteOK(c, data)`。
- 失败 `{"code":<业务码>,"message":"<本地化>","errors":[...]}`，用 `response.WriteError(c, status, err...)`。
- 业务错误 HTTP 一律 400，由 body `code` 区分；401 未认证、403 无权限、500 内部错误。
- 前端统一按 `code === 200 ? data : errors` 处理。

**错误码与 i18n**

- 错误码在 `pkg/apis/response/constants.go`，从 10001 起，只能末尾追加。
- 语义 key 登记在 `pkg/apis/response/errors.go` 的 `messageIDs`。
- 文案放 `pkg/i18n/locales/en-US.yaml` 与 `zh-CN.yaml`。
- 用 `response.ErrXxx(...)` 构造，不在业务层拼字符串。

**分页与校验**

- 参数用 `apis.PageQuery`（`Normalize()` 后 `Offset()` / `Limit()`），返回 `apis.PageResp[T]`。
- 用 `httpx.BindJSON` / `httpx.BindQuery`，失败返回 400，`errors[]` 含 `field` 与本地化 `message`。

**存储**

- 实体统一内嵌 `apis.Meta`（`id`/`version`/`createdUser`/`updatedUser`/`createdTime`/`updatedTime`），不要重复定义字段。
- `id` 用 ULID（列宽 32）；每次更新 `version + 1`；不设 `deleted` 列。
- 表结构变更只走 `migrations/NNNNNN_name.up.sql` / `.down.sql`。
- SQL 参数化；Redis key 带业务前缀并设 TTL。
- 审计字段取 `principal.UserID(ctx)`，禁止硬编码或留空。

**日志**

- Service 层导出方法第一行写 `defer logging.Trace("identity.Xxx", kv...)()`。
- 入口 kv 只带关键入参，禁止打印 password / authKey / token 等敏感信息。
- handler 与 store 不重复记录入口/出口。

**配置**

- 配置项定义在 `cmd/*/options/Options`（`json` tag），默认值在 `NewDefaultOptions`。
- 新增配置项要同步更新 `etc/*/config.yaml` 模板与注释。
- 优先级：CLI flag > 配置文件 > 内置默认。
- 敏感信息默认留空，不入库、不打日志。

**审计**

- 只记录 `POST/PUT/PATCH/DELETE`，GET 不记录。
- 中间件顺序：`Trace → Auth → Audit → handler`。
- 异步有界队列批量入库，body 16KB 截断，敏感键脱敏为 `***`。
- `app` 由服务绑定，platform 为 `IOT`。

## 6. 前端规范摘要

- 技术栈：React 19 + TypeScript + Vite 7 + Tailwind v4 + TanStack Query + React Router 7。
- 所有 HTTP 走 `src/lib/http.ts` 的 `request(service, path, options)`，错误统一 `ApiError`，展示用 `errorMessage(error)`。
- Query key 集中在 `src/api/query-keys.ts`，不要在组件里手写数组。
- 每个接口一个 hook，放 `src/api/hooks.ts`；设备列表 10s、详情 5s 轮询。
- `components/ui` 只放无业务原语；`components/shared` 放跨页面业务组件；页面代码放 `features/`。
- 颜色只用 `index.css` 的语义令牌（`bg-surface`、`border-border`、`text-primary` 等），不写死调色板。
- 表单用 react-hook-form + zod；字段改动要同步 `defaultFormValues`、`toFormValues`、`toPayload`。
- 空态用 `EmptyState`，错误用 `ErrorState`，加载用 `Skeleton`；不要只渲染 `null`。
- `VITE_USE_MOCK=true` 时走 `mock/`，新接口必须补 mock 路由且响应形状与 Go 一致。
- 详见 `ui/platform-ui/AGENTS.md`。

## 7. 提交前检查

```bash
gofmt -l cmd pkg migrations      # 应无输出
go build ./...
go test -vet=off ./...

cd ui/platform-ui && pnpm typecheck && pnpm build   # 改了前端时
```

- 新增错误码：同时改 `constants.go`、`messageIDs`、`locales`。
- 新增配置项：同时改 `options` 与 `etc/*/config.yaml`。
- 新增接口：同步 `apis/platform/*.yaml` 与前端 mock。
- Service 导出方法：确认已加 `logging.Trace`。
- 未触碰第 4 节红线。

## 8. 已知边界

- `pkg/tsdb` 是 Prometheus TSDB 的裁剪副本，改动风险高，除非任务明确要求，否则不要修改。
- `make lint` 与 CI 的 vet/test 未覆盖全部包（CI 不覆盖 `pkg/gateway`、`pkg/audit`、`pkg/utils`），本地提交前请自行跑 `go build ./...`。
- 仓库未配置 `cmd/data`，前端相关 proxy 配置在真实联调时需确认。
