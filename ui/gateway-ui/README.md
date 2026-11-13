# AI-harns-platform 网关管理界面

`cmd/gateway` 的 Web 管理应用,基于仓库 `docs/` 与 `apis/gateway/*.yaml` 描述的接口实现。

本包是**独立应用**,不再作为 `ui/host` 的子应用被组合;它只与 `ui/core` 共享公共代码(`@harns/core`),自行挂载路由、导航、i18n 与 mock,并直接由 `cmd/gateway` 托管(`etc/gateway/config.yaml` 的 `ui-dir: "ui/gateway-ui/dist"`)。路由部署在站点根路径,默认进入 `/overview`。

## 功能

- **概览**:网关身份与在线状态、采集/推送统计、异常通道、协议分布、CPU / 内存 / 磁盘与最近操作。
- **采集管理**:分页查看采集通道(状态、协议、周期、启用开关);创建 Modbus TCP/RTU/RTU-over-TCP、OPC UA、S7-1500、S7-200 通道;编辑连接参数、启动/停止/重启/删除;详情页查看原始连接参数、维护变量点位并对可写变量下发反控。
- **推送管理**:MQTT 连接器 CRUD 与启动/停止(`brokerUrl`、账号密码、数据/命令 Topic、QoS、Retain、Keep Alive);推送组按连接器、周期、全量或指定通道推送。
- **驱动管理**:按驱动类型分组展示内置驱动,查看连接字段、变量数据类型与地址规则;支持以 JSON Schema 注册/编辑/删除自定义驱动。
- **日志管理**:采集与连接器事件、浏览器本地操作留痕,支持级别/关键字过滤与 JSON 导出。
- **性能监控**:网关身份标识、CPU / 内存 / 磁盘使用率、客户端趋势采样与刷新。

## 快速开始

本包属于 `ui/` pnpm workspace,依赖 `@harns/core`;首次使用请先在 `ui/` 根目录 `pnpm install`。

```bash
# 在 ui/ 根目录
pnpm dev:gateway

# 或在本目录
VITE_USE_MOCK=true pnpm dev   # 内置有状态 mock
VITE_USE_MOCK=false pnpm dev  # 连接本地 gateway(32200)
```

开发服务器默认 `http://localhost:5175`,所有 `/api/v1` 请求代理到 gateway。`cmd/gateway` 的 Web 服务(默认 32200)直接托管本包构建产物,访问根路径即可打开网关管理界面。

## 数据约定

- 网关接口统一返回 `{ code, message, data }`,`src/api/gateway.ts` 的 `gatewayData` 负责解包;业务失败 HTTP 400/500,以 body `code` 区分。
- 采集通道:
  - 列表 `GET /collections` 支持 `page` / `pageSize` / `keyword` / `driverType` / `driverModel`。
  - 创建 `POST /collections` 携带 `driverType`、`driverModel`、`name`、`collectorCycleMs`、`protocolConfig` 与 `variables`;更新 `PUT /collections/:id` 不含 `variables`。
  - 变量通过 `GET|POST|DELETE /collections/:id/variables` 单独维护;反控走 `PUT /collections/:id/action`,值为字符串。
  - `POST /collections/:id/start|stop|restart`、`PUT /collections/:id/enable`(`{ enable }`)。
  - 修改连接参数或变量后需要显式重启,后端不会自动重连。
- 驱动:
  - `GET /drivers` 默认按 `driverType` 分组,`?flat=true` 返回平铺列表。
  - `GET /drivers/schema?driverModel=` 返回连接字段与变量类型,采集表单完全由 Schema 驱动。
- 推送:
  - `GET|POST /push/connectors`,`PUT|DELETE /push/connectors/:id`,`POST /push/connectors/:id/start|stop`;删除被推送组引用的连接器需 `?cascade=true`。
  - `GET|POST /push/groups`,`PUT|DELETE /push/groups/:id`;`allCollections` 为真时忽略 `collectionIds`。
- 采集状态:`collecting`、`collectingError`、`unconnected`、`emptyVariable`、`stopped`、`error`。
- mock 是**有状态**的:创建/更新/删除、启停、变量增删与反控都会修改内存数据,刷新页面重置。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `VITE_USE_MOCK` | `true` | 启用内置网关 mock,不注册 proxy |
| `VITE_GATEWAY_TARGET` | `http://127.0.0.1:32200` | 开发期网关代理目标 |
| `VITE_GATEWAY_BASE` | `/api/v1` | 网关服务请求基址 |

## 构建

```bash
pnpm build     # tsc --noEmit + vite build -> dist/
pnpm preview
```

也可以在仓库根目录执行 `make gateway-ui` 构建,或 `make gateway` 构建后启动网关。
