# AI-harns-platform 管理后台

工业物联网平台的 Web 管理界面,覆盖平台服务(认证、物模型、设备、用户/角色/组织、审计)、网关服务与数据服务 API,并提供工厂 3D 可视化大屏。

## 功能

- **登录与会话**:账号密码登录(支持 RSA 加密传输)、当前用户、退出登录、401 自动跳转登录页
- **物模型**:列表/创建/编辑/删除、属性增删改(含 int/float/bool/enum/text/array/struct 等数据类型)、TSL 导入导出
- **平台设备(Thing)**:列表与在线状态、批量创建(1–100)、凭证查看/复制、图片上传、启用/停用、属性值编辑
- **用户/角色/组织**:用户 CRUD、重置密码、角色 CRUD、组织树 CRUD 与成员归属
- **审计日志**:按应用/用户/路径/方法/结果/时间筛选,查看请求与响应体
- **概览**:网关身份、CPU / 内存 / 磁盘、设备状态分布、最近设备
- **网关设备**:筛选、创建、编辑、启动 / 停止 / 重启、删除、变量反控
- **数据服务**:统计指标、KPI、测点、时序查询
- **工厂大屏** (`/screen`):独立全屏页,按协议分区展示 3D 工厂场景与设备实时变量
- **设置**:主题、运行环境、服务连通性


## 快速开始


```bash
pnpm install
VITE_USE_MOCK=true pnpm dev    # 内置 mock,无需 Go 后端
VITE_USE_MOCK=false pnpm dev   # 连接本地后端
```

开发服务器默认 `http://localhost:5173`。mock 模式可使用种子账号 `admin` / `harns@iot` 登录。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `VITE_USE_MOCK` | `false` | `true` 启用 mock,且不注册任何 proxy |
| `VITE_GATEWAY_TARGET` | `http://127.0.0.1:32200` | 开发期网关服务代理目标 |
| `VITE_DATA_TARGET` | `http://127.0.0.1:32201` | 开发期数据服务代理目标(历史遗留) |
| `VITE_PLATFORM_TARGET` | `http://127.0.0.1:32300` | 开发期平台服务代理目标 |
| `VITE_GATEWAY_BASE` | `/api/v1` | 网关服务请求基址 |
| `VITE_DATA_BASE` | `/api/v1` | 数据服务请求基址 |
| `VITE_PLATFORM_BASE` | `/api/v1` | 平台服务请求基址 |
| `VITE_MINIO_PUBLIC_BASE` | 空 | 可选的 MinIO 公共读地址,配置后设备图片按 `<base>/<key>` 预览 |

`.env.example` 默认开启 mock。生产环境可分别指定各服务的完整基址;`/api/v1/auth` 与 `/api/v1/iot` 需由反向代理转发到平台服务。

## 构建与部署

```bash
pnpm build     # tsc --noEmit + vite build -> dist/
pnpm preview   # 本地预览构建产物
```
