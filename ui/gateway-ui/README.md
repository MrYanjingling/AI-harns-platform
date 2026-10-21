# AI-harns-platform 网关管理界面

`cmd/gateway` 的 Web 管理子应用,基于仓库 `docs/` 与 `apis/gateway/*.yaml` 描述的接口实现。

## 功能

- **订阅管理**:展示 `data/<gatewayId>/v1/<deviceId>` 主题规则,按设备采集状态、协议、主题筛选;主题可一键复制。订阅关系由设备主题派生,后端没有独立的订阅 CRUD。
- **采集管理**:设备列表(状态、协议、采集周期)、创建 Modbus TCP/RTU/RTU-over-TCP、OPC UA、S7-1500 设备;编辑、删除、启动/停止/重启;详情页对可读写点位下发反控。
- **插件管理**:内置协议插件目录(Modbus / OPC UA / S7)与设备接入数量。插件目录来自前端协议元数据,网关后端未暴露插件安装接口。
- **日志管理**:采集状态事件(由设备列表快照变化产生)与本地操作日志(localStorage),支持级别/关键字过滤与 JSON 导出。文件日志仍以网关进程日志为准。
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

独立开发服务器默认 `http://localhost:5175`。主应用(`ui/host`,默认 5173)通过 `@harns/gateway-ui` 以源码方式组合本包。

## 数据约定

- 网关接口统一返回 `{ code, message, data }`,`src/api/gateway.ts` 的 `gatewayData` 负责解包;业务失败 HTTP 400/500,以 body `code` 区分。
- 设备更新使用 `ETag` / `If-Match`;缺 `If-Match` 返回 428,版本不匹配返回 412,mock 与 Go 行为一致。
- 状态动作(`start` / `stop` / `restart`)返回 202 无 body;前端成功后刷新列表与详情。
- 采集状态:`collecting`、`collectingError`、`unconnected`、`emptyVariable`、`stopped`、`error`。
- mock 是**有状态**的:创建/更新/删除会修改内存数据并递增版本,刷新页面重置。

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
