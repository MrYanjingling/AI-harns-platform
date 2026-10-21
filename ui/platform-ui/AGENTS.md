# ui/platform-ui/AGENTS.md — 前端工程约定

本文件只讲**前端工程约定**。业务规则（页面行为、字段约束、状态语义、接口契约）以 [`BUSINESS.md`](./BUSINESS.md) 为唯一事实来源：

- 实现需求前先读 `BUSINESS.md` 对应章节；
- 业务变化时，**先改 `BUSINESS.md`，再改代码与 mock**；
- 实现与文档冲突时，以文档为准，并同步修正代码。

## 技术栈

| 层 | 选型 |
|---|---|
| 语言 | TypeScript 5.9，`strict` 模式 |
| 运行时 | Node.js ≥ 20（推荐 22 LTS） |
| 包管理 | **pnpm**（`pnpm-lock.yaml`） |
| 框架 | React 19 + React Router 7 |
| 构建 | Vite 7 + `@vitejs/plugin-react` |
| 样式 | Tailwind CSS v4（CSS-first，`@tailwindcss/vite`） |
| 服务端状态 | TanStack Query v5 |
| 表单 | React Hook Form 7 + Zod 3 + `@hookform/resolvers` |
| 图表 | Recharts 3 |
| UI 辅助 | lucide-react、sonner、clsx、tailwind-merge |

- 统一使用 pnpm：`pnpm install`、`pnpm dev`、`pnpm typecheck`、`pnpm build`、`pnpm preview`。不要混用 npm/yarn，也不要提交其他包管理器的锁文件。
- 不要擅自升级大版本或引入新的状态/样式/请求库；确需引入时先说明理由与替代方案。

## 目录

```
src/
├── main.tsx / router.tsx / index.css
├── api/          # types.ts（后端契约）、endpoints、query-keys、hooks
├── lib/          # http、format、constants、cn、query-client
├── providers/    # ThemeProvider
├── components/
│   ├── layout/   # AppShell + Sidebar
│   ├── ui/       # 无业务语义的原语：Button/Card/Field/Dialog/Table/Badge…
│   ├── charts/   # GaugeRing 等
│   └── shared/   # 跨页面复用的业务组件：PageHeader/StatCard/StatusBadge…
└── features/     # 按页面域组织：dashboard/devices/metrics/kpi/tags/timeseries/settings
mock/
├── index.ts      # Vite 中间件，内存态 API
└── fixtures.ts   # 种子数据
```

## 数据请求

- 所有 HTTP 走 `src/lib/http.ts` 的 `request(service, path, options)`：
  - `service` 为 `'gateway'` 或 `'data'`，基址默认都是 `/api/v1`，由 Vite proxy 或生产反向代理分流。
  - 错误统一抛 `ApiError`，展示时用 `errorMessage(error)`。
  - 更新/删除设备必须传 `ifMatch`（来自 `DeviceSummary.eTag` / `DeviceDetail.eTag`）；启停与反控接口按后端契约不要求。
- 后端契约类型集中在 `src/api/types.ts`；改类型前核对 `apis/` 与 Go 源码，不在组件里临时声明接口。
- Query key 集中定义在 `src/api/query-keys.ts`；组件中禁止手写 key 数组。
- 每个接口一个函数（`src/api/*.ts`），每个接口一个 hook（`src/api/hooks.ts`）：
  - 设备列表/网关资源 10s 轮询，设备详情 5s 轮询；其余查询按需配置。
  - 写设备成功后失效设备列表与相关详情缓存。
  - 412/404 冲突时先失效缓存，让用户看到最新版本，不静默覆盖。
- 数据服务响应统一 `{ data, total }`，解包逻辑集中在 `src/api/data.ts`。

## 组件

- 依赖方向：`features → components/shared → components/ui`，以及 `features/api/lib`；禁止 `features` 之间互相导入。
- `components/ui` 只放通用原语，不含业务概念；跨两个以上页面复用的业务组件放 `components/shared`；只服务单一页面的放对应 `features`。
- 复用现有视觉语义：卡片 `rounded-2xl border border-border bg-surface`，主色 `text-primary`，弱文本 `text-muted/faint`，数字用 `tabular`。
- 颜色只引用 `index.css` 中的语义令牌（`bg-surface`、`border-border`、`text-danger` 等）；不写死 zinc/emerald 调色板（状态徽标等既有例外除外）。
- 弹窗使用原生 `<dialog>`（`components/ui/Dialog.tsx`），自带焦点陷阱；危险操作使用 `ConfirmDialog`。
- 表格在窄屏必须可横向滚动（`components/ui/Table.tsx` 已内置 `overflow-x-auto`）。
- 空态用 `EmptyState`，错误用 `ErrorState`，加载用 `Skeleton`/`TableSkeleton`/`PageLoading`。不要只渲染 `null`。

## 表单

- 设备表单：react-hook-form + zod（`features/devices/schema.ts`），由 `zodResolver` 驱动。
- 表单是**扁平模型**：三种协议共享一组 RHF 字段，协议差异在 `superRefine` 中按 `deviceType` 校验，提交时由 `toPayload` 映射成 v1 形状。
- 默认值必须避开 Go `binding:"required"` 的零值：采集周期 1 秒、slave 1、functionCode 3、namespace 2。
- 编辑时 `deviceType`/`deviceModel` 不可切换；从 `?exploded=true` 载入后用 `toFormValues` 转回表单。
- 新增变量字段时，同时更新 `schema.ts` 的三处映射：`defaultFormValues`、`toFormValues`、`toPayload`，并同步 `BUSINESS.md`。

## Mock

- `VITE_USE_MOCK=true` 时启用 `mock/index.ts`，同时不注册 Vite proxy。
- mock 是**有状态**的：创建/更新/删除/启停/反控都会改变内存数据并递增 eTag；刷新页面重置。
- 新接口必须同时实现 mock 路由；未实现的 `/api/v1/*` 显式返回 404，避免误判为真实后端。
- mock 响应形状必须与 Go 一致，尤其 `eTag`、`cpus/mem/disk`、`Total/Used/UsedPercent`、`{data,total}`。
- mock 行为必须与 `BUSINESS.md` 第 9 节一致；业务变更时同步更新 fixtures。

## 样式与主题

- Tailwind v4 CSS-first：`index.css` 中 `@theme inline` 把语义变量映射为工具类。
- 深色模式通过 `<html class="dark">`；`ThemeProvider` 持久化到 `localStorage['harns-theme']`，首屏脚本在 `index.html` 内联，避免闪烁。
- 图表配色使用固定色板（`features/timeseries` 中的 `SERIES_COLORS`），保证明暗主题下都可辨识。
- 重复出现的类名组合用 `cn()` 合并或抽成常量，不在 JSX 中堆叠大段重复样式。

## 代码风格

**内聚与耦合**

- 默认按业务域就近放置代码；只有被多个域复用时才上移到 `shared`/`ui`，避免“万能工具”目录。
- 一个文件只承担一类职责：页面负责编排，数据映射放 `schema.ts`/`lib`，请求放 `api`，展示原语放 `components`。
- 禁止跨 feature 直接 import；需要共享时先抽到 `components/shared` 或 `lib`。
- 组件超过约 200 行或同时处理多个独立区块时，拆分子组件或 hooks。

**统一风格**

- 组件 `PascalCase`，函数/变量 `camelCase`，hook `useXxx`，常量 `UPPER_SNAKE_CASE`，类型/接口 `PascalCase`。
- 使用命名导出；文件内不用的导出及时删除。`import type` 用于纯类型导入。
- 导入顺序：React → 第三方 → `@/api` → `@/lib` → `@/providers` → `@/components` → 相对路径。
- 保持现有格式：2 空格缩进、单引号、无分号、尾随逗号；提交前 `pnpm typecheck` 必须通过。
- 注释使用中文，只解释“为什么”和非显然的业务约束；不复述代码。

**避免硬编码**

- 状态、协议、数据类型、单位、阈值、默认值、标签文案集中在 `lib/constants.ts` 或对应 feature 的常量文件；页面中不得重复一份。
- 颜色、间距、圆角优先用语义令牌与 Tailwind 工具类；禁止在组件里散落十六进制色值（图表固定色板除外）。
- 不写魔法数字/字符串：至少提取为具名常量；同一文案在多处出现时复用常量或映射。
- 时间、数字、百分比、字节格式化统一走 `lib/format.ts`，不要在每个页面各写一套。

**类型与状态**

- 禁止 `any`；确需未知类型用 `unknown` 并收窄。避免无理由的非空断言 `!`。
- 协议差异用可辨识联合（如 `DeviceDetail`），先判断 `deviceType` 再访问专有字段。
- 服务端状态交给 TanStack Query；组件内只保留必要的 UI 状态，不复制可由 props/query 派生的数据。
- 派生数据用 `useMemo`，仅在依赖变化或计算昂贵时使用，不做无意义 memo。

**健壮性**

- 所有请求错误都要处理；禁止空 `catch`、禁止静默失败。用户可见错误统一经 `errorMessage`。
- 异步操作要有 loading/disabled 状态，防止重复提交。
- 列表与详情必须覆盖 loading/error/empty 三态；未知枚举值不得导致崩溃。
- 不提交 `console.log`、调试代码、被注释掉的大段代码。

**可访问性**

- 表单控件必须有 label 或 `aria-label`；图标按钮提供 `title`/`aria-label`。
- 交互元素优先使用 `button`/`a`，保留键盘可达性与可见焦点。

## 提交前检查

```bash
pnpm typecheck
pnpm build
VITE_USE_MOCK=true pnpm dev
```

手动走查：概览 → 设备列表（启停/删除）→ 新建设备（三种协议）→ 详情（反控）→ 指标/KPI/测点/时序 → 设置；并切换一次深色主题。业务验收项见 `BUSINESS.md` 第 11 节。
