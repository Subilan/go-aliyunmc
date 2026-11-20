---
title: 机器人接入
description: 内网机器人调用 /bot-v1 的本机访问控制、配置与客户端示例。
---

# 机器人接入

机器人进程与服务端部署在同一台机器上，直接通过 `127.0.0.1` 调用 `/bot-v1`。该路由只接受来自本机回环地址的直连请求，不涉及公私钥通信，也不需要请求签名；服务端不向机器人签发用户登录态，也不复用 `/lab-v1`、`/forum-v1` 的 Cookie Session。

当前只提供 `GET /bot-v1/ping`，用于确认服务可达；业务接口后续再加入同一前缀。通过来源地址检查后即可访问 `/bot-v1` 下的全部接口。

反方向（后端主动调用机器人发送群消息）见 [机器人消息推送](bot-message-api.md)。

## 安全边界

- 只允许来源地址为 `127.0.0.1` / `::1` 的直连请求，其他来源一律返回 `403`。
- 携带 `Forwarded`、`X-Forwarded-For` 或 `X-Real-IP` 的请求视为经过反向代理，同样返回 `403`；即使代理与服务端同机，也不会被放行。
- 反向代理**不得**把 `/bot-v1` 暴露到外部。应用层只做来源判断，无法识别一个始终直连本机的代理。
- 服务端不验证机器人身份。任何能在本机发起回环连接的本机进程都可以访问 `/bot-v1`，请按“整台主机可信”来部署。
- 回环流量不出机器，不需要 HTTPS；调用时忽略 `base.tls`，直接使用 `base.expose`（启用 TLS 时为 `base.tls.http_port`）的明文端口。

## 配置

```toml
[bot]
enabled = true
rate_limit_per_minute = 120
```

| 字段 | 说明 |
|---|---|
| `enabled` | 为 false 时不注册 `/bot-v1`，访问返回 404 |
| `rate_limit_per_minute` | 每分钟请求数；按来源地址限流，0 表示不限制 |

请求体沿用进程级 `4 MiB` 上限。

## 服务端配置步骤

1. 在 `$GO_ALIYUNMC_STATE_DIR/config/config.toml` 写入上文 `[bot]` 配置。
2. 执行 `GO_ALIYUNMC_STATE_DIR=/opt/aliyunmc ./aliyunmc check_config`。
3. 重启服务。
4. 验证：本机执行 `curl http://127.0.0.1:45678/bot-v1/ping` 应返回 `200`；从其他机器访问应返回 `403`（若端口只监听回环地址，则连接会被直接拒绝）。

## 客户端配置步骤

1. 机器人进程与服务端在同一台机器上，`BOT_BASE_URL` 指向 `http://127.0.0.1:<port>`（默认端口 `45678`）。
2. 直接发送普通 HTTP 请求，不需要生成签名，也不需要额外的认证请求头。
3. 不要经过本机反向代理调用，否则会因转发头被拒绝。

### Python

```python
import requests

BASE_URL = "http://127.0.0.1:45678"

resp = requests.get(BASE_URL + "/bot-v1/ping")
print(resp.status_code, resp.text)  # 200 {"data":{"ok":true,"time":...}}
```

## 响应与错误码

响应沿用项目约定：成功为 `{"data": ...}`，失败为 `{"details": "..."}`。

| 状态码 | 含义 |
|---|---|
| `200` | 请求通过 |
| `403` | 来源不是本机回环地址，或请求带有代理转发头 |
| `429` | 超过 `rate_limit_per_minute` |

`GET /bot-v1/ping` 返回服务端时间，可用于确认机器人能否连通服务。