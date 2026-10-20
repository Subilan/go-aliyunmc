---
title: 订阅广播
description: Hub 的订阅、缓冲溢出处理与关闭语义，以及状态存储如何复用。
---

# 订阅广播

`internal/pubsub` 提供泛型广播内核，状态分发与 SSE 事件流共用。

## 订阅

`Subscribe(buffer, onOverflow)` 为每个订阅者创建独立的缓冲 channel，`buffer` 小于 0 时按 0 处理。返回值包括只读 channel、幂等的取消函数与 `ok`。Hub 已关闭时 `ok` 为 false，返回的 channel 已经关闭，取消函数是空操作。

## 溢出

`Broadcast` 非阻塞投递。订阅者缓冲已满时按 `onOverflow` 处理。

| `onOverflow` | 行为 | 适用场景 |
|---|---|---|
| nil | 丢弃本次值，保留订阅者 | 状态类数据，后续值会覆盖当前值 |
| 非 nil | 移除订阅者并关闭 channel，在锁外调用回调 | 事件流，慢消费者应被断开 |

回调在锁外执行，可以再次访问 Hub 或执行取消操作。

## 关闭

`Close` 关闭所有订阅 channel 并拒绝后续订阅，可以重复调用。关闭后 `Broadcast` 不再产生效果。

`states.HubbedStore` 使用缓冲为 1 的订阅合并状态更新，来不及消费的中间值会被丢弃，只保留最新状态。
