# 论坛实时通道

论坛前台使用一条用户级 SSE 连接承载通知与内容互动事件。lab 模块的任务输出与状态监听
仍使用各自的任务级 SSE，不在此通道内。

## 连接

- 端点：`GET /forum-v1/events`
- 认证：论坛 session cookie（`forum_session`，`Path=/forum-v1`）。端点必须挂在 `/forum-v1`
  下，否则浏览器不会携带该 cookie。
- 每个登录标签页只建立一条 `EventSource`；退出登录时前端主动关闭。
- 服务端每 5 秒发送 `:keep-alive` 注释；连接建立后立即发送 `:connected` 注释并 flush。

## 事件

所有事件都是 SSE 命名事件（`event:`），`data` 为 JSON。

| 事件 | 载荷 | 受众 |
|---|---|---|
| `ready` | `{unreadCount}` | 当前连接 |
| `notification` | `{notification, unreadCount}` | 目标用户 |
| `notification.read` | `{unreadCount, ids?, all?}` | 目标用户 |
| `thread.like` | `{threadId, likeCount, actorId, liked}` | 全局广播 |
| `reply.like` | `{threadId, replyId, likeCount, actorId, liked}` | 全局广播 |
| `user.like` | `{userId, profileLikeCount, actorId, liked}` | 全局广播 |
| `board.follow` | `{boardId, boardSlug, followerCount, actorId, following}` | 全局广播 |

- `ready` 在每次连接/重连时发送，携带服务端权威未读数；前端据此重同步通知列表。
- `notification` 携带完整的 `Notification` DTO（字段见 `/notifications`），前端直接 upsert
  到通知列表与弹层缓存，不再为每条通知重新拉取。
- 内容互动事件是 best-effort 实时计数：客户端只补丁自己已缓存的 query；并发操作时不同事件的
  计数可能乱序到达，下一次 REST 拉取会纠正。`actorId` 与当前用户一致时，客户端同步
  `likedByMe` / `followingByMe`。
- 通知不做删除/撤回，因此没有删除类事件。
- 匿名用户不建立连接，互动计数在刷新后以 REST 为准。

## 断线与重连

- `EventSource` 自动重连；浏览器不会暴露 401/403 状态，session 过期时会持续重试。
- 每次重连都会收到新的 `ready`，前端据此重新拉取通知。断线期间丢失的互动事件由下一次页面
  或列表拉取纠正。
- 单连接事件缓冲为 64；缓冲溢出时服务端关闭该连接，浏览器重连并重新同步。

## 部署

- 开发环境：Vite 只代理 `/forum-v1`，SSE 与页面同源。
- 生产环境：反向代理必须把 `/forum-v1` 转发到 Go 服务，并关闭对该路径的响应缓冲
  （后端已设置 `X-Accel-Buffering: no`），同时放宽 `proxy_read_timeout`。
- 如果使用绝对跨域的 `VITE_API_BASE`，前端会带 `withCredentials: true`，后端 CORS 需要
  允许对应 origin 且 `allow_credentials = true`。

## 进程模型

Hub 是进程内实现，只向连接到当前进程的客户端推送。当前部署为单进程；若未来多实例部署，
需要把发布层替换为 Redis/NATS 等跨进程广播。数据库始终是通知的事实源，SSE 丢失不会导致
数据不一致。