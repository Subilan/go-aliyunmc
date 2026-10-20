---
title: 抢占式实例回收保护
description: 回收通知的检测方式、保护性停机流程与保护状态的解除条件。
---

# 抢占式实例回收保护

阿里云会在抢占式实例被回收前约 5 分钟，通过实例的 `OperationLocks` 字段发出待回收通知，`LockReason` 为 `Recycling`。`SpotInterruptionMonitor` 定期调用 `DescribeInstances` 读取该字段。轮询间隔由 `[monitors.spot_interruption]` 的 `poll_interval_sec` 控制，默认 30 秒。

## 保护流程

检测到待回收通知后，监控器立即执行保护性停机。

1. 向 Minecraft 服务器发送 `stop`，触发全盘保存
2. 等待服务器离线，超时由 `stop_wait_timeout_sec` 控制，默认 90 秒

离线检查间隔由 `offline_check_interval_sec` 控制，默认 3 秒。

流程不执行归档。数据依赖实例磁盘保留，因此 `[aliyun.ecs]` 的 `spot_interruption_behavior` 应为 `Stop`。实例被回收后，管理员通过 `POST /instance/start` 重新开启实例来恢复服务。

如果等待离线超时，说明存档可能仍在写入，实例被回收时存在回档风险，日志会提示关注实例磁盘数据。

## 保护状态

停机后系统进入回收保护状态，状态包含 `Active`、`InstanceId` 与 `TriggeredAt`。

- 保护期间 `start_server` 与一键启动游戏都会拒绝启动服务器
- 实例恢复 `Running` 后保护状态自动解除
- 自动解除失败时，管理员可以调用 `DELETE /instance/spot-interruption` 手动解除

状态通过 `/state/snapshot/spot-interruption` 与 `/state/watch/spot-interruption` 对外提供。
