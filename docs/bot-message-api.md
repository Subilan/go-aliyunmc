---
title: 机器人消息推送
description: 后端调用机器人发送群消息的客户端、消息模板、配置与错误处理。
---

# 机器人消息推送

与 [机器人接入](bot-integration.md) 的方向相反：这里是后端主动调用机器人进程，向固定 QQ 群发送消息。
机器人进程在 `127.0.0.1` 上暴露 HTTP 接口，后端通过 `internal/common/botapi` 调用。

## 接口

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/bot/group-message` | 向固定群发送一条纯文本消息 |
| GET | `/bot/health` | 查询机器人是否已连接、目标群是否已配置 |

机器人只接受本机回环地址的直连请求，携带 `Forwarded`、`X-Forwarded-For`、`X-Real-IP`
的请求一律返回 `403`。客户端固定连接 `127.0.0.1`，并关闭环境代理，避免请求经本机代理转发。

## 配置

```toml
[bot.api]
port = 8081
```

| 字段 | 默认值 | 说明 |
|---|---|---|
| `port` | 8081 | 机器人 HTTP 接口的监听端口 |

`[bot] enabled` 只控制后端是否注册 `/bot-v1` 入站路由，与这里的出站调用无关。

## 调用

```go
client := botapi.New(config.Get().Bot.API.Port)
result, err := client.SendGroupMessage(ctx, "服务器将在 10 分钟后重启")
```

需要反复发送时可使用 `botapi.Default()`：它按当前配置创建进程级客户端并复用连接。

不在意发送结果时使用异步方法或包级快捷方式，它们在后台发送并立即返回，失败只写一条 warning 日志：

```go
botapi.NotifyGroup("服务器将在 10 分钟后重启")

// 与上一行等价，适合已经持有客户端时使用。
client.SendGroupMessageAsync("服务器将在 10 分钟后重启")
```

- 不阻塞调用方，也不返回错误；机器人离线、QQ 限流等失败都被忽略。
- 消息为空时直接跳过，不发起请求。
- 不保证发送顺序，也不保证进程退出前送达。

同步调用还有以下约定：

- 消息会先去除首尾空白；去空白后为空时直接返回 `ErrEmptyMessage`，不发起请求。
- 单次请求超时 10 秒，调用方的 `ctx` 取消会中断请求。
- 只支持纯文本，机器人默认上限 2000 字符，超长由机器人返回 `400`。

## 消息模板

业务消息的文本统一放在 `$GO_ALIYUNMC_STATE_DIR/config/messages.toml`，不写死在代码里。
每次发送前都会重新读取该文件，修改后无需重启；文件不存在、缺少对应键或模板渲染失败时只记录
warning，消息被跳过，不影响业务流程。可以从 `examples/messages.toml` 复制一份：

```toml
server_started = "服务器已开启~"
```

模板使用 Go `text/template` 语法，现有模板：

| 键 | 触发时机 | 可用变量 |
|---|---|---|
| `server_started` | `oneclick_start_game` 成功完成 | `Address`（`IP:端口`，IP 未知时为空）、`Port` |
| `server_released` | 实例被删除或释放后（手动删除、自动归档回收、检测到外部删除） | 无 |
| `idle_countdown_reminder` | 空服倒计时的剩余时间首次到达 `reminder_before_sec` | `IdleMinutes`（已空服分钟数，含暂停期间）、`RemainingMinutes`（距释放分钟数） |
| `player_joined` | 玩家加入服务器后，合并窗口结束时 | `Players`（以 `、` 分隔的玩家名） |
| `player_left` | 玩家离开服务器后，合并窗口结束时 | `Players`（以 `、` 分隔的玩家名） |

`idle_countdown_reminder` 每个倒计时周期只发送一次。触发阈值由
`[monitors.auto_archive_idle] reminder_before_sec` 配置：`0` 表示关闭提醒，其余取值必须小于
`idle_countdown_sec`。负数、超过 7 天或不小于 `idle_countdown_sec` 都视为配置不合理，启动时会输出
warning 并跳过该提醒，不影响倒计时与归档流程。

玩家进出播报由 `[monitors.player_presence]` 控制，复用 `[samplers.player_count]` 的 GameSpy query
采样，不额外发起请求：

```toml
[monitors.player_presence]
enabled = true
batch_sec = 10
```

监控按窗口比较在线集合的净变化，同一窗口内的多个玩家合并成一条消息；窗口内加入又离开不会播报。
后端启动、采样断档超过阈值、服务器离线或停服时会重建基线，不播报这些变化。该功能要求服务器开启
`enable-query=true`。

例如在消息里带上地址：

```toml
server_started = "服务器已开启~ 地址：{{.Address}}"
```

探活：

```go
health, err := client.Health(ctx)
if err == nil && health.BotReady && health.GroupConfigured {
    // 可以发送
}
```

`Health` 不触发 QQ 侧请求，可以高频轮询。

## 错误处理

非 `200` 响应返回 `*botapi.APIError`，`StatusCode` 与 `Detail` 对应机器人返回的状态码和 `detail` 字段；
`Retryable()` 在 `502`、`503` 时为真，适合退避重试。

| 状态码 | 含义 |
|---|---|
| 400 | 消息为空或超长 |
| 403 | 非本机请求或带转发头 |
| 422 | 请求体结构不符合 schema |
| 502 | 调用 QQ 接口失败 |
| 503 | 机器人未连接、未就绪或目标群未知 |

## 注意

该接口使用 QQ 的主动消息，受开放平台频率与权限限制；返回 `502` 时先检查机器人日志与平台消息配额。
机器人侧接口的完整定义见机器人仓库的 `apis.md`。