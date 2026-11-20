---
title: 机器人接入
description: 外部机器人调用 /bot-v1 的 Ed25519 请求签名、配置与 Python 客户端示例。
---

# 机器人接入

外部机器人通过 HTTP 调用 `/bot-v1`，使用 Ed25519 对每个请求签名。服务端只保存机器人公钥，不向机器人签发用户登录态，也不复用 `/lab-v1`、`/forum-v1` 的 Cookie Session。

当前只提供 `GET /bot-v1/ping`，用于验证密钥、时钟与签名链路；业务接口后续再加入同一前缀。认证通过后即可访问 `/bot-v1` 下的全部接口，不再划分权限范围。

## 安全边界

- 只签请求，不签响应。请求与响应的机密性由 HTTPS 保证；生产必须由本机 TLS 或反向代理提供 HTTPS。
- 认证只依赖签名，不依赖来源 IP。`base.trusted_proxies` 只影响日志与限流里使用的 `ClientIP()`。
- 反向代理不得改写 path 或 query；签名覆盖原始 `RequestURI`。
- 防重放缓存位于进程内。多实例部署时 nonce 不共享，重放可能落到另一实例；当前按单实例设计。
- 单机器人、单密钥。轮换密钥需要协调机器人与服务端同时切换并重启。

## 配置

```toml
[bot]
enabled = true
key_id = 'default'
# 始终相对配置目录解析，与 config.toml 同目录
public_key_path = 'bot_public.pem'
timestamp_tolerance_sec = 300
nonce_capacity = 10000
max_body_bytes = 1048576
rate_limit_per_minute = 120
```

| 字段 | 说明 |
|---|---|
| `enabled` | 为 false 时不注册 `/bot-v1`，访问返回 404 |
| `key_id` | 请求头 `X-Bot-Key-Id` 必须与它完全一致 |
| `public_key_path` | 机器人公钥文件，PKIX PEM；始终相对配置目录（`$STATE_DIR/config`）解析，不支持绝对路径 |
| `timestamp_tolerance_sec` | 允许的时钟偏差，同时决定 nonce 缓存时长 |
| `nonce_capacity` | 防重放缓存容量 |
| `max_body_bytes` | 机器人请求体上限 |
| `rate_limit_per_minute` | 每分钟请求数；0 表示不限制 |

`./aliyunmc check_config` 会额外加载公钥文件；路径错误或文件不是 PKIX Ed25519 公钥都会在启动前失败。

`public_key_path` 始终相对配置目录解析，不支持绝对路径。CI 与生产机的目录布局不同，只有相对路径能让 `check_config` 在两边都指向随 `config/` 发布的公钥。

## 密钥生成与分发

机器人持有私钥，服务端只保存公钥。建议在机器人主机生成，只把公钥交给服务端。

```bash
# 项目自带命令
./aliyunmc gen_bot_key -out-dir /tmp/botkeys

# 等价 openssl 命令
openssl genpkey -algorithm ED25519 -out bot_private.pem
openssl pkey -in bot_private.pem -pubout -out bot_public.pem
```

`gen_bot_key` 默认写到 `$STATE_DIR/state/bot/bot_private.pem`（0600）与 `bot_public.pem`（0644），支持 `-out-dir` 与 `-force`，并打印公钥 SHA-256 指纹。私钥不要提交到仓库，也不要留在服务端。

## 请求协议（v1）

每次请求带四个头：

| Header | 说明 |
|---|---|
| `X-Bot-Key-Id` | 等于配置中的 `key_id` |
| `X-Bot-Timestamp` | Unix 秒；与服务器时间相差不超过 `timestamp_tolerance_sec` |
| `X-Bot-Nonce` | 16–128 个 `[A-Za-z0-9_-]` 字符，每次请求唯一 |
| `X-Bot-Signature` | Ed25519 签名，标准 Base64（含 padding） |

每个头必须恰好出现一次；同名头重复会被拒绝。

签名对象是下面按 `\n` 连接、末尾不带换行的字符串：

```
v1
<key_id>
<METHOD>
<RequestURI>
<timestamp>
<nonce>
<lowercase hex(sha256(body))>
```

- `key_id`、`timestamp`、`nonce` 使用请求头里的原始字符串，不做格式化。
- `RequestURI` 是请求行里的原始 path+query；例如 `/bot-v1/ping?x=1`。
- 空请求体的 SHA-256 是 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- 签名针对原始 body 字节。不要压缩请求体，也不要让中间层改写 body。
- Ed25519 直接对上述字符串签名，不做额外预哈希。

服务端按以下顺序校验：头部 → `key_id` → 时间戳 → nonce 格式 → 请求体大小 → 签名 → nonce 重放。认证失败统一返回 `401` 与 `{"details":"机器人认证失败"}`，具体原因只写日志。

## 服务端配置步骤

1. 生成密钥对，把私钥安全交给机器人。
2. 把公钥放到服务端配置目录（`$STATE_DIR/config/bot_public.pem`），与 `config.toml` 同目录并确保服务运行用户可以读取；`public_key_path` 填 `'bot_public.pem'` 即可。
3. 在 `$GO_ALIYUNMC_STATE_DIR/config/config.toml` 写入上文 `[bot]` 配置。
4. 执行 `GO_ALIYUNMC_STATE_DIR=/opt/aliyunmc ./aliyunmc check_config`。
5. 重启服务，确认 HTTPS 由本机 TLS 或反向代理提供。
6. 验证：不带认证头访问 `GET /bot-v1/ping` 应为 `401`；带正确签名应为 `200`。失败原因在 `$STATE_DIR/state/logs/main.log` 中，记录 `key_id`、method、path 与 client_ip。

## 客户端配置步骤

1. 把 `bot_private.pem` 放到机器人主机，权限设为 `0600`；配置 `BOT_BASE_URL`、`BOT_KEY_ID`、`BOT_PRIVATE_KEY_PATH`。
2. 每次请求生成新的 nonce（至少 16 字节随机数），时间戳使用当前 Unix 秒；确保主机 NTP 正常。
3. 对将要发送的原始 body 字节计算 SHA-256；GET 请求 body 为空字节串。
4. 用私钥签名，标准 Base64 后放入 `X-Bot-Signature`。
5. `path+query` 必须与实际发送的值完全一致；不要压缩请求体。
6. 用 `/bot-v1/ping` 验证。`401` 时依次检查时钟、`key_id`、公钥是否匹配、Base64 是否带 padding、URI 与 body 是否被改写。

### Python

依赖 `cryptography` 与 `requests`。

```python
import base64, hashlib, secrets, time
import requests
from cryptography.hazmat.primitives import serialization

KEY_ID = "default"
BASE_URL = "https://mc.example.com"
with open("bot_private.pem", "rb") as f:
    PRIVATE_KEY = serialization.load_pem_private_key(f.read(), password=None)

def signed_headers(method: str, path_query: str, body: bytes = b"") -> dict:
    ts = str(int(time.time()))
    nonce = secrets.token_urlsafe(16)              # 22 个 URL-safe 字符
    body_hash = hashlib.sha256(body).hexdigest()
    canonical = "\n".join(["v1", KEY_ID, method, path_query, ts, nonce, body_hash])
    signature = base64.b64encode(PRIVATE_KEY.sign(canonical.encode())).decode()
    return {
        "X-Bot-Key-Id": KEY_ID,
        "X-Bot-Timestamp": ts,
        "X-Bot-Nonce": nonce,
        "X-Bot-Signature": signature,
    }

resp = requests.get(
    BASE_URL + "/bot-v1/ping",
    headers=signed_headers("GET", "/bot-v1/ping"),
)
print(resp.status_code, resp.text)
```

POST 时，`body` 必须是实际发送的字节，且与签名时使用的完全一致：

```python
import json

body = json.dumps({"message": "hello"}, separators=(",", ":")).encode()
headers = signed_headers("POST", "/bot-v1/example", body)
headers["Content-Type"] = "application/json"
resp = requests.post(BASE_URL + "/bot-v1/example", data=body, headers=headers)
```

## 响应与错误码

响应沿用项目约定：成功为 `{"data": ...}`，失败为 `{"details": "..."}`。

| 状态码 | 含义 |
|---|---|
| `200` | 请求通过 |
| `401` | 认证失败：头部缺失/重复、`key_id` 不符、时间戳超差、nonce 格式错误或重放、签名无效 |
| `413` | 请求体超过 `max_body_bytes` |
| `429` | 超过 `rate_limit_per_minute` |

`GET /bot-v1/ping` 返回服务端时间与 `key_id`，可用于排查时钟与密钥配置问题。