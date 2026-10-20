---
title: 任务执行与生命周期
description: 任务定义的字段、Executor 的生命周期信号、互斥分组与输出回放。
---

# 任务执行与生命周期

## 任务定义

每个任务在 `internal/tasks` 注册一个 `TaskDefinition`。

| 字段 | 作用 |
|---|---|
| `Type` | 任务类型，与执行函数一一对应 |
| `Timeout` | 超时时间，0 表示不超时 |
| `Exclusive` | 是否互斥 |
| `ExclusiveGroup` | 互斥分组，为空时等于 `Type` |
| `Role` | 执行所需的最低权限，未填时为 `perms.RoleUser` |
| `F` | 执行函数，接收 `TaskContext` 与参数 |
| `C` | 参数与环境检查，执行前调用 |
| `E` | 权限检查，执行前调用 |

`F` 在独立 goroutine 中运行，返回 nil 表示成功，返回 error 或 panic 表示失败。`TaskContext` 负责输出、步骤上报与中断。

## 生命周期

`Executor` 的生命周期由三个按序发生的信号描述。

1. `tc.ctx.Done()`，任务已确定结局（done、throw 或超时），monitor 开始收尾
2. `taskFuncDone`，执行函数已经返回，不会再产生副作用
3. `Done()`，monitor 收尾完毕，落库、广播 `task_done`、注销执行器，任务真正停止

`e.task` 归 monitor goroutine 所有，其他 goroutine 只能读取启动时复制的快照，否则会与 monitor 的写入产生数据竞争。

执行函数的所有退出路径，包括正常返回、错误、panic 与 `runtime.Goexit`，都会先确定终态再关闭 `taskFuncDone`。直接返回但没有调用 done 或 throw 会被判为失败。

## 互斥

`Exclusive` 为 true 时，同一 `ExclusiveGroup` 的任务在执行期间不能再次启动。互斥检查与占位在 `RunTask` 中原子完成，重复触发返回 `ErrTaskTypeExecuting`。

创建实例、部署、启动服务器与启动游戏共享 `deploy_chain` 分组，因为它们操作同一台实例。

## 输出与订阅

任务输出按行写入 `task_output_lines`，每行带递增序号。`tasks.output` 列仅为旧库兼容保留。

`SubscribeFrom` 先订阅实时事件，再请求 monitor 把待写入的输出落库并返回已落库的最大序号，然后回放 `seq > afterSeq` 的持久化行，最后转发实时事件。这个顺序保证回放与实时部分既不丢失也不重复。回放期间到达的实时事件先缓存，回放结束后再按序号转发，已经被回放覆盖的输出事件会被丢弃。

客户端通过 `GET /task/:id/output` 建立 SSE 连接。重连时用 `Last-Event-ID` 指定 `afterSeq`。收到 `task_done` 后再次请求返回 204，EventSource 不再重连。

## 中断

`POST /task/:id/interrupt` 返回 202 表示中断请求已经发出，不表示任务已经停止。执行函数的退出与终态收尾由 monitor 异步完成，调用方需要通过 SSE 的 `task_done` 或 `GET /task/:id` 确认最终状态。

一键启动游戏的流程见 [一键启动游戏](one-click-start.md)。
