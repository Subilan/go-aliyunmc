---
title: 论坛后端
description: 论坛模块的数据模型、权限、内容管线、文件存储与 API 摘要。
---

# 论坛后端

论坛运行在同一进程内，前缀为 `/forum-v1`，管理后台为 `/forum-v1/admin`。业务表全部使用
`forum_` 前缀，与 lab 共用 `users` 表、数据库、配置与迁移版本序列。

代码位于 `internal/modules/forum/`：

```
models/      GORM 实体
content/     sanitize、纯文本、标签、楼层引用、提及解析
uploads/     本地图片存储、解码校验、静态文件路径
perms/       权限点目录、用户权限节点、sub/dom/perm Casbin 模型
session/     forum_session 与 forum_admin_session
mid/         Auth / OptionalAuth / AdminAuth / RequirePermission / 限流
dao/         列表、计数与批量查询
service/     事务与业务规则
dto/         API JSON 契约与组装
routes/      public / self / admin 路由
settings/    站点设置默认值、缓存与更新
```

## 数据模型

论坛表包括：`forum_profiles`、`forum_boards`、`forum_threads`、`forum_replies`、
`forum_tags`、`forum_thread_tags`、`forum_thread_likes`、`forum_reply_likes`、
`forum_user_likes`、`forum_board_follows`、`forum_notifications`、`forum_uploads`、
`forum_user_permissions`、`forum_settings`、`forum_audit_logs`、`forum_login_records`。

关键约定：

- 共享 `users.username` 是唯一登录凭据与 @ 标识；`forum_profiles.nickname` 可选、可重复、可修改。
- 板块 `slug` 创建后不可修改，软删除后仍占用。
- 跟帖 `floor` 稳定：在事务内锁定主题行后取 `MAX(floor)+1`，删除后保留占位，不重排。
- 回复数、点赞数、关注数由聚合查询派生，不维护缓存列；列表接口使用批量聚合避免 N+1。
- 通知去重依赖唯一索引 `(type, recipient_id, actor_id, thread_id, reply_id)`。
- 上传文件只保存随机 key，不使用原始文件名。

迁移文件：`internal/common/migrations/mysql/20261118000000_add_forum.*.sql`。
DEV 启动时 `store.AutoMigrate(modules.Entities()...)` 直接按模型补表；测试通过
`testutil.OpenMySQLTestDB()` 应用同一套版本化迁移。

## 会话

| 用途 | cookie | Path | Manager |
|---|---|---|---|
| 论坛前台 | `forum_session` | `/forum-v1` | `forum/session.Manager` |
| 管理后台 | `forum_admin_session` | `/forum-v1/admin` | `forum/session.AdminManager` |

两套 cookie 独立，管理员与普通用户可以在同一浏览器分别登录不同账号。`main.go` 加载配置后
对 lab、forum、后台三个 manager 统一设置 cookie `Secure`（本地 TLS 或 `secure_cookie=true` 时）。
登录成功后会轮换 session token，避免会话固定。

## 权限

权限点是代码常量，定义在 `perms/catalog.go`，按用户直接授予（`forum_user_permissions`），
不存在角色实体。Casbin 使用论坛自持的模型：

```
r = sub, dom, perm
p = sub, dom, perm
m = r.sub == p.sub && (p.dom == "*" || p.dom == r.dom) && r.perm == p.perm
```

- 全局请求 `dom="*"` 只匹配全局授权。
- 板块请求 `dom="board:<id>"` 同时匹配全局授权与该板块授权。
- `users.role == superuser` 始终通过（破窗）。
- 新用户首次登录时授予默认自助权限；`forum_profiles.permissions_initialized` 防止管理员撤销
  授权后又被自动补回。
- 管理后台登录门槛为 `admin.access`（全局或任一板块）。拥有板块级 `admin.access` 的管理员
  只能看到授权板块的内容。
- 授权与撤销只能操作操作者自己持有的权限节点；超级用户不受此限制。
- 不允许封禁自己；不允许撤销自己最后一个全局 `admin.access` / `permission.manage`
  （当系统中没有其他非超级用户持有时）。

权限变更事务提交后调用 `perms.Reload()` 原子替换 enforcer，已登录 session 无需重新登录。

## 内容管线

所有主题/跟帖正文写入都经过 `content` 包：

```
原始 HTML
  → 大小 / 图片数量 / 提及数量校验
  → 拒绝 data: 图片
  → bluemonday 白名单 sanitize
  → 提及 chip 规范化（无效提及降级为文本）
  → 纯文本 content_text
  → 标签提取（主题）
  → 楼层引用解析（跟帖）
  → 事务写库
```

chip 契约：

- 楼层引用：`<span class="reply-reference" data-tag-type="reply" data-floor="3">#3</span>`
- 提及：`<span class="mention" data-tag-type="mention" data-user-id="3">@username</span>`

服务端只依赖 `data-tag-type`，`class` 仅用于样式。提及 chip 的文本会按 `data-user-id`
重建为当前 `@username`；不存在、已删除、已封禁或无论坛资料的用户会被降级为纯文本。

图片只接受外链 `http(s)` 或站内 `/forum-v1/files/` 地址；`data:` 图片直接返回 400。
单篇图片数、提及数、正文字节上限来自 `[forum]` 配置。

## 文件上传

- 目录：`[forum] upload_dir`，留空为 `$STATE_DIR/state/forum/uploads`。
- key 形如 `2026/11/<32 hex>.<jpg|png|webp>`。
- 上传校验：JPEG/PNG/WebP、单边 ≤ 8192、总像素 ≤ `max_image_pixels`；
  JPEG/PNG 重新解码编码以剥离元数据，WebP 原样保存。
- 读取：`GET /forum-v1/files/*key`，随机 key 不可猜，响应 `immutable` 缓存头并支持 Range。
- 头像/背景通过 `PUT /me/avatar`、`PUT /me/background` 绑定上传 ID；未设置头像时使用
  `default_avatar_url_template`（`{username}` 占位）。
- 每用户上传占用受 `user_quota_bytes` 限制（默认 256 MiB），超限返回 413。
- 后台删除上传前检查用户资料与正文引用，被引用返回 409。

## 通知

类型：`reply`、`like_thread`、`like_reply`、`mention`。

- 回复主题通知主题作者；自回复不通知。
- 点赞通知内容作者；自赞不通知；取消点赞删除对应通知。
- 提及通知被提及者；同一内容已向主题作者发送 `reply` 通知时不再重复发送 `mention`。
- 编辑内容同步通知摘要，只为新增提及补写通知；移除提及不删除历史通知。
- 删除主题/跟帖清理对应通知；恢复内容不恢复通知。
- 未读数走 `GET /notifications/unread-count`，客户端轮询即可。

## API 摘要

成功：`{"data": ...}`；失败：`{"details": "..."}`。JSON 字段 camelCase，时间为 RFC3339 UTC，
分页为 `{items,total,page,pageSize}`，前台默认 `20/页`、最大 `50`，后台最大 `100`。
用户主页中 `likeCount` 是内容获赞数，`profileLikeCount` 是主页收到的点赞数。

### 公开读

`GET /site`、`GET /boards`、`GET /boards/:slug`、`GET /threads`、`GET /pinned-threads`、
`GET /threads/:id`、`GET /threads/:id/replies`、`GET /recent-replies`、`GET /users/:id`、
`GET /users/:id/threads`、`GET /users/:id/replies`、`GET /search`、`GET /tags`、
`GET /files/*key`。

站点设置 `allow_guest_read=false` 时，除 `/site`、`/auth/*` 与 `/files/*` 外的公开读接口要求登录。

### 认证与自助

`POST /auth/login`、`POST /auth/logout`、`GET /auth/me`；
`PATCH /me/profile`、`PUT /me/avatar`、`PUT /me/background`、`PUT /me/avatar-shape`；
`GET /users/search`；`PUT/DELETE /users/:id/like`；`PUT/DELETE /boards/:slug/follow`。

### 内容

`POST /threads`、`PATCH /threads/:id`、`DELETE /threads/:id`、`PUT/DELETE /threads/:id/like`、
`POST /threads/:id/pin|lock|move|restore`；
`POST /threads/:id/replies`、`PATCH/DELETE /replies/:id`、
`PUT/DELETE /replies/:id/like`、`POST /replies/:id/restore`；
`GET /notifications`、`GET /notifications/unread-count`、`POST /notifications/read`；
`POST /uploads`（multipart，字段 `kind`、`file`）。

删除用户自己的内容记录 `deleted_by`；审核删除写审计（含 `reason`）。

### 管理后台

前缀 `/forum-v1/admin`，使用 `forum_admin_session`：

- 认证：`POST /auth/login|logout`、`GET /auth/me`
- 看板/审计：`GET /dashboard`、`GET /audit-logs`
- 板块：`GET/POST /boards`、`PATCH/DELETE /boards/:id`、`POST /boards/reorder`、
  `POST /boards/:id/restore`
- 内容：`GET/PATCH/DELETE /threads...`、`POST /threads/batch`、
  `GET/PATCH/DELETE /replies...`、`GET /replies/:id`、`POST /replies/batch`、`POST .../restore`；
  `GET /replies/:id` 返回完整 `contentHtml`（含已删除跟帖），列表接口只返回摘要。
- 用户：`GET /users`、`GET /users/:id`、`PATCH /users/:id/profile`、
  `POST /users/:id/ban|unban`、`GET/PUT /users/:id/permissions`、
  `GET /users/:id/login-records`
- 权限：`GET /permissions`、`GET /permissions/:code/users`
- 设置：`GET/PUT /settings`
- 上传：`GET /uploads`、`GET /uploads/stats`、`DELETE /uploads/:id`

后台写操作按用户限流；`batch` 对每个 ID 单独校验权限，返回
`{succeeded:[], failed:[{id, reason}]}`。

## 站点设置

| key | 默认值 |
|---|---|
| `site_name` | `Tide` |
| `site_description` | `一个简单的论坛。` |
| `announcement` | `""` |
| `footer_text` | `""` |
| `allow_guest_read` | `true` |
| `default_avatar_url_template` | `https://minotar.net/helm/{username}/200.png` |
| `registration_enabled` | `false`（保留键） |

数据库没有行时使用代码默认值；未知键在后台更新时返回 400。

## 限流

进程内限流（`golang.org/x/time/rate` + `ttlcache`），默认值见 `[forum.rate_limit]`：
登录 5 次/分钟/IP+username、单 IP 登录总尝试 30 次/分钟，发主题 10、编辑主题/跟帖 20、
发跟帖 20、上传 20、点赞/关注 60、提及搜索 60、全文搜索 30、后台写 60 次/分钟/用户。
超限返回 429。
限流器按 key 缓存并在 10 分钟空闲后过期，容量有上限。多实例部署需要改为共享限流。

## 请求安全

- 所有 `/forum-v1` 写请求校验 `Origin`：同源放行，跨源必须命中 `[base.cors] allow_origins`
  （`*` 不视为通配）；未带 `Origin` 的服务端调用不受影响。
- JSON 写接口要求 `Content-Type: application/json`，请求体上限按正文配置推导，避免无界请求体。
- 登录成功后会轮换 session token；用户不存在时也执行一次 bcrypt 比较，降低用户名枚举的时序差异。
- 非超级用户不能封禁超级用户，避免委派管理员锁死破窗账号。
- 全文搜索限制 `q` 长度（100 字符）与关键词数（5 个），并单独限流；只有裸 `#` 等无效关键词时返回空结果。
- 单篇主题最多 `max_tags_per_post` 个标签（默认 20），编辑主题/跟帖单独限流。
- 前台列表接口每页最多 50 条，后台最多 100 条。
- 静态文件路由对 key→MIME 做进程内正/负缓存，避免随机 key 反复查库。
- 批量接口单次最多 100 个 ID；`POST /notifications/read` 单次最多 100 条；
  `PUT /admin/users/:id/permissions` 最多 500 个节点。
- `PUT /me/avatar`、`PUT /me/background` 必须提供 `uploadId`（数字或 `null`）；
  字段缺失返回 400，显式 `null` 才表示清除。

## 运维

- 备份：数据库 + `state/forum/uploads/` 必须同时恢复，否则正文图片 404。
- 首次部署空库可用：超级用户破窗登录后台，创建板块并授予权限。
- 权限点目录不落库；新增权限点必须改代码并发布。
- 上传目录按用户/总量监控；后台 `GET /admin/uploads/stats` 提供分类字节数。

## 测试

- 单元测试：`internal/modules/forum/content`、`internal/modules/forum/perms`。
- API 级 E2E：`internal/modules/forum/forum_test.go`，使用 MySQL 测试库 + `httptest`，
  覆盖登录、建板块、上传、发帖、回复引用、点赞、通知、搜索、删除占位/恢复、板块级权限、
  封禁、游客开关、静态文件、审计与看板，以及 Content-Type、Origin、请求体上限、搜索上限与
  封禁超级用户等安全边界。

运行：

```bash
go test ./...
atlas migrate validate --env mysql
```
## 部署与安全

- 跨源前台/管理后台必须把 origin 加入 `[base.cors].allow_origins`。所有 `/forum-v1` 写请求带
  `Origin` 时会校验该列表（同源自动放行，`*` 不视为通配），不在列表内返回 403；未带 `Origin`
  的 curl/代理请求不强制校验。
- 反向代理部署时，把代理地址加入 `[base].trusted_proxies`；留空表示不信任任何代理头，
  `ClientIP` 使用直连地址，避免伪造 `X-Forwarded-For` 绕过登录限流。
- 本地 TLS 设置 `[base.tls].enabled=true` 会启用 cookie `Secure`；TLS 在上游终止时设置
  `[base].secure_cookie=true`。
- Nginx 需要放行最大上传体积（大于 `max_upload_bytes`）并转发 `X-Forwarded-For`。
