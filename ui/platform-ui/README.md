# AI-harns-platform 管理后台

工业边缘网关的 Web 管理界面,覆盖网关服务与数据服务现有 API,并提供工厂 3D 可视化大屏。

## 功能

- **概览**:网关身份、CPU / 内存 / 磁盘、设备状态分布、最近设备
- **设备管理**:筛选、创建、编辑、启动 / 停止 / 重启、删除、变量反控
- **数据服务**:统计指标、KPI、测点、时序查询
- **工厂大屏** (`/screen`):独立全屏页,按协议分区展示 3D 工厂场景与设备实时变量
- **设置**:主题、运行环境、服务连通性

技术栈:React 19 · TypeScript 5.9 · Vite 7 · Tailwind CSS v4 · TanStack Query 5 · React Router 7 · React Hook Form + Zod · Recharts · Three.js。

## 快速开始

要求 Node.js ≥ 20(推荐 22 LTS)与 pnpm。

```bash
pnpm install
VITE_USE_MOCK=true pnpm dev    # 内置 mock,无需 Go 后端
VITE_USE_MOCK=false pnpm dev   # 连接本地后端
```

开发服务器默认 `http://localhost:5173`。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `VITE_USE_MOCK` | `false` | `true` 启用 mock,且不注册任何 proxy |
| `VITE_GATEWAY_TARGET` | `http://127.0.0.1:32200` | 开发期网关服务代理目标 |
| `VITE_DATA_TARGET` | `http://127.0.0.1:32201` | 开发期数据服务代理目标 |
| `VITE_GATEWAY_BASE` | `/api/v1` | 网关服务请求基址 |
| `VITE_DATA_BASE` | `/api/v1` | 数据服务请求基址 |

`.env.example` 默认开启 mock。生产环境可分别指定两套服务的完整基址。

## 构建与部署

```bash
pnpm build     # tsc --noEmit + vite build -> dist/
pnpm preview   # 本地预览构建产物
```
