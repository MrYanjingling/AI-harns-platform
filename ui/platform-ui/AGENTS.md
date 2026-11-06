# ui/platform-ui/AGENTS.md — 前端工程约定

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
| UI 辅助 | lucide-react、sonner、clsx、tailwind-merge |

- 统一使用 pnpm：`pnpm install`、`pnpm dev`、`pnpm typecheck`、`pnpm build`、`pnpm preview`。不要混用 npm/yarn，也不要提交其他包管理器的锁文件。
- 不要擅自升级大版本或引入新的状态/样式/请求库；确需引入时先说明理由与替代方案。

## 目录

```
src/
├── main.tsx / router.tsx / index.css
├── api/          # types.ts（后端契约）、endpoints、query-keys、hooks
├── i18n/         # 消息字典（zh-CN / en-US）、Provider、translate
├── lib/          # http、session、format、constants、cn、query-client
├── providers/    # ThemeProvider、AuthProvider
├── components/
│   ├── layout/   # AppShell + Sidebar + nav-config + PagePad
│   ├── ui/       # 无业务语义的原语：Button/Card/Field/Dialog/Table/Badge…
│   ├── charts/   # GaugeRing 等
│   └── shared/   # 跨页面复用的业务组件：PageHeader/StatCard/PlatformStatusBadge/Pagination…
└── features/     # 按页面域组织：auth/dashboard/thing-models/things/identity/audit
mock/
├── index.ts      # Vite 中间件，内存态 API（网关）
├── platform.ts   # 平台服务中间件（auth + /iot/**）
├── utils.ts      # send/readJson 等共享工具
└── fixtures.ts   # 种子数据
```

## 登录态

- token 存放于 `localStorage['harns-session-token']`，读写集中在 `src/lib/session.ts`。
- `request('platform', …)` 自动附加 `Authorization: Bearer <token>`；登录、验证码等公开接口传 `skipAuth: true`。
- 平台接口返回 401 时 `http.ts` 清理 token 并派发 `SESSION_EXPIRED_EVENT`，`AuthProvider` 清空查询缓存，`RequireAuth` 跳转 `/login`。
- `/login` 在 `AppShell` 外，其余管理页由 `RequireAuth` 包裹。
- 会话状态交给 `AuthProvider` + TanStack Query（`useCurrentUser`）；组件里不要自行读写 token。

## 数据请求

- 所有 HTTP 走 `src/lib/http.ts` 的 `request(service, path, options)`：
  - `service` 为 `'gateway'` 或 `'platform'`。网关服务基址默认 `/api/v1`，平台服务基址由 `VITE_PLATFORM_BASE` 决定（默认 `/api/v1`），由 Vite proxy 或生产反向代理分流。
  - 错误统一抛 `ApiError`，展示时用 `errorMessage(error)`。
  - 上传图片用 `request('platform', path, { formData })`，不要手写 multipart 头。
- 后端契约类型集中在 `src/api/types.ts`；改类型前核对 `apis/` 与 Go 源码，不在组件里临时声明接口。
- Query key 集中定义在 `src/api/query-keys.ts`；组件中禁止手写 key 数组。
- 每个接口一个函数（`src/api/*.ts`），每个接口一个 hook（`src/api/hooks.ts`）：
  - 网关资源 10s 轮询；平台物列表 10s、详情 5s；其余查询按需配置。
  - 写操作成功后按域失效列表与相关详情缓存（`platform.things` / `thingModels` / `users` 等）。
- 平台服务响应统一 `{ code, message, data }`，分页为 `{ total, page, pageSize, list }`：
  - 成功解包由 `src/api/platform.ts` 的 `platformData` / `platformPage` 完成，页面只拿业务数据。
  - 业务失败 HTTP 400，以 body `code` 区分，`ApiError` 需保留 `errors[]`；401 视为会话失效。

## 组件

- 依赖方向：`features → components/shared → components/ui`，以及 `features/api/lib`；禁止 `features` 之间互相导入。
- `components/ui` 只放通用原语，不含业务概念；跨两个以上页面复用的业务组件放 `components/shared`；只服务单一页面的放对应 `features`。
- 复用现有视觉语义：卡片 `rounded-2xl border border-border bg-surface`，主色 `text-primary`，弱文本 `text-muted/faint`，数字用 `tabular`。
- 颜色只引用 `index.css` 中的语义令牌（`bg-surface`、`border-border`、`text-danger` 等）；不写死 zinc/emerald 调色板（状态徽标等既有例外除外）。
- 弹窗使用原生 `<dialog>`（`components/ui/Dialog.tsx`），自带焦点陷阱；危险操作使用 `ConfirmDialog`。
- 表格在窄屏必须可横向滚动（`components/ui/Table.tsx` 已内置 `overflow-x-auto`）。
- 空态用 `EmptyState`，错误用 `ErrorState`，加载用 `Skeleton`/`TableSkeleton`/`PageLoading`。不要只渲染 `null`。

## 表单

- 平台表单（登录、用户、角色、组织、物模型、物）统一 RHF + zod；编辑用户不提交 `roleIds/orgIds`（后端不支持），批量创建物按行维护数组字段。
- 物模型属性的 `dataType` 是递归结构，`features/thing-models/DataTypeEditor.tsx` 与 `PropertyDialog.tsx` 使用受控状态 + `data-type.ts` 中的 zod 校验函数，字段错误路径需与后端一致（如 `dataType.enumSpec`）。
- 属性/功能标识符规则：`^[A-Za-z][A-Za-z0-9_]*$`，保留字 `time`、`device_id`；前端校验只是提前反馈，最终以后端为准。

## Mock

- `VITE_USE_MOCK=true` 时启用 `mock/index.ts`，同时不注册 Vite proxy。
- mock 是**有状态**的：创建/更新/删除都会改变内存数据并递增版本号；刷新页面重置。
- 平台接口在 `mock/platform.ts`：维护内存会话、默认密码、种子用户/角色/组织/物模型/物/审计日志；`/auth/publicKey` 返回空串以便走明文登录回退。
- 新接口必须同时实现 mock 路由；未实现的 `/api/v1/*` 显式返回 404，避免误判为真实后端。
- mock 响应形状必须与 Go 一致，尤其 `eTag`、`cpus/mem/disk`、`Total/Used/UsedPercent`、平台 `{code,message,data}` 与 `{total,page,pageSize,list}`。
- 物的在线状态由 mock 按 `enable === 1 && 最近 60s 有活动` 派生，不要存成静态字段。
- mock 行为必须与 `BUSINESS.md` 第 10 节一致；业务变更时同步更新 fixtures。

## 样式与主题

- Tailwind v4 CSS-first：`index.css` 中 `@theme inline` 把语义变量映射为工具类。
- 深色模式通过 `<html class="dark">`；`ThemeProvider` 持久化到 `localStorage['harns-theme']`，首屏脚本在 `index.html` 内联，避免闪烁。
- 重复出现的类名组合用 `cn()` 合并或抽成常量，不在 JSX 中堆叠大段重复样式。

## 国际化

- 界面文案只支持 `zh-CN` / `en-US`，默认 `zh-CN`；语言持久化到 `localStorage['harns-locale']`，切换时同步 `<html lang>`，并随请求发送 `Accept-Language`。
- 组件内使用 `useI18n()` 的 `t(key, vars?)`；非组件模块（`lib/format.ts`、`lib/http.ts`、`schema.ts`）使用 `translate(key, vars?)`。禁止在 JSX、toast、校验提示里硬编码中英文。
- 消息字典放 `src/i18n/messages/<domain>.ts`，用 `defineMessages({ 'zh-CN': {...}, 'en-US': {...} })` 并排维护；key 以域名为前缀（如 `things.title`），两种语言的 key 必须完全一致，缺键会在 `pnpm typecheck` 报错。
- 插值用 `{name}`；带数量的文案提供 `xxx.one` / `xxx.other` 两个 key，由 `t` 按 `count` 自动选择。
- 状态、协议、审计动作等枚举的展示文案存 `labelKey` / `descriptionKey`，在渲染处 `t(...)`；技术名称（Modbus、OPC UA、S7、bool 等）不翻译。
- zod 校验消息通过 `createXxxSchema(t)` 工厂构建，表单内用 `useMemo` 缓存；不要在模块顶层固化某一种语言。
- 时间、数字、相对时间统一走 `lib/format.ts`，内部按当前语言格式化；不要自行拼接本地化文案。
- 后端返回的业务数据（设备名称、物模型名称、用户名等）按原样展示；后端错误文案由平台服务按 `Accept-Language` 返回。

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