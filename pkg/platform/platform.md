# harns-platform 模块开发规范（Vibe Coding）

> 本文件是 `platform` 模块的唯一事实来源。开发/生成代码前先读本文；实现必须满足「约束」，并按「功能清单」序号逐条交付。

---

## 1. 背景与目标

- 1.1 `harns-platform` 多服务物联网平台，已有 `gateway`，现新增 `platform`。
- 1.2 platform 定位：设备接入之外的基础能力（账号、租户、组织、资产、数据存储等，见第 10 节）。
- 1.3 增量开发：不破坏 gateway，不改动通用层语义。

## 2. 技术栈与依赖

- 2.1 Go 1.24（`go.mod` 的 `go` 指令）。
- 2.2 Web：`gin`；配置：`sigs.k8s.io/yaml` + cobra/pflag（`/etc/platform/config.yaml`，`-c` 可覆盖）。
- 2.3 日志：`harns-platform/pkg/logging`（slog 门面，非 klog）。
- 2.4 存储：MySQL（`go-sql-driver/mysql`）、Redis（`redis/go-redis/v9`）、MinIO（`minio/minio-go/v7`）。
- 2.5 主键用 `oklog/ulid/v2`；密码哈希用 `golang.org/x/crypto/bcrypt`。
- 2.6 新增第三方依赖需说明理由，且兼容 Go 1.24。

## 3. 目录与分层

- 3.1 通用层（禁止被模块反向依赖）：`apis`、`generic`、`host`、`i18n`、`logging`、`tsdb`、`utils`、`version`。
- 3.2 模块层：业务代码放 `pkg/platform/`，入口在 `cmd/platform/`；禁止写进 `pkg/gateway` 或通用层。
- 3.3 模块内聚：身份与组织（用户/角色/组织/关联）合并为单包 `pkg/platform/identity/`（model/store/service/server/v1），同包直接调用，避免循环依赖。
- 3.4 模块间调用：工业应用为独立服务，通过 HTTP API 交互；platform 内部跨模块时才由消费方定义接口（contract）并在组合根注入，禁止全局单例。
- 3.5 路径约定：业务接口 `/api/v1/{app}/...`（当前 app=`iot`）；认证/当前用户等共享接口 `/api/v1/auth/...`；健康检查 `/api/v1/health`。审计 `app` 由服务绑定（platform 为 `IOT`）。

## 4. 通用编码约束

- 4.1 Go 官方风格；提交前 `gofmt`。
- 4.2 错误显式处理，禁止 `_ = err`、禁止 panic 处理可预期错误。
- 4.3 不在业务层直接 `c.JSON`，统一走第 5 节响应封装。
- 4.4 面向接口 + 依赖注入（构造函数传 `*config.Config`），便于测试。
- 4.5 面向用户的错误必须走第 6 节 i18n，禁止硬编码文案。

## 5. API 响应规范

- 5.1 统一返回 `pkg/apis/response` 的 `Result`：
  - 成功 `{"code":200,"message":"success","data":<payload>}`
  - 失败 `{"code":<业务码>,"message":"<本地化消息>","errors":[{"code":..,"message":..}]}`
- 5.2 成功 `response.WriteOK(c, data)`；失败 `response.WriteError(c, status, err...)`。
- 5.3 业务错误（参数校验、重复、不存在等）**HTTP 一律 400**，具体原因由 body 的 `code` 区分；HTTP 语义码只用于非业务场景：401 未认证、403 无权限、500 内部错误。
- 5.4 前端约定：`code===200` 用 `data`，否则用 `errors`。
- 5.5 分页：参数用 `apis.PageQuery`（`Normalize()` 后 `Offset()`/`Limit()`），返回 `apis.PageResp[T]`（`data` 含 `total/page/pageSize/list`）。
- 5.6 字段命名 lowerCamelCase；接口文档放 `apis/platform/`。
- 5.7 参数校验：用 `httpx.BindJSON`/`BindQuery`；失败返回 400，`errors[]` 每项含 `field` 与本地化 `message`（`validate.*`）。
- 5.8 CORS：`httpx.CORS` 由 `cors` 配置控制（allowed-origins/methods/headers、credentials、max-age）；OPTIONS 预检返回 204。

## 6. 错误码与国际化

- 6.1 错误码在 `pkg/apis/response/constants.go`（`ErrCode`，10001 起，只能末尾追加）。
- 6.2 在 `pkg/apis/response/errors.go` 的 `messageIDs` 登记语义 key（如 `error.user_not_found`）。
- 6.3 文案放 `pkg/i18n/locales/<lang>.yaml`（至少 en-US、zh-CN，占位符 `%s`/`%d`）。
- 6.4 用 `response.ErrXxx(...)` 构造，不在业务层拼字符串。
- 6.5 语言来自 `?lang=` 或 `Accept-Language`，用 `i18n.Lang(c)` 读取。

## 7. 配置规范

- 7.1 配置项在 `cmd/platform/options/Options`（`json` tag），默认值在 `NewDefaultOptions`。
- 7.2 同步更新 `etc/platform/config.yaml` 模板（含注释）。
- 7.3 敏感信息默认留空，不入库、不打日志。
- 7.4 优先级：CLI flag > 配置文件 > 内置默认（框架已实现，勿破坏）；启动时 `options.Validate` 校验。
- 7.5 认证配置 `security`（见 F4）：`token-secret` 必须 ≥32 字节，变更会使所有旧 token 失效；`rsa-enabled=true` 时启动依赖 MinIO。
- 7.6 启动自检：启动时 Ping MySQL/Redis 并配置连接池（`max-open-conns`/`max-idle-conns`/`conn-max-lifetime-minutes`）；`migrate: true` 时用 **golang-migrate** 执行内嵌迁移（`migrations/NNNNNN_name.up.sql` / `.down.sql`，版本表 `schema_migrations`）。任一失败即启动失败。
  > 备注：若环境曾运行旧版自研迁移器，旧 `schema_migrations`（`version VARCHAR`）与新表结构不兼容，升级前请先删除该表（新库无影响）。
- 7.7 探针：`GET /api/v1/health`（存活，恒 200）、`GET /api/v1/ready`（检查 DB/Redis，失败返回 503）。
- 7.8 审计/CORS：`audit.retention-days`（默认 180）；CORS 见 `cors` 段（§5.8）。

## 8. 日志规范

- 8.1 用 `pkg/logging`：`logging.V(n).InfoS(...)`、`logging.ErrorS(err, ...)`；禁止 `fmt.Println`/标准库 `log`。
- 8.2 级别：V(1) 里程碑、V(2) 日常（默认可见）、V(3+) 细节；错误带 `err`。
- 8.3 文件输出与切割由 logging 配置控制（`max-size`/`max-backups`/`max-age`/`compress`），勿自行实现。
- 8.4 方法入口/出口：Service 层每个导出方法**第一行**写 `defer logging.Trace("identity.Xxx", kv...)()`；`Trace` 以 V(3) 输出 `enter`/`exit` 及 `cost`。
- 8.5 入口 kv 只带关键入参（如 `id`、`name`、`code`），**禁止打印 password/authKey 等敏感信息**。
- 8.6 handler 与 store 不重复记录入口/出口（避免日志风暴）；错误在出错处用 `logging.ErrorS`/`V(2).InfoS` 记录。
- 8.7 排查时把 `logging.verbosity` 调到 3 即可看到全部方法入口/出口。

## 9. 存储规范

- 9.1 数据访问用 `config.Config`：MySQL `DB`、Redis `Redis`、MinIO `MinIO`；SQL 参数化，Redis key 带业务前缀并设 TTL。
- 9.2 表结构变更用 SQL 迁移脚本，放 `migrations/`。
- 9.3 实体统一内嵌 `apis.Meta`（`id`/`version`/`createdUser`/`updatedUser`/`createdTime`/`updatedTime`，勿重复定义）；`id` 用 ULID（列宽 32）；`version` 每次更新 +1，`ETag`=version 仅作参考，不做 If-Match 校验。用户关联用 `user_roles`/`user_orgs`（无外键），删除时同事务清理。
- 9.4 审计字段：创建时 `CreatedUser`/`UpdatedUser`、更新时 `UpdatedUser` 一律取 `principal.UserID(ctx)`（鉴权中间件已注入）；禁止硬编码或留空。

## 10. 功能清单（按序号编写）

> 每个功能用如下格式，序号不可跳号；未实现前保持 `状态: TODO`。
>
> 初始化数据：迁移会创建超级管理员角色（`code=admin`）与账号 `admin / admin@qwer`，首次登录后请修改密码。

### F1. 用户管理（CRUD + 分页）
- 状态：TODO
- 目标：用户创建、按 id 查询、更新、删除、分页查询 5 个 REST 接口。
- 输入：字段见下表。
- 处理：1) 迁移脚本建 `users`、`user_roles`、`user_orgs` 表；2) 实体 `User`（内嵌 `apis.Meta`）；3) 单包 `pkg/platform/identity/`（model/store/service/server/v1）；4) 路由注册到 `pkg/platform/web`，文档到 `apis/platform/user.yaml`。
- 输出：5 个接口 + 迁移脚本 + API 文档。
- 字段：

  | 字段 | Go 类型 | 约束 |
  |---|---|---|
  | userName | string | 必填，唯一 |
  | password | string | 可选，默认 `用户名+@qwer`；bcrypt 落库，响应不回传 |
  | userNo | string | 必填，唯一 |
  | nickName | string | 必填 |
  | birthDate | string `2006-01-02` | 可空 |
  | gender | string | 枚举 male/female |
  | mail | string | 可空 |
  | phone | string | 可空 |
  | note | string | 可空 |
  | authKey | string | 可空，响应不回传 |
  | authMode | string | 枚举 local/ldap/radius |
  | lastLoginTime | *time.Time | 可空 |
  | loginFailureTimes | int | 默认 0 |
  | expireTimes | int | 默认 0 |
  | status | string | VARCHAR，枚举 enabled/disabled |
  | roleIds | []string | 可多选，校验角色存在 |
  | orgIds | []string | 可多选，校验组织存在 |

  > 基础字段来自 `apis.Meta`（见 §9.3），不要 `deleted` 列。
- 接口：
  - POST `/api/v1/iot/users` → 200 + UserVo
  - GET `/api/v1/iot/users/:id` → 200 + UserVo
  - PUT `/api/v1/iot/users/:id` → 200 + UserVo
  - DELETE `/api/v1/iot/users/:id` → 200（物理删除，同事务清理关联）
  - PUT `/api/v1/iot/users/:id/password` → 200 + UserVo（重置密码为 `用户名+@qwer`）
  - GET `/api/v1/iot/users` → 200 + `PageResp[UserVo]`
- 约束：遵循 §3~§9；DTO 放 `pkg/platform/identity/v1`（`XxxReq`/`XxxQuery`）；对外一律 `XxxVo`，禁止返回实体（防泄露 password/authKey）；`createdUser`/`updatedUser` 取自登录上下文（`principal.UserID(ctx)`）。
- 验收标准：
  - [ ] 5 个接口按 §5.1 返回
  - [ ] 列表用 `PageQuery`/`PageResp`，`data` 含 total/page/pageSize/list
  - [ ] userName/userNo 重复 → 400 + 业务码 `error.user_name_exists`/`error.user_no_exists`；id 不存在 → 400 + `error.user_not_found`
  - [ ] 响应体不含 password/authKey；password 落库为 bcrypt
  - [ ] 创建时未传 password → 使用 `用户名+@qwer`；重置密码接口将密码置为 `用户名+@qwer`
  - [ ] 创建时 `roleIds`/`orgIds` 校验存在（不存在 → 400 `error.role_not_found`/`error.org_not_found`）
  - [ ] 用户 + `user_roles` + `user_orgs` 同一事务写入，失败整体回滚
  - [ ] `UserVo` 返回 `roleIds`/`orgIds`，以及关联的 `roles`/`orgs` 对象数组（列表亦是）
  - [ ] 删除用户同事务清理关联；更新后 `version` +1、`ETag` 反映新值
  - [ ] 错误文案支持 zh-CN/en-US；产出迁移脚本与 API 文档

### F2. 角色管理（CRUD + 分页）
- 状态：TODO
- 目标：角色创建、按 id 查询、更新、删除、分页查询 5 个 REST 接口。
- 输入：字段见下表。
- 处理：
  1. 迁移脚本建 `roles` 表；
  2. 实体 `Role`（内嵌 `apis.Meta`）；
  3. 单包 `pkg/platform/identity/`（model/store/service/server/v1）；
  4. 路由注册到 `pkg/platform/web`，文档到 `apis/platform/role.yaml`。
- 输出：5 个接口 + 迁移脚本 + API 文档。
- 字段：

  | 字段 | Go 类型 | 约束 |
  |---|---|---|
  | name | string | 必填，唯一 |
  | code | string | 必填，唯一 |
  | description | string | 可空 |
  | status | string | VARCHAR，枚举 enabled/disabled |

  > 基础字段来自 `apis.Meta`（见 §9.3），不要 `deleted` 列。
- 接口：
  - POST `/api/v1/iot/roles` → 200 + RoleVo
  - GET `/api/v1/iot/roles/:id` → 200 + RoleVo
  - PUT `/api/v1/iot/roles/:id` → 200 + RoleVo
  - DELETE `/api/v1/iot/roles/:id` → 200（物理删除，同事务清理 `user_roles`）
  - GET `/api/v1/iot/roles` → 200 + `PageResp[RoleVo]`
- 约束：遵循 §3~§9；DTO 放 `pkg/platform/identity/v1`（`XxxReq`/`XxxQuery`）；对外一律 `RoleVo`（无敏感字段，含全部业务字段）；`createdUser`/`updatedUser` 取自登录上下文（`principal.UserID(ctx)`）。
- 验收标准：
  - [ ] 5 个接口按 §5.1 返回
  - [ ] 列表用 `PageQuery`/`PageResp`，`data` 含 total/page/pageSize/list
  - [ ] name/code 重复 → 400 + `error.role_name_exists`/`error.role_code_exists`；id 不存在 → 400 + `error.role_not_found`
  - [ ] 删除角色同事务清理 `user_roles`；更新后 `version` +1
  - [ ] 错误文案支持 zh-CN/en-US；产出迁移脚本与 API 文档

### F3. 组织(部门)管理（CRUD + 树状查询）
- 状态：TODO
- 目标：组织创建、按 id 查询、更新、删除、树状查询 5 个接口（**无分页**）。
- 输入：字段见下表。
- 处理：
  1. 迁移脚本建 `orgs` 表；
  2. 实体 `Org`（内嵌 `apis.Meta`）；
  3. 单包 `pkg/platform/identity/`（model/store/service/server/v1）；
  4. 路由注册到 `pkg/platform/web`，文档到 `apis/platform/org.yaml`。
- 输出：5 个接口 + 迁移脚本 + API 文档。
- 字段：

  | 字段 | Go 类型 | 约束 |
  |---|---|---|
  | code | string | 必填，唯一 |
  | name | string | 必填 |
  | parentId | string | 可空，空/0 为根 |
  | sort | int | 排序，默认 0 |
  | description | string | 可空 |
  | remark | string | 可空 |
  | status | string | VARCHAR，枚举 enabled/disabled |

  > 基础字段来自 `apis.Meta`（见 §9.3），不要 `deleted` 列。
- 接口：
  - POST `/api/v1/iot/orgs` → 200 + OrgVo
  - GET `/api/v1/iot/orgs/:id` → 200 + OrgVo
  - PUT `/api/v1/iot/orgs/:id` → 200 + OrgVo
  - DELETE `/api/v1/iot/orgs/:id` → 200（物理删除；**存在子节点则拒绝**；同事务清理 `user_orgs`）
  - GET `/api/v1/iot/orgs/tree` → 200 + `[]OrgVo`（树，`children` 嵌套）
- 约束：遵循 §3~§9；DTO 放 `pkg/platform/identity/v1`；对外一律 `OrgVo`（树状携带 `children`）；`createdUser`/`updatedUser` 取自登录上下文（`principal.UserID(ctx)`）；更新 `parentId` 需防成环（不能设为自己或其后代）。
- 验收标准：
  - [ ] 5 个接口按 §5.1 返回
  - [ ] 树状查询返回根节点数组，`children` 嵌套
  - [ ] code 重复 → 400 + `error.org_code_exists`；id 不存在 → 400 + `error.org_not_found`
  - [ ] 父节点不存在 → 400 + `error.org_parent_not_found`；parentId 设为自己/后代 → 400 + `error.org_parent_cycle`
  - [ ] 删除存在子节点的组织 → 400 + `error.org_has_children`
  - [ ] 删除组织同事务清理 `user_orgs`；更新后 `version` +1
  - [ ] 错误文案支持 zh-CN/en-US；产出迁移脚本与 API 文档

### F4. 认证
- 状态：TODO
- 目标：登录、登出、当前用户、公钥；JWT(HS256) + Redis 会话。
- 接口（`/api/v1`）：
  - POST `/login` → `{token}`；body `{username,password,code?,key?}`
  - POST `/verificationCode` → `{uuid,img}`（数学验证码，base64 图片）
  - POST `/logout` → 删除会话（幂等）
  - GET `/index?refresh=` → 当前用户（含 roles/orgs）；`refresh=true` 查库并写回 Redis
  - GET `/publicKey` → Base64(X509 DER) 公钥
- 规则：
  - 创建用户密码 bcrypt；登录用 `bcrypt.CompareHashAndPassword`。
  - 密码错 5 次 → 锁定 1 小时（`status=disabled`、`locked_until=+60m`）；锁定返回 `errors[0].data.unlockAt`（Unix 毫秒）。
  - 会话 TTL 默认 60 分钟（用户 `expireTimes` 覆盖；<0 永不过期）；剩余 <10 分钟时在 filter 中滑动续期。
  - JWT 只含 `login_user`(会话UUID)/`sub`/`iat`，不设 `exp`；`token-secret` ≥32 字节。
  - RSA：启动时在 MinIO 检查 `login-rsa-private.pem`/`login-rsa-public.pem`，缺失才生成；MinIO 不可达 → 启动失败；解密用 **OAEP(SHA-256)**，失败按明文。
  - 验证码：生成后存 Redis `platform:captcha:<uuid>`（TTL `captcha-ttl-seconds`，默认 300s）；**登录中的校验默认注释未启用**（`auth/service.go` 的 login）。
  - Trace：由 `generic.Trace()` 统一处理（`X-Trace-Id` 生成/回写 + 访问日志），与认证解耦；`auth.Auth()` 只管认证。
  - 上下文：认证成功后把**完整用户**（`identity/v1.UserVo`，含 roles/orgs）注入 `c.Request.Context()`；业务层用 `principal.UserID(ctx)` / `principal.FromContext(ctx)` 取用户信息（见 §9.4）。
  - 会话失效：改密/重置密码/删除用户/账户禁用/登录失败锁定后，按 userId 清理该用户全部会话（Redis 反向索引 `platform:user_sessions:<userId>`）。
- 约束：遵循 §3~§9；受保护接口（users/roles/orgs）挂鉴权中间件；`health`/login/logout/publicKey 放行。
- 错误码：`10030 login_expired(401)`、`10031 unauthorized(401)`、`10032 username_password_error(400)`、`10033 account_disabled(400)`、`10034 account_locked(400)`、`10035 login_failed(500)`。
- 验收标准：
  - [ ] 登录成功返回 token；密码错 400 + 业务码
  - [ ] 连续 5 次失败锁定 1h，返回 `unlockAt`（毫秒）；1h 后自动解锁
  - [ ] 无/过期 token 访问受保护接口 → 401 `login_expired`
  - [ ] 登出后原 token 失效
  - [ ] filter 滑动续期（剩余 <10min 重置 60min）
  - [ ] `index?refresh=true` 查库并写回 Redis；`false` 读 Redis
  - [ ] 验证码接口返回 `{uuid,img}`，答案存 Redis（TTL 300s）；登录校验默认注释未启用
  - [ ] 受保护请求响应头返回 `X-Trace-Id`，日志包含 traceId、校验结果与耗时
  - [ ] RSA 加密密码与明文均可登录；MinIO 不可达启动失败
  - [ ] 改密/重置密码/禁用/锁定后旧 token 失效

### F5. 审计日志
- 状态：TODO
- 目标：记录写操作的入参/出参，支持查询。
- 中间件（拆分）：
  - `generic.Trace()`：traceId（`X-Trace-Id`）生成/透传，访问日志读取同一 trace。
  - `auth.Auth()`：仅认证；审计从 `generic.TraceID` 取 trace。
  - `ginmw.Middleware()` + `auditapi.Middleware()`：捕获入参/出参并异步入库。
  - 顺序：`Trace → Auth → Audit → handler`。
  - platform 绑定 `app=IOT`，auth 接口同样记 `IOT`。
- 审计范围：仅 `POST/PUT/PATCH/DELETE`（含 login/logout）；`GET` 不记录。排除 `health`/`auth/publicKey`/`auth/verificationCode`。
- 落库（异步批量）：有界队列；worker 攒批后多行 INSERT（`audit.batch-size` 默认 100，上限 500；`audit.flush-interval-ms` 默认 1000）；满则丢弃并告警；批量失败重试一次，仍失败则逐条回退；失败不影响响应。
- 字段：`app`（由服务绑定，platform/auth 均为 `IOT`）、`traceId`、`userId/userName`（`principal`）、`operationType/operationDesc`（路由 `httpx.WithOperation`/`ginmw.WithOperation` 显式声明）、`method/path/query/ip/userAgent`、`requestBody/responseBody`、`status/resultCode/errorMsg`、`durationMs`。
- 入参/出参：仅 JSON；按 16KB 截断；敏感键（password/authKey/token/accessKey/secretKey/ak/sk/secret）脱敏为 `***`；非 JSON 记 `<non-json body omitted>`。
- 查询：`GET /api/v1/iot/auditLogs`（分页 + app/userId/path/method/resultCode/start/end）；`app` 可选、大写、单值。
- 保留：`audit.retention-days`（默认 180 天）；启动时及每 24h 清理过期日志，按 `audit.cleanup-batch-size` 分批删除。
- 验收标准：
  - [ ] 写操作落 `audit_logs`，GET 不落
  - [ ] app/traceId/userId/operationType/入参/出参/耗时/业务码正确
  - [ ] 敏感字段脱敏、16KB 截断
  - [ ] 异步不阻塞；队列满丢弃并告警
  - [ ] 批量 INSERT 生效；批量失败可重试并按需逐条回退
  - [ ] 登录/登出和业务审计 app=`IOT`
  - [ ] 审计查询分页与筛选可用
  - [ ] 超过保留期的日志被自动清理

### F6. 物模型管理
- 状态：TODO
- 目标：物模型的创建、分页查询、按 id 查询、更新、删除。
- 输入：字段见下表。
- 处理：
  1. 迁移脚本建 `thing_models` 表；
  2. 实体 `ThingModel`（内嵌 `apis.Meta`）；
  3. 单包 `pkg/platform/thingmodel/`（model/store/service/server/v1，数据类型见 `tsl/`）；
  4. 路由注册到 `pkg/platform/web`，文档到 `apis/platform/thingmodel.yaml`。
- 输出：5 个接口 + 迁移脚本 + API 文档。
- 字段：

  | 字段 | Go 类型 | 约束 |
  |---|---|---|
  | thingModelKey | string | 可选输入；格式 `^[a-zA-Z][a-zA-Z0-9_-]{3,63}$`，区分大小写，全局唯一；为空时后端生成 `tm_` + 小写 ULID；创建后不可修改 |
  | name | string | 必填，唯一，≤64 |
  | description | string | 可空，≤255 |
  | features | array | 特征定义列表；纯静态，结构见 `docs/thing-model.md`，identifier 可与属性重名 |

  > 基础字段来自 `apis.Meta`（见 §9.3），不要 `deleted` 列。
- 表结构：

  ```sql
  CREATE TABLE IF NOT EXISTS `thing_models` (
    `id`              CHAR(32)     NOT NULL COMMENT 'ULID',
    `version`         BIGINT       NOT NULL DEFAULT 0,
    `thing_model_key` VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT '物模型 Key，全局唯一，区分大小写',
    `name`            VARCHAR(64)  NOT NULL COMMENT '物模型名称',
    `description`     VARCHAR(255) NOT NULL DEFAULT '',
    `features`        JSON         NULL COMMENT '特征定义列表',
    `created_user`    VARCHAR(64)  NOT NULL DEFAULT '',
    `updated_user`    VARCHAR(64)  NOT NULL DEFAULT '',
    `created_time`    DATETIME     NOT NULL,
    `updated_time`    DATETIME     NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_thing_models_key` (`thing_model_key`),
    UNIQUE KEY `uk_thing_models_name` (`name`)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='物模型';
  ```

- 接口：
  - POST `/api/v1/iot/thingModels` → 200 + ThingModelVo（可带初始 `properties`、`features`）
  - GET `/api/v1/iot/thingModels` → 200 + `PageResp[ThingModelVo]`（keyword 匹配 name/key，返回 `propertyCount`）
  - GET `/api/v1/iot/thingModels/:id` → 200 + ThingModelDetailVo（含 `properties`、`features`）
  - PUT `/api/v1/iot/thingModels/:id` → 200 + ThingModelVo（仅 `name`/`description`）
  - DELETE `/api/v1/iot/thingModels/:id?cascade=false` → 200（同事务删除全部属性）
- 约束：
  - 遵循 §3~§9；DTO 放 `pkg/platform/thingmodel/v1`；对外一律 Vo，禁止返回实体。
  - `thingModelKey` 创建后固定，`PUT` 不接收该字段；用户传入的 key 冲突 → 400 `error.thing_model_key_exists`；自动生成冲突最多重试 3 次。
  - `name` 全局唯一，冲突 → 400 `error.thing_model_name_exists`。
  - 删除物模型与属性在同一事务内完成。
  - `cascade`（默认 false）为预留参数，后续与 thing 联动时生效；当前属性始终随物模型删除，非 `true` 的取值一律按 false 处理。
  - `createdUser`/`updatedUser` 取 `principal.UserID(ctx)`。
  - Service 导出方法首行 `defer logging.Trace("thingmodel.Xxx", ...)`。
- 验收标准：
  - [ ] 5 个接口按 §5.1 返回
  - [ ] 列表用 `PageQuery`/`PageResp`，`data` 含 total/page/pageSize/list 与 `propertyCount`
  - [ ] 未传 key 时生成 `tm_`+小写 ULID；格式错误 → 400 `error.thing_model_invalid`；重复 → 400 `error.thing_model_key_exists`
  - [ ] 创建后无法修改 `thingModelKey`
  - [ ] `name` 重复 → 400 `error.thing_model_name_exists`；id 不存在 → 400 `error.thing_model_not_found`
  - [ ] 删除物模型同事务清理属性；更新后 `version` +1、`ETag` 反映新值
  - [ ] 错误文案支持 zh-CN/en-US；产出迁移脚本与 API 文档

### F7. 物模型属性管理
- 状态：TODO
- 目标：属性的批量追加、单个更新、幂等批量删除。
- 输入：字段见下表。
- 处理：
  1. 迁移脚本建 `thing_model_properties` 表；
  2. 实体 `Property`（内嵌 `apis.Meta`）；
  3. 数据类型 `DataTypeDetail` 与归一化/校验（定义在 `pkg/platform/thingmodel/tsl/`，见下）；
  4. 路由注册到 `pkg/platform/web`，文档到 `apis/platform/thingmodel.yaml`。
- 输出：3 个接口 + 迁移脚本 + 字段级校验错误。
- 表结构：属性通过 `thing_model_id`（ULID）关联物模型，不使用 `thing_model_key`。

  ```sql
  CREATE TABLE IF NOT EXISTS `thing_model_properties` (
    `id`             CHAR(32)     NOT NULL COMMENT 'ULID',
    `version`        BIGINT       NOT NULL DEFAULT 0,
    `thing_model_id` CHAR(32)     NOT NULL COMMENT '所属物模型 ID',
    `identifier`     VARCHAR(64)  NOT NULL,
    `name`           VARCHAR(64)  NOT NULL,
    `data_type`      JSON         NOT NULL,
    `rw_flag`        VARCHAR(8)   NOT NULL DEFAULT 'r',
    `required`       TINYINT      NOT NULL DEFAULT 0,
    `description`    VARCHAR(255) NOT NULL DEFAULT '',
    `custom`         JSON         NULL,
    `sort`           INT          NOT NULL DEFAULT 0,
    `created_user`   VARCHAR(64)  NOT NULL DEFAULT '',
    `updated_user`   VARCHAR(64)  NOT NULL DEFAULT '',
    `created_time`   DATETIME     NOT NULL,
    `updated_time`   DATETIME     NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tmp_model_identifier` (`thing_model_id`, `identifier`),
    UNIQUE KEY `uk_tmp_model_name` (`thing_model_id`, `name`)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='物模型属性';
  ```

  > `id` 为聚簇主键；`(thing_model_id, identifier)` 与 `(thing_model_id, name)` 为模型内唯一索引，不同物模型可以使用相同的 identifier/name。

- 字段：

  | 字段 | Go 类型 | 约束 |
  |---|---|---|
  | thingModelId | string | 内部关联列，取 path 中的物模型 id；不对外暴露，不在请求体中接收 |
  | identifier | string | 必填，`^[A-Za-z][A-Za-z0-9_]*$`，≤64，不可为 `time`/`device_id`，不可为 SQL 关键字，创建后不可修改，同模型内唯一 |
  | name | string | 必填，字母开头，≤64，同模型内唯一 |
  | dataType | DataTypeDetail | 必填，见下 |
  | rwFlag | string | 枚举 `r`/`w`/`rw` |
  | required | bool | 默认 false |
  | description | string | 可空，≤255 |
  | custom | map[string]any | 可空 |
  | sort | int | 默认 0，维护展示顺序 |

  > 基础字段来自 `apis.Meta`（见 §9.3），不要 `deleted` 列。
- dataType（B 方案）：`type` 为判别字段，复杂规格各自独立。

  | type | 允许字段 | 必填校验 |
  |---|---|---|
  | int / long / float / double | unit、unitName、min、max | min ≤ max（可解析时） |
  | bool | trueValue、falseValue | 无 |
  | enum | enumSpec.values[] | values 非空，value 不重复 |
  | text | length | 正整数 |
  | time | 无 | 无 |
  | array | arraySpec.element、arraySpec.size | element 必填；size 可选，1~512 |
  | struct | structSpec.fields[] | fields 非空，field identifier 唯一 |

  ```go
  type DataTypeDetail struct {
      Type DataType `json:"type"` // int|long|float|double|bool|enum|text|time|array|struct
      Unit, UnitName, Min, Max *string
      TrueValue, FalseValue    *string
      Length                   *string
      EnumSpec   *EnumSpec   `json:"enumSpec,omitempty"`
      ArraySpec  *ArraySpec  `json:"arraySpec,omitempty"`
      StructSpec *StructSpec `json:"structSpec,omitempty"`
  }
  ```

  > 校验采用**宽松忽略**：与 `type` 不匹配的字段在解析时丢弃，落库只保存归一化后的结构；必填项仍严格校验。
- 接口：
  - POST `/api/v1/iot/thingModels/:id/properties` → 200（body 为属性数组，事务内全成功或全失败）
  - PUT `/api/v1/iot/thingModels/:id/properties/:identifier` → 200（按 identifier 更新；body 中 identifier 不一致则 400）
  - DELETE `/api/v1/iot/thingModels/:id/properties` → 200 + `{"deleted":n}`（body 为 identifier 数组，幂等，忽略不存在）
- 约束：
  - 遵循 §3~§9；DTO 放 `pkg/platform/thingmodel/v1`；对外一律 Vo。
  - 关联用 `thing_model_id`（主键 ULID），不用 `thing_model_key`：key 是业务标识/导入导出用，id 才是关系键；与其他模块的关联表（如 `user_roles`）保持一致。
  - `thing_model_id` 由 path 决定，不接受请求体传入；属性 Vo 不返回该字段，避免与 path 不一致。
  - 追加为批量且原子；批内与库内 `identifier`/`name` 重复均返回字段级错误。
  - `identifier` 不可修改；`PUT` 以 path 为准，body 不一致 → 400 `error.thing_model_invalid`。
  - 删除幂等，不存在的 identifier 不报错，返回实际删除数量。
  - 属性任何变更在同一事务内使 `thing_models.version + 1`。
  - 字段级错误通过 `errors[].field` 返回，如 `properties[2].dataType.arraySpec.element.type`。
- 验收标准：
  - [ ] 3 个接口按 §5.1 返回
  - [ ] 批量追加原子；批内/库内 identifier/name 重复均返回字段级错误
  - [ ] 单属性更新按 identifier 定位；body identifier 不一致 → 400
  - [ ] 批量删除幂等，返回实际删除数量，不存在的 identifier 不报错
  - [ ] dataType 归一化正确，enum/array/struct/数值范围校验完整，字段路径准确
  - [ ] 每次属性变更后主表 `version` +1
  - [ ] 错误文案支持 zh-CN/en-US；产出迁移脚本与 API 文档

### F8. 物模型导入
- 状态：TODO
- 目标：通过 JSON 整体替换物模型的名称、描述与属性。
- 输入：JSON 文档，结构为 `{thingModelKey, name, description, properties[]}`。
- 处理：
  1. 解析 JSON 为 `ImportThingModelReq`；
  2. 校验 `thingModelKey`（为空或与当前一致）、`name`（可选，需唯一）、`properties`（必填，可为空数组）；
  3. 在事务内更新名称/描述、替换属性与特征，主表 `version` +1。
- 输出：导入后的物模型详情 + 字段级校验错误。
- 接口：
  - PUT `/api/v1/iot/thingModels/:id/tsl` → 200 + ThingModelDetailVo
- 字段：

  | 字段 | 类型 | 约束 |
  |---|---|---|
  | thingModelKey | string | 可省略；非空时必须与当前物模型一致，key 不可修改 |
  | name | string | 可省略；非空时更新，需满足名称规则且全局唯一 |
  | description | string | 可省略；提供时更新，`""` 表示清空 |
  | properties | array | 必填；`[]` 表示清空全部属性；否则整体替换 |
  | features | array | 可选；省略保留原特征，提供时整体替换，`[]` 表示清空 |

- 约束：
  - 遵循 §3~§9；**仅支持 JSON 导入，不支持 XML 导入**。
  - `properties` 缺失 → 400 `error.thing_model_invalid`，字段 `properties`。
  - `thingModelKey` 不一致 → 400 `error.thing_model_invalid`，字段 `thingModelKey`。
  - JSON 语法错误 → 400 `error.thing_model_import_invalid`；语义校验失败 → 400 `error.thing_model_invalid` + `errors[].field`。
  - 名称/描述/属性在同一事务内更新，失败整体回滚。
- 验收标准：
  - [ ] JSON 导入成功，名称、描述、属性被更新，`thingModelKey` 保持不变
  - [ ] `thingModelKey` 不一致被拒绝
  - [ ] `properties` 缺失被拒绝；`[]` 可清空属性
  - [ ] 名称重复 → `error.thing_model_name_exists`
  - [ ] 校验失败返回字段级错误，事务不提交
  - [ ] 导入成功后主表 `version` +1

### F9. 物模型导出
- 状态：TODO
- 目标：导出物模型为 JSON。
- 输入：物模型 id。
- 处理：
  1. 查询物模型、属性（按 `sort`、`id` 排序）和特征；
  2. 组装导出结构；
  3. 序列化为 JSON。
- 输出：JSON 文件。
- 接口：
  - GET `/api/v1/iot/thingModels/:id/tsl` → 200（`application/json`）
- 约束：
  - 遵循 §3~§9；导出含 `thingModelKey`，导入时忽略。
  - 仅支持 JSON，不提供 XML/XSD。
- 验收标准：
  - [ ] 导出 JSON 包含 properties 与 features
  - [ ] JSON 往返一致
  - [ ] 不再提供 `/schema` 和 `?format=xml`

### F10. Thing 创建（批量）
- 状态：TODO
- 目标：基于物模型批量创建设备（Thing），自动生成凭证与 MQTT Topic。
- 输入：`thingModelId` + 设备数组，字段见下表。
- 处理：
  1. 迁移脚本建 `things` 表；
  2. 实体 `Thing`（内嵌 `apis.Meta`）；
  3. 单包 `pkg/platform/things/`（model/store/service/server/v1）；
  4. 路由注册到 `pkg/platform/web`，文档到 `apis/platform/things.yaml`。
- 输出：批量创建接口 + 迁移脚本 + API 文档。
- 表结构：

  ```sql
  CREATE TABLE IF NOT EXISTS `things` (
    `id`              CHAR(32)     NOT NULL COMMENT 'ULID',
    `version`         BIGINT       NOT NULL DEFAULT 0,
    `thing_model_id`  CHAR(32)     NOT NULL COMMENT '所属物模型 ID',
    `device_name`     VARCHAR(64)  NOT NULL COMMENT '设备名称，同模型内唯一',
    `device_key`      VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT '设备 Key，全局唯一',
    `device_secret`   VARCHAR(64)  NOT NULL COMMENT '设备密钥',
    `token`           VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT 'HTTP 上报 Token，全局唯一',
    `up_topic`        VARCHAR(255) COLLATE utf8mb4_bin NOT NULL COMMENT '上行 Topic，iot/<deviceKey>/up',
    `down_topic`      VARCHAR(255) COLLATE utf8mb4_bin NOT NULL COMMENT '下行 Topic，iot/<deviceKey>/down',
    `device_desc`     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '设备描述',
    `features`        JSON         NULL COMMENT '设备静态特征值',
    `image`           VARCHAR(255) NOT NULL DEFAULT '' COMMENT '设备图片 MinIO key',
    `mqtt_client_id`  VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT '默认等于 id',
    `mqtt_username`   VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT '默认等于 deviceKey',
    `mqtt_password`   VARCHAR(64)  NOT NULL COMMENT '默认等于 deviceSecret',
    `enable`          TINYINT      NOT NULL DEFAULT 1 COMMENT '1 启用，0 禁用',
    `created_user`    VARCHAR(64)  NOT NULL DEFAULT '',
    `updated_user`    VARCHAR(64)  NOT NULL DEFAULT '',
    `created_time`    DATETIME     NOT NULL,
    `updated_time`    DATETIME     NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_things_device_key` (`device_key`),
    UNIQUE KEY `uk_things_token` (`token`),
    UNIQUE KEY `uk_things_up_topic` (`up_topic`),
    UNIQUE KEY `uk_things_down_topic` (`down_topic`),
    UNIQUE KEY `uk_things_model_name` (`thing_model_id`, `device_name`),
    UNIQUE KEY `uk_things_mqtt_client_id` (`mqtt_client_id`),
    UNIQUE KEY `uk_things_mqtt_username` (`mqtt_username`),
    KEY `idx_things_mqtt_password` (`mqtt_password`)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='设备 Thing';
  ```

  > 在线状态、上次上报时间不落 MySQL，由 Redis 全局活跃时间索引提供（见 F11）。

- 字段：

  | 字段 | Go 类型 | 约束 |
  |---|---|---|
  | deviceName | string | 必填，≤64，同一物模型内唯一 |
  | deviceKey | string | 可选；`^[a-zA-Z][a-zA-Z0-9_-]{3,63}$`，全局唯一；为空时生成 `dk_` + 16 位随机小写字母数字 |
  | deviceDesc | string | 可空，≤255 |
  | features | object | 可空；key 必须是物模型已定义特征，值只校验合法 JSON |
  | image | string | 可空，≤255；非空时必须以 `images/` 开头，不校验 MinIO 是否存在 |

  > `id`、`deviceSecret`、`token`、`upTopic`、`downTopic`、`mqtt_*`、`enable` 由后端生成，基础字段来自 `apis.Meta`。
- 生成规则：
  - `id`：ULID。
  - `deviceSecret`：32 字节 `crypto/rand` → hex 64 位。
  - `token`：32 字节 `crypto/rand` → hex 64 位，全局唯一，冲突重试。
  - `upTopic`：`iot/` + `deviceKey` + `/up`（上行），不接受入参覆盖、不可修改。
  - `downTopic`：`iot/` + `deviceKey` + `/down`（下行），不接受入参覆盖、不可修改。
  - `mqtt_client_id = id`，`mqtt_username = deviceKey`，`mqtt_password = deviceSecret`。
  - `enable = 1`，创建时不接受入参。
- 接口：
  - POST `/api/v1/iot/things` → 200 + `[]ThingDetailVo`（含完整凭证）
- 约束：
  - 遵循 §3~§9；DTO 放 `pkg/platform/things/v1`；对外一律 Vo。
  - 单批 1~100 条，单事务写入，任一失败整体回滚。
  - `thingModelId` 必须存在，否则 400 `error.thing_model_not_found`。
  - 批内与库内 `deviceName` 重复 → 字段级错误 `things[i].deviceName`；`deviceKey` 重复 → `things[i].deviceKey`。
  - `features` 的 key 未在物模型定义 → `things[i].features.<key>`。
  - 字段级错误通过 `errors[].field` 返回。
  - `createdUser`/`updatedUser` 取 `principal.UserID(ctx)`。
  - Service 导出方法首行 `defer logging.Trace("things.Xxx", ...)`。
- 验收标准：
  - [ ] 批量创建成功，返回完整凭证与 mqtt 字段
  - [ ] 未传 deviceKey 时生成 `dk_`+16 位随机；重复 → 400 字段级错误
  - [ ] deviceName 同模型内重复 → 400 字段级错误
  - [ ] thingModelId 不存在 → 400 `error.thing_model_not_found`
  - [ ] 数量超限或空数组 → 400
  - [ ] 事务失败整体回滚
  - [ ] 错误文案支持 zh-CN/en-US；产出迁移脚本与 API 文档

### F11. Thing 查询
- 状态：TODO
- 目标：分页查询与按 id 查询设备，在线状态来自 Redis 全局活跃时间索引。
- 输入：分页参数 + `keyword`。
- 处理：
  1. 查 MySQL 得到设备页；
  2. 用一次 `ZMSCORE iot:thing:last-active id1 id2 ... idN` 取当前页的上次上报时间；
  3. 计算 `onlineStatus` 与 `lastActiveTime` 后返回。
- 输出：分页接口 + 详情接口。
- Redis 活跃时间索引（HTTP 与 MQTT 共用同一个结构）：

  ```
  key: iot:thing:last-active
  type: ZSET
  member: thing id
  score: 最后上报数据时间（Unix 毫秒）
  ```

  - HTTP 上报：`ZADD iot:thing:last-active GT CH <score> <id>`
  - MQTT 上报（含网关代传子设备）：对每个子设备 `ZADD ... GT CH <score> <subThingId>`
  - `score = min(dataTime, now)`，防止设备时钟超前。
  - 使用 `GT`，只允许时间前进，避免网关补传历史数据把 score 改小。
  - MQTT 断开**不删除**成员，由 TTL 判定离线。
  - 删除设备：`ZREM iot:thing:last-active <id>`。
  - 不做 retention 清理，成员数受 thing 总数约束。
  - 设备影子 `iot:thing:shadow:<id>` 同时保存 `lastReportTime`（与 score 相同的 clamped 值），以及 `reported/desired/version`；影子不参与在线判定。

  在线判定：

  ```
  online = enable == 1 && score != null && now - score <= onlineTTL
  ```

  `onlineTTL` 来自配置 `things.online-ttl-seconds`，默认 60。
- 接口：
  - GET `/api/v1/iot/things?page=&pageSize=&keyword=` → 200 + `PageResp[ThingVo]`
  - GET `/api/v1/iot/things/:id` → 200 + ThingDetailVo
- 约束：
  - 遵循 §3~§9；分页用 `PageQuery`/`PageResp`。
  - `keyword` 只模糊匹配 `device_name`、`device_key`；按 `created_time DESC`。
  - **列表不返回** `deviceSecret`、`token`、`mqttPassword`；详情与创建返回完整值。
  - 列表与详情返回 `onlineStatus`、`lastActiveTime`；索引中不存在时 `onlineStatus=0`、`lastActiveTime=null`。
  - `image` 仅在详情返回。
  - 列表查询对当前页固定只发 1 条 `ZMSCORE`，命令数与页大小无关。
  - Redis 不可用时降级为离线并记录日志，不影响列表返回。
  - `enable=0` 时强制返回离线。
  - MQTT 断开不单独标记离线，统一按“最后数据时间 + TTL”判定。
- 验收标准：
  - [ ] 分页返回 total/page/pageSize/list
  - [ ] keyword 同时匹配 deviceName、deviceKey
  - [ ] 列表不含 deviceSecret/token/mqttPassword
  - [ ] 在线状态与 lastActiveTime 来自 `iot:thing:last-active`，计算正确
  - [ ] 每页只发 1 条 `ZMSCORE`
  - [ ] score 使用 `min(dataTime, now)` 且只增不减
  - [ ] enable=0 返回离线
  - [ ] Redis 异常时列表仍可用

### F12. Thing 更新
- 状态：TODO
- 目标：更新设备的可变字段，复用同一接口切换启用/禁用。
- 输入：字段见下表，全部可选。
- 处理：
  1. 查询设备是否存在；
  2. 校验唯一性；
  3. 更新可变字段并 `version` +1。
- 输出：更新接口。
- 字段：

  | 字段 | Go 类型 | 约束 |
  |---|---|---|
  | deviceName | *string | ≤64，同模型内唯一 |
  | deviceDesc | *string | ≤255 |
  | mqttClientId | *string | 非空、≤64，全局唯一 |
  | mqttUsername | *string | 非空、≤64，全局唯一 |
  | mqttPassword | *string | 非空、≤64，不要求唯一 |
  | enable | *int | 仅 0 或 1 |
  | features | *map[string]any | 提供时整体替换，`{}` 清空；key 必须在物模型中定义 |
  | image | *string | 提供时更新，`""` 清空；≤255，非空时以 `images/` 开头 |

  > `deviceKey`、`deviceSecret`、`token` 不可修改。
- 接口：
  - PUT `/api/v1/iot/things/:id` → 200 + ThingDetailVo
- 约束：
  - 遵循 §3~§9；只更新传入字段，nil 表示不修改。
  - 唯一性冲突返回 400 对应错误码；字段级错误带 `errors[].field`。
  - `upTopic`、`downTopic` 不可通过更新接口修改。
  - `enable` 非 0/1 → 400 字段级错误。
  - `updatedUser` 取 `principal.UserID(ctx)`。
- 验收标准：
  - [ ] 只更新传入字段，未传字段保持不变
  - [ ] deviceName/topic/mqttClientId/mqttUsername 唯一性校验生效
  - [ ] enable 仅接受 0/1
  - [ ] deviceKey/deviceSecret/token 无法通过该接口修改
  - [ ] 更新后 version +1

### F13. Thing 删除
- 状态：TODO
- 目标：批量删除设备，`cascade` 为后续关联数据预留。
- 输入：id 数组 + `cascade`。
- 处理：
  1. 单事务按 id 批量删除；
  2. 删除成功后从 `iot:thing:last-active` 中 `ZREM` 对应 id。
- 输出：删除接口。
- 接口：
  - DELETE `/api/v1/iot/things?cascade=false`，body 为纯数组 `["id1","id2"]` → 200 + `{"deleted":n}`
- 约束：
  - 遵循 §3~§9；`cascade` 默认 false，非 `true` 的取值一律按 false 处理。
  - 幂等：不存在的 id 忽略，返回实际删除数量。
  - 当前只删除 things 并清理活跃时间索引；`cascade` 预留给后续影子、时序数据清理。
  - 单事务批量删除；`ZREM` 失败只记录日志，不影响删除结果。
- 验收标准：
  - [ ] 批量删除成功，返回 deleted 数量
  - [ ] 不存在的 id 不报错
  - [ ] cascade 参数不合法时按 false 处理
  - [ ] 删除后设备不可查询

### F14. 设备认证（EMQX）
- 状态：TODO
- 目标：为 EMQX HTTP 认证器提供 clientid/username/password 认证，并返回属性与指令 Topic 的 ACL。
- 输入：EMQX 认证请求，字段见下表。
- 处理：
  1. 校验共享密钥 Header；
  2. 按 `mqtt_client_id + mqtt_username + mqtt_password + enable=1` 查询 things，并 join thing_models 取 `thing_model_key`；
  3. 命中则返回 allow、client_attrs 与两条 ACL；否则 deny。
- 输出：EMQX 裸 JSON（不使用平台 `Result` 包装）。
- 接口：
  - POST `/api/v1/iot/thingsAuth`（不挂平台 Auth / Audit 中间件）
- 字段：

  | 字段 | JSON | 说明 |
  |---|---|---|
  | clientid | `clientid` | 对应 `things.mqtt_client_id` |
  | username | `username` | 对应 `things.mqtt_username` |
  | password | `password` | 对应 `things.mqtt_password` |
  | client_attrs | `client_attrs` | 预留，忽略 |
  | cn | `cn` | 证书认证预留，忽略 |

- 响应：

  ```json
  {
    "result": "allow",
    "is_superuser": false,
    "client_attrs": {
      "thing_id": "...",
      "thing_model_id": "...",
      "thing_model_key": "...",
      "device_key": "...",
      "device_name": "..."
    },
    "acl": [
      { "permission": "allow", "action": "all", "topic": "iot/<deviceKey>/up", "qos": [0,1,2], "retain": false },
      { "permission": "allow", "action": "all", "topic": "iot/<deviceKey>/down", "qos": [0,1,2], "retain": false }
    ]
  }
  ```

  拒绝：

  ```json
  { "result": "deny", "is_superuser": false }
  ```

- 共享密钥：
  - 配置 `things.auth-secret`（默认空，空表示不校验）。
  - Header `X-Things-Auth-Secret`，使用常量时间比较。
  - 非空时长度必须 ≥16。
- 约束：
  - 遵循 §3~§9；该接口输出 EMQX 协议格式，不套 `response.WriteOK`。
  - 任一凭据为空、查不到、`enable=0`、共享密钥不匹配 → HTTP 401 + `result=deny`。
  - EMQX 会把 401 当作 `ignore` 继续认证链，与现有 Java 行为一致。
  - DB 异常 → 记录日志并返回 401 deny（fail-closed）。
  - 日志禁止打印 password。
- 验收标准：
  - [ ] 正确三元组返回 200 + allow
  - [ ] client_attrs 含 thing_id/thing_model_id/thing_model_key/device_key/device_name
  - [ ] ACL 含 up/down 两条，`allow` + `all`
  - [ ] 凭据错误、空值、enable=0 返回 401 + deny
  - [ ] 共享密钥缺失/错误返回 401 + deny
  - [ ] DB 异常 fail-closed
  - [ ] 日志不含密码

### F15. EMQX 消息回调
- 状态：TODO
- 目标：接收 EMQX `message.publish` 回调，按配置直接处理或经 Kafka 缓冲后批量处理。
- 输入：EMQX 原始 `EmqxPubEventMessage`，字段与 EMQX webhook 对齐。
- 处理：
  1. 校验共享密钥 Header；
  2. 仅处理 `event == "message.publish"` 且 `client_attrs.device_key` 非空的事件，其余返回 204；
  3. 把事件转换为平台 `DeviceMessage`，`things.message.kafka-enabled=true` 时写入 Kafka：
     `key = deviceKey`，保证同一设备进入同一分区；
  4. 后台 consumer 批量消费，按 `DeviceMessage.Kind` 路由到 Handler；
  5. Kafka 关闭时，入口直接调用同一个 Router。
- 输出：EMQX 回调接口 + 可选的 Kafka 生产者/消费者。
- 接口：
  - POST `/api/v1/iot/thingsPubMessage`（与 `/thingsAuth` 同组，不挂平台 Auth/Audit）
- 分类规则：由 payload 的 `type` 字段决定，`type` 必填。
  - `property` → property
  - `event` → event
  - `command_reply` → command_reply
  - `command` → 下行类型，入站忽略
  - 缺失/其他 → 不处理并记录上报日志

- 处理抽象：

  ```go
  type DeviceMessage struct {
      Kind          MessageKind
      Topic         string
      DeviceKey     string
      ThingID       string
      ThingModelID  string
      ThingModelKey string
      DeviceName    string
      Payload       string
      Timestamp     int64
      MessageID     string
      Source        string
      QoS           int
  }

  type MessageHandler interface {
      Handle(ctx context.Context, msg *DeviceMessage) error
  }

  type MessageRouter struct {
      handlers map[MessageKind]MessageHandler
  }
  ```

  - `MessageRouter` 按 `Kind` 查找 `PropertyHandler` / `EventHandler` / `CommandReplyHandler`。
  - Handler 负责解码，业务逻辑放在独立的 `PropertyProcessor` / `EventProcessor` / `CommandReplyProcessor`。
  - 生产环境由 `web` 组装 `DefaultPropertyProcessor`；`NewDefaultMessageRouter()` 仅用于测试和兜底占位。
  - 未注册的 type 记录日志后忽略。
- 配置：

  ```yaml
  things:
    auth-secret: ""
    message:
      kafka-enabled: false
      kafka:
        brokers: []
        topic: "iot.things.pub-message"
        group-id: "harns-platform-things-message"
        batch-size: 100
        max-wait-ms: 1000
  ```

- 约束：
  - 遵循 §3~§9；该接口输出/响应遵循 EMQX webhook 约定，不使用平台 `Result` 包装。
  - Kafka 中保存平台 `DeviceMessage`，不包含 EMQX 专用字段。
  - consumer 逐条处理，单条失败记录日志并继续；批次结束后提交 offset（与现有 Java 行为一致）。
  - Kafka 生产失败返回 500，让 EMQX 重试。
  - 开关仅在启动时读取，不支持运行时切换。
  - 日志禁止打印完整 payload。
- 验收标准：
  - [ ] 非 `message.publish` 返回 204
  - [ ] 缺少 `client_attrs.device_key` 返回 204
  - [ ] Kafka 开启时按 deviceKey 分区写入 DeviceMessage
  - [ ] consumer 批量消费并正确分类到 property/command_reply/event
  - [ ] Kafka 关闭时同步调用 Router
  - [ ] 处理失败返回 500
  - [ ] 共享密钥不匹配返回 401

### F16. HTTP 设备数据上报
- 状态：TODO
- 目标：设备通过 HTTP 上报属性数据，使用创建设备时生成的 token 认证，并复用同一 Router。
- 输入：Header `X-Things-Token` + 属性原始 JSON Body。
- 处理：
  1. 从 `X-Things-Token` 取 token，先查 Redis 缓存，未命中再查 MySQL 并回填；
  2. 校验 body 为合法 JSON；
  3. 更新在线状态（`iot:thing:last-active` + `iot:thing:shadow:<id>.lastReportTime`）；
  4. 构造 `DeviceMessage`（`Kind` 取自 payload `type`，缺失则拒绝；`Source=http`；`Topic=upTopic`）；
  5. 交给同一个 `MessageSink`（Kafka 开启则投递 Kafka，否则直接 Dispatch）。
- 输出：HTTP 上报接口。
- 接口：
  - POST `/api/v1/iot/thingsData`（不挂平台 Auth/Audit）
- 认证与缓存：
  - key：`iot:thing:token:<sha256(token)>`，value 为非敏感 `ThingIdentity`；
  - TTL 30 分钟，命中时通过 `GETEX` 刷新（滑动过期）；
  - 更新设备、删除设备时直接 `DEL` 对应 key，下次重新查库；
  - Redis 异常时回落 MySQL。
- 响应：平台风格 `{"code":200,"message":"success"}`。
- 错误：
  - token 为空/无效 → 401 `error.unauthorized`；
  - body 空或非法 JSON → 400 `error.malformed_json` / `error.request_body`；
  - 处理/Kafka 失败 → 500 `error.internal`。
- 约束：
  - 遵循 §3~§9；token 不得出现在日志中。
  - 在线状态更新失败只记录日志，不影响上报结果。
  - 按 payload `type` 分类：`property`、`event`、`command_reply`；`command`、缺失或未知类型返回 400。
- 验收标准：
  - [ ] token 缺失/错误返回 401
  - [ ] 合法 token + 合法 JSON 返回 200
  - [ ] 非法 JSON 返回 400
  - [ ] 缓存命中不查库，命中后 TTL 刷新
  - [ ] 更新/删除设备后缓存 key 被删除
  - [ ] 上报后 ZSET 与影子 `lastReportTime` 更新
  - [ ] Kafka 开启时走 Kafka，关闭时直接调用 Router
  - [ ] 日志不含 token

### F17. 上报数据结构
- 状态：TODO
- 目标：定义设备上报的统一信封与各类型 payload，供 MQTT / HTTP 共用。
- 对外文档：`docs/device-communication.md`（唯一事实来源）。
- 已实现类型：`property`、`event`、`command_reply`。
- 下行类型：`command`（结构已定义，下发待实现）。
- 代码：
  - `pkg/platform/things/report.go`：`InboundMessage`、`PropertyReport`、`EventReport`、`CommandReply`、`Command`、`DecodeInbound`、`DecodeCommand`
  - `pkg/platform/things/device_message.go`：`ResolveInboundKind`
- 约束：
  - `type` 必填，缺失不处理并记录上报日志。
  - `time` 为 Unix 毫秒；缺失或 `<= 0` 时使用服务器当前时间。
  - `collectionId` 预留，允许为空。
  - `property`、`eventIdentify` 值保留原始 JSON（`json.RawMessage`）。
  - `eventIdentify` 每个上报项必须非空。
  - `command` 的 `messageId` 必填且唯一，`property` 非空。
  - `command_reply` 的 `messageId` 必填，`property` 非空，`result` 必须出现，`time` 为毫秒。
  - 批量条数不限制，受请求体大小约束。
  - 重复数据由时序库按时间覆盖，平台不去重。
- property 处理：
  - 属性定义按 `thingModelKey` 获取，Redis key `iot:thingmodel:properties:<modelKey>`，TTL 24 小时；未命中回落 MySQL 并回填。
  - 物模型属性增删改、导入、删除模型后主动删除缓存。
  - `required` 属性必须存在且非 null；未知 identifier 忽略。
  - 值按 `dataType` 转换校验。
  - 校验失败写 `device_property_log_<modelKey>`（`success=false`），不写时序数据，返回 nil 不触发重试。
  - 全部通过后提交时序数据，再写 `success=true` 日志。

### F18. 图片上传
- 状态：TODO
- 目标：提供通用的图片上传接口，上传到 MinIO 并返回 object key，供设备 `image` 等字段使用。
- 输入：multipart/form-data，字段 `file`。
- 处理：
  1. 校验登录态；
  2. 限制大小 `files.max-image-bytes`（默认 10MiB）；
  3. 用 `http.DetectContentType` 检测真实类型；
  4. 允许 `image/jpeg`、`image/png`、`image/webp`、`image/gif`；
  5. 生成 key `images/<yyyy>/<mm>/<ulid>.<ext>`；
  6. 确保 bucket 存在后上传 MinIO。
- 输出：`{"key":"images/..."}`。
- 接口：
  - POST `/api/v1/iot/files`（需要平台登录态，走审计）
- 约束：
  - 上传路由使用独立 group，BodyLimit 为 `files.max-image-bytes`，不受全局 1MB 限制。
  - 不提供删除接口；替换设备图片不删除旧 object。
  - 返回的 key 由客户端写入 `things.image`。
  - `things.image` 只校验长度和 `images/` 前缀，不查询 MinIO。
- 错误码：
  - `10055 file_too_large`
  - `10056 file_type_invalid`
  - `10057 file_upload_failed`
- 配置：

  ```yaml
  files:
    max-image-bytes: 10485760
  ```

- 验收标准：
  - [ ] jpeg/png/webp/gif 上传成功并返回 key
  - [ ] 非图片类型返回 400 `error.file_type_invalid`
  - [ ] 超过 10MiB 返回 400 `error.file_too_large`
  - [ ] 上传路由使用独立 BodyLimit
  - [ ] key 格式为 `images/yyyy/mm/ulid.ext`
  - [ ] 详情返回 `image`，列表不返回
  - [ ] `image` 非空时必须以 `images/` 开头

### F19. EMQX Publish Service
- 状态：TODO
- 目标：提供共享的 EMQX HTTP API 客户端，用于反控时向设备下行 topic 发布命令。
- 包位置：`pkg/emqx/`（顶层共享包，其他模块可直接依赖）。
- 接口：
  - `POST {emqx.api-url}/api/v5/publish`
  - 认证：HTTP Basic，用户名 `emqx.ak`，密码 `emqx.sk`
  - body：`{"topic":"...","payload":"...","qos":1}`
- Service：

  ```go
  type PublishRequest struct {
      Topic   string
      Payload string
      QoS     int
  }

  func NewService(cfg Config) *Service
  func (s *Service) Publish(ctx context.Context, req PublishRequest) error
  ```

- 配置：

  ```yaml
  emqx:
    api-url: "http://127.0.0.1:18083"
    ak: ""
    sk: ""
    broker: "127.0.0.1:1883"
    timeout-ms: 10000
  ```

  - `broker` 本期保留，不参与 publish，也不做校验。
  - `ak` / `sk` 必须成对出现；`ak`/`sk` 不得打日志。
- 约束：
  - 遵循 §3~§9；不依赖 platform 业务包。
  - `topic` 非空，`qos` 只允许 0/1/2。
  - 非 2xx 返回错误，包含状态码和响应摘要。
  - 在组合根创建并注入 `things.Service`，供后续反控使用。
- 验收标准：
  - [ ] 正确调用 `/api/v5/publish`，Basic Auth 正确
  - [ ] topic/payload/qos 正确
  - [ ] 非法 topic/qos 返回错误
  - [ ] 非 2xx 返回错误
  - [ ] `sk` 不出现在日志中

## 11. 任务提示词模板（每次 vibe coding 用）

> 连同本文件发给 AI，明确「只做某一个序号」。

```
请先阅读 pkg/platform/platform.md 与涉及的通用包源码，然后实现 F<序号>。

范围：
- 只改动：<允许改的文件/目录>
- 不要改动：gateway 模块、通用层语义、其他功能

要求：
- 遵守 platform.md §3~§9（分层、响应、错误码/i18n、配置、日志、存储）
- 复用现有框架：response.WriteOK/WriteError、i18n、logging、config.Config
- 新增错误码按 §6 同时补 constants.go、messageIDs、locales
- 新增配置项按 §7 同步 options 与 etc/platform/config.yaml

验收：
- 满足 F<序号> 的验收标准
- go build ./... 通过；gofmt 干净
- 有接口时附请求/响应示例

请先给出实现计划与改动文件清单，再动手写代码。
```

## 12. 禁止事项

- 12.1 引入 klog/component-base 等 k8s 日志链路（日志统一 `pkg/logging`）。
- 12.2 业务层返回裸对象（必须包 `Result`）。
- 12.3 硬编码文案、魔法数、明文密钥。
- 12.4 破坏 `code===200?data:errors` 约定。
- 12.5 通用层 import 模块层。

## 13. 交付自检清单

- [ ] 功能序号改为 DONE
- [ ] `go build ./...` 通过、`gofmt -l` 无输出
- [ ] 新增错误码/文案/配置齐全
- [ ] Service 层方法入口/出口已加 `logging.Trace`（见 §8.4）
- [ ] 有接口时附请求/响应示例
- [ ] 未触碰禁止事项
