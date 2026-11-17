# 网关应用视觉与交互规范

适用于 `ui/gateway-ui`。通用前端规则见 `ui/platform-ui/AGENTS.md`；本文件补充网关列表、表格、按钮与表单的统一基线，冲突时以本文件为准。

## 页面骨架

- 页面内容放在 `GatewayPagePad` 下，由 AppShell 提供路由标题，页面内不要再写 `PageHeader`。
- 页面根节点使用 `flex min-h-0 flex-1 flex-col`。
- 顶部工具栏使用 `mb-3 flex shrink-0 flex-wrap items-center gap-2`。
- 列表 Card 使用 `flex min-h-0 flex-1 flex-col overflow-hidden`。
- 表格滚动区使用 `min-h-0 flex-1 overflow-auto`。
- 分页固定在 Card 底部，外层使用 `shrink-0`。
- 工具栏按钮和搜索框统一 `h-9`；搜索/新建使用 primary，重置使用 ghost。

## 字体

| 用途 | 样式 |
|---|---|
| 区块标题 | `text-sm font-semibold` |
| 表单标签 | `text-sm font-medium text-muted` |
| 表格表头 | `text-sm font-medium text-fg` |
| 菜单主列表表格数据 | `text-sm font-normal font-table text-table-fg` |
| 普通弱文本 | `text-xs text-muted` |
| 提示/辅助文本 | `text-[11px] text-faint` |
| 非表格技术字段 | `font-mono` |
| 状态指示 | 图标 `size-3.5` + `text-sm font-medium` |

- 菜单主列表表格数据统一为 14px、字重 400，浅色主题 `#303133`，深色主题使用 `--fg`。
- 表格数据使用 `--font-table`：`-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", sans-serif`。
- `GatewayTD` 内不得再覆盖字号、字重、颜色或 `font-mono`；序号、时间、类型、名称、地址、Topic、ID、数值全部继承统一数据样式。
- 可点击计数列保持统一数据色，仅保留 `hover:text-primary`。
- 状态指示、开关、操作按钮不受表格数据字体规则影响。
- 弹框与采集实例详情页内表格暂不套用菜单主列表规则，修改时先确认范围。
- 技术名称（Modbus、MQTT、QoS、bool 等）按原样展示，不翻译。
- 时间统一使用 `formatDateTime`，数字格式化优先使用 core 提供的工具。

## 表格

- 网关列表统一使用 `GatewayTable`、`GatewayTHead`、`GatewayTH`、`GatewayTBody`、`GatewayTR`、`GatewayTD`。
- 菜单主列表的数据字体由 `GatewayTD` 默认提供，页面内不要重复声明数据字号、字重、颜色和等宽字体。
- 固定布局：
  - `tableLayout: 'fixed'`
  - `width: max(100%, ${totalWidth}px)`
- 列宽使用 `useColumnResize`，每个表头挂 `ColumnResizeHandle`。
- 序号列居中；操作列表头左对齐。
- 行 hover 使用组件内置的 `surface-2`，不要重复定义表头、分隔线和行样式。
- 列表必须覆盖 loading、error 两态：
  - loading：`TableSkeleton`
  - error：`ErrorState compact`
- 数据区没有数据时只留空，不显示 `EmptyState`、引导文案、图标或其他占位内容；未选择筛选条件时同样留空。

## 按钮

- 页面主操作：primary，例如搜索、新建、保存。
- 次要操作：outline，例如取消、编辑、启动、停止。
- 弱操作：ghost，例如重置。
- 行内操作统一使用 `TableActionButton` 图标按钮：
  - 默认图标 `text-muted`，hover `text-fg` + `bg-surface-2`
  - 必须带自定义深色 tooltip、`title` 和 `aria-label`
  - 更多操作使用 `size-7` 图标触发按钮 + Portal 菜单，菜单内部保留图标和文字
- 列表操作列默认左对齐、宽度 `140px`，单元格内容使用 `flex items-center gap-2`。
- 常用操作直接展示；只有运行、推送等低频操作保留“更多”菜单。
- 弹框底部统一为：outline 取消 + primary 保存。
- 危险删除使用 `ConfirmDialog danger`。
- 启动/停止等状态操作使用 `ConfirmDialog`。
- 图标按钮必须有 `title` 和 `aria-label`。

## 弹框与表单

- 新建/编辑弹框统一：
  - `Dialog size="xl"`
  - `className="max-w-[970px] rounded-lg"`
  - `bodyClassName="max-h-[calc(100dvh-190px)] space-y-3 overflow-y-auto p-4"`
- 弹框内容按业务分区使用 Card、CardHeader、CardTitle、CardContent。
- 弹框描述文字常规统一为 `text-[13px] leading-relaxed text-muted`，与删除确认弹框的描述文字样式保持一致，不使用 11px。
- 接口提交成功后立即关闭弹框并保留成功 toast；失败时保持弹框打开并展示错误。
- 确认弹框在接口成功后立即清除目标状态并关闭。
- 字段优先使用 inline `Field`：
  - `labelClassName="text-sm"`
  - `labelWidthClassName="w-[84px]"` 或 `w-[96px]"`
- 普通输入宽度控制在 200~240px；地址、表达式、Topic 可占整行。
- 下拉统一使用 `SearchSelect` 或 `Select`，不要直接写裸 `<select>`。
- 使用 `SearchSelect` 时，触发框和选项面板必须同宽；组件按触发框实际宽度定位，`panelClassName` 不得单独设置不同宽度。
- 同一表单内的普通输入框默认同宽。Broker 地址、用户名、密码、Topic 等基础输入统一 240px，除非明确需要整行展示。
- 开关使用 `Switch`，旁边文字使用 `text-sm font-medium text-muted`，说明使用 `text-[11px] text-faint`。

## 状态与反馈

- 采集状态、连接状态统一使用「彩色图标 + 状态文字」的 Naive UI Alert 风格：浅色背景、同色边框、圆角，不使用旧 Badge 圆点样式。
- 状态组件为 `CollectStatusBadge` / `ConnectorStatusBadge`；图标 `size-3.5`，文字 `text-sm font-medium`，图标和文字使用同一语义色。
- 状态颜色映射：
  - 绿色 `border-success/35 bg-success/10 text-success`：采集中、运行中，图标 `CircleCheck`
  - 黄色 `border-warning/35 bg-warning/10 text-warning`：已停止、无点位，图标 `CirclePause` / `CircleSlash`
  - 红色 `border-danger/35 bg-danger/10 text-danger`：未连接、采集异常、错误，图标 `CircleAlert` / `CircleX`
  - 未知状态：`border-border bg-surface-2 text-muted`，图标 `CircleHelp`
- 状态组件统一提供圆角、浅色背景、边框和 `px-2 py-0.5`；调用方不要再传背景、边框或 padding 类名。
- 状态 hover tooltip 与操作图标 tooltip 保持一致：`bg-header` / `text-header-fg`、11px、正上方居中、带小箭头；异常状态展示 `lastError`。
- 状态 tooltip 仅在鼠标 hover 或键盘 `:focus-visible` 时显示；弹框自动聚焦不得触发 tooltip。
- 采集状态、连接状态等带错误说明的状态，使用自定义 tooltip，不只依赖原生 `title`。
- 成功操作使用 toast；失败统一 `showErrorToast(error)`。
- 接口提示参照 Naive UI Message：中性表面、细边框、轻阴影；成功使用绿色 `CircleCheck`，错误使用红色 `CircleX`，文字保持主文本色，不使用整块彩色背景。
- toast 默认 3 秒自动关闭，不显示关闭按钮；多条错误信息在正文内分行展示。
- 异步提交按钮必须有 loading/disabled，避免重复提交。
- 删除、启动、停止、重启等操作先确认再请求。

## 命名

- 左侧菜单的“连接实例”是页面名称；代码和 API 类型仍可使用 `Connector`。
- 面向用户的文案统一使用“连接实例”和“连接组”，不再使用“连接器”“推送组”。
- 面向用户的文案统一使用“采集实例”或“采集通道”，按现有页面语境选择，不混用。
