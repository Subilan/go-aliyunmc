---
title: 机器人服务器接口
description: /bot-v1 的服务器状态聚合与一键启动接口、响应字段、幂等与并发规则。
---

# 机器人服务器接口

这两个接口沿用 [机器人接入](bot-integration.md) 的本机回环访问控制：只能从 `127.0.0.1` / `::1`
直连，携带 `Forwarded`、`X-Forwarded-For` 或 `X-Real-IP` 的请求返回 `403`。响应同样使用
`{"data": ...}` / `{"details": "..."}` 包装。

## GET /bot-v1/server-status

聚合当前实例与 Minecraft 服务器的状态，以「服务器」为整体返回，不包含实例 ID、规格或 ECS 状态。

### 没有活跃实例

```json
{"data": {"available": false}}
```

### 存在活跃实例

```json
{
  "data": {
    "available": true,
    "online": true,
    "ip": "203.0.113.10",
    "players": {"online": 0, "max": 20},
    "uptimeSeconds": 7320,
    "release": {"inSeconds": 540, "at": 1763642400},
    "memory": {"usedBytes": 2147483648, "totalBytes": 8589934592, "usagePercent": 25},
    "cpu": {"usedCores": 0.4, "cores": 4, "usagePercent": 10.2}
  }
}
```

| 字段 | 说明 |
|---|---|
| `available` | 是否存在活跃实例 |
| `online` | 服务器是否在线；状态尚未取得时视为离线 |
| `ip` | 实例公网 IP，尚未分配时为空字符串 |
| `players.online` | 当前在线玩家数，离线或状态未知时为 `0` |
| `players.max` | 最大玩家数；服务器离线或状态未知时为 `null` |
| `uptimeSeconds` | 实例主机的连续运行时长（`/proc/uptime`）；无法采集时为 `null` |
| `release` | 预计释放时间，仅当玩家数为 `0` 且空服倒计时正在运行或收尾时出现 |
| `release.inSeconds` | 距离释放的秒数；关服流程已进入收尾时为 `0` |
| `release.at` | 预计释放时刻的 Unix 秒时间戳 |
| `memory.usedBytes` | 已用内存：`MemTotal - MemAvailable` |
| `memory.totalBytes` | 可分配内存总量 |
| `memory.usagePercent` | 内存占用率，保留一位小数 |
| `cpu.usedCores` | 按占用率折算的使用核心数 |
| `cpu.cores` | 主机 CPU 核心数 |
| `cpu.usagePercent` | 全部核心的整体占用率，保留一位小数 |

`release` 在倒计时暂停、未启用或尚未开始时省略，此时不存在确定的释放时间。`players.online`
以服务器状态轮询（`[monitors.server] poll_interval_sec`）为准。

`uptimeSeconds`、`memory` 与 `cpu` 由 `mc` 用户通过 SSH 读取实例的 `/proc` 得到，采样间隔由
`[samplers.system_metrics] sample_interval_sec` 控制（默认 `60` 秒）。无法连接实例或实例未部署、
未运行时，这些字段为 `null`；`cpu` 还需要相邻两次采样才能计算，因此服务刚启动或实例刚恢复时
会在一个采样间隔内为 `null`。

## POST /bot-v1/trigger-one-click-start

以机器人身份触发 `oneclick_start_game` 任务，不需要请求体或查询参数。触发会产生副作用，因此使用
`POST`。接口立即返回，任务在后台执行。

```bash
curl -X POST http://127.0.0.1:45678/bot-v1/trigger-one-click-start
```

```json
{"data": {"taskId": 42, "status": "running", "alreadyRunning": false}}
```

| 字段 | 说明 |
|---|---|
| `taskId` | 任务 ID |
| `status` | 任务状态，触发成功后为 `running` |
| `alreadyRunning` | 为 `true` 时表示复用了正在执行的任务，本次没有新建任务 |

### 幂等与并发

- 一键启动任务执行期间重复调用不会新建任务，返回同一个 `taskId`，且 `alreadyRunning` 为 `true`。
- 创建实例、部署、单独启动服务器与一键启动共用 `deploy_chain` 互斥分组。分组已被其他任务占用时返回
  `409`，等该任务结束后再触发。
- 已有活跃实例时不会执行一键启动，返回 `409`；一键启动不会续跑已有实例。

### 错误

| 状态码 | 含义 |
|---|---|
| `403` | 来源不是本机回环地址，或请求带有代理转发头 |
| `409` | 已有活跃实例，或其他部署链任务正在执行 |
| `429` | 超过 `rate_limit_per_minute` |
| `500` | 创建任务记录失败等内部错误 |

任务进度与输出不在 `/bot-v1` 暴露。机器人可以轮询 `GET /bot-v1/server-status`，以 `online`
和 `players` 判断启动是否完成。