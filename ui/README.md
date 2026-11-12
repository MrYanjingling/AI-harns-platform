# 前端工作区

`ui/` 是一个 pnpm workspace:一个主应用、一个平台子应用、一个独立的网关应用,以及一个不含业务语义的公共核心库。三个应用都通过 `@harns/core` 共享 HTTP、会话、i18n、主题、UI 原语与 `AppShell`。

| 包 | 目录 | 端口(dev) | 职责 |
|---|---|---|---|
| `@harns/host` | `host` | 5173 | 主应用:登录、路由装配、平台导航、平台 mock |
| `@harns/platform-ui` | `platform-ui` | 5174 | 平台子应用:认证、物模型、物、用户/角色/组织、审计 |
| `@harns/gateway-ui` | `gateway-ui` | 5175 | 网关应用(独立运行):概览、采集、推送、驱动、日志、性能 |
| `@harns/core` | `core` | — | 公共库:HTTP、会话、i18n、主题、UI 原语、导航与 AppShell |

`eam-ui`、`ems-ui` 目前仍只有 `.gitkeep`,不参与工作区构建。

## 快速开始

```bash
cd ui
pnpm install

pnpm dev              # 主应用(host),http://localhost:5173
pnpm dev:platform     # 只启动平台子应用,http://localhost:5174
pnpm dev:gateway      # 只启动网关应用,http://localhost:5175

pnpm typecheck        # core → platform-ui → gateway-ui → host
pnpm build            # 依次构建 platform-ui、gateway-ui、host
```

默认 `.env` 中 `VITE_USE_MOCK=true`,不需要 Go 后端。平台 mock 种子账号为 `admin` / `harns@iot`。连接真实后端时设为 `false`:host 的 Vite proxy 把 `/api/v1/auth`、`/api/v1/iot` 转发到 platform(`32300`);gateway-ui 的 proxy 把 `/api/v1/*` 转发到 gateway(`32200`)。

> 依赖说明:当前 pnpm 版本未自动解析 Rollup / Tailwind Oxide 的平台二进制可选依赖,根 `package.json` 显式声明了 macOS / Linux / Windows 常见架构的 `optionalDependencies`。升级 Vite / Tailwind 时需同步这些版本;若上游修复后可删除。

## 组合方式

主应用在**构建期**直接编译平台子应用源码,不做运行时模块联邦:

- 子应用通过 `package.json` 的 `exports.types` 暴露 `app.d.ts`,声明 `platformRoutes`、`createPlatformNav`、`platformMessages` 等稳定接口;`exports.import` 指向 `src/app.tsx`。
- 主应用在 `host/src/routes.tsx` 中拼接平台子应用的路由与导航段,在 `host/src/main.tsx` 中合并 i18n 字典。
- 子应用源码内部仍使用 `@/` 指向自己的 `src/`。`host/vite.config.ts` 的 `harns-subapp-source-alias` 插件按 importer 所在包解析 `@/`,避免别名串包。
- TypeScript 只读取子应用的 `app.d.ts`,不把子应用源码纳入 host 的类型检查;因此子应用必须保持 `app.d.ts` 与 `src/app.tsx` 同步。
- `resolve.dedupe` 保证 React、React Router、TanStack Query 在整棵树上只有一份实例。

**网关应用 (`gateway-ui`) 不参与上述组合。** 它是独立应用:

- 自带 `index.html`、`src/main.tsx`、路由、导航与 i18n 注册,开发时 `pnpm dev:gateway` 独立启动。
- 构建产物由 `cmd/gateway` 直接托管(`etc/gateway/config.yaml` 的 `ui-dir: "ui/gateway-ui/dist"`),路由位于站点根路径;不依赖 host,也不向 host 暴露 `app.d.ts`。
- 与 host、platform-ui 的唯一共享方式是 `@harns/core`:公共能力一律先放进 core,业务代码留在各应用内。

## 核心库边界

`core` 只放**与业务无关**的代码:

- `lib/`:HTTP 请求、会话 token、格式化、查询客户端、`cn`。
- `i18n/`:全局字典注册表、`I18nProvider`、`defineMessages`,以及 `common/components/layout` 通用文案。
- `providers/`:主题。
- `components/ui`:`Button/Card/Dialog/Table/...` 等无业务原语。
- `components/shared`:`PageHeader/Pagination/StatCard/ErrorState/NotFoundPage` 等跨应用组件。
- `components/layout`:`AppShell`、`Sidebar`、`AccountMenu`、导航模型与快速搜索。

平台/网关的业务常量、接口类型、feature 页面必须留在各自应用中。新增核心代码前先确认它不含业务概念。

## 新增子应用

1. 新建 `ui/<name>-ui`,复制现有子应用的 `package.json`、`tsconfig.json`、`vite.config.ts` 骨架,依赖 `@harns/core`。
2. 导出 `src/app.tsx`(routes、nav、messages)并在 `app.d.ts` 中声明。
3. 在 `host/src/routes.tsx` 与 `host/src/main.tsx` 注册路由、导航和字典。
4. 在根 `pnpm-workspace.yaml` 的 `packages` 中加入目录。
5. 如果它需要独立部署而不是被 host 组合(参照 `gateway-ui`),则跳过第 2、3 步:自带 `index.html`、`src/main.tsx` 与路由,构建为独立站点。

## 约定

- 前端工程约定见 `platform-ui/AGENTS.md`;根目录 `AGENTS.md` 第 6 节是摘要。
- 所有 HTTP 走 `@harns/core` 的 `request(service, path, options)`;查询 key 集中定义,每个接口一个 hook。
- 色彩只用 `core/src/styles.css` 中的语义令牌,不写死调色板。
- 文案只支持 `zh-CN` / `en-US`,两种语言的 key 必须一致;应用用 `defineMessages` 在编译期保证。
