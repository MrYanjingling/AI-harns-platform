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
| 表格数据 | `text-sm leading-5 text-fg` |
| 普通弱文本 | `text-xs text-muted` |
| 提示/辅助文本 | `text-[11px] text-faint` |
| 地址、Topic、ID、数值 | `font-mono` |
| 状态徽标 | `text-xs font-medium` |

- 列表状态徽标统一使用 `border-[1.5px] px-2 py-1`。
- 技术名称（Modbus、MQTT、QoS、bool 等）按原样展示，不翻译。
- 时间统一使用 `formatDateTime`，数字格式化优先使用 core 提供的工具。

## 表格

- 网关列表统一使用 `GatewayTable`、`GatewayTHead`、`GatewayTH`、`GatewayTBody`、`GatewayTR`、`GatewayTD`。
- 固定布局：
  - `tableLayout: 'fixed'`
  - `width: max(100%, ${totalWidth}px)`
- 列宽使用 `useColumnResize`，每个表头挂 `ColumnResizeHandle`。
- 序号列居中；操作列表头左对齐。
- 行 hover 使用组件内置的 `surface-2`，不要重复定义表头、分隔线和行样式。
- 列表必须覆盖 loading、error、empty 三态：
  - loading：`TableSkeleton`
  - error：`ErrorState compact`
  - empty：`EmptyState`，有主操作按钮

## 按钮

- 页面主操作：primary，例如搜索、新建、保存。
- 次要操作：outline，例如取消、编辑、启动、停止。
- 弱操作：ghost，例如重置。
- 行内操作使用文本按钮：
  - 更新：`text-sm text-primary`
  - 删除：`text-sm text-danger`
  - 更多：`size-7` 图标按钮 + Portal 菜单
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
- 字段优先使用 inline `Field`：
  - `labelClassName="text-sm"`
  - `labelWidthClassName="w-[84px]"` 或 `w-[96px]"`
- 普通输入宽度控制在 200~240px；地址、表达式、Topic 可占整行。
- 下拉统一使用 `SearchSelect` 或 `Select`，不要直接写裸 `<select>`。
- 使用 `SearchSelect` 时，触发框和选项面板必须同宽；组件按触发框实际宽度定位，`panelClassName` 不得单独设置不同宽度。
- 同一表单内的普通输入框默认同宽。Broker 地址、用户名、密码、Topic 等基础输入统一 240px，除非明确需要整行展示。
- 开关使用 `Switch`，旁边文字使用 `text-sm font-medium text-muted`，说明使用 `text-[11px] text-faint`。

## 状态与反馈

- 采集状态、连接状态等带错误说明的徽标，使用自定义 tooltip，不只依赖原生 `title`。
- 成功操作使用 toast；失败统一 `showErrorToast(error)`。
- 异步提交按钮必须有 loading/disabled，避免重复提交。
- 删除、启动、停止、重启等操作先确认再请求。

## 命名

- 左侧菜单的“连接实例”是页面名称；代码和 API 类型仍可使用 `Connector`。
- 面向用户的文案统一使用“连接实例”和“连接组”，不再使用“连接器”“推送组”。
- 面向用户的文案统一使用“采集实例”或“采集通道”，按现有页面语境选择，不混用。
