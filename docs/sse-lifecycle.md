# SSE 生命周期

下面这张时序图整理了“客户端请求任务 SSE 输出”从建立连接到 `task_done` 后主动关闭的完整生命周期。

```mermaid
sequenceDiagram
    autonumber
    participant C as 客户端
    participant H as HandleGetTaskOutput
    participant CL as sse.Client
    participant E as tasks.Executor
    participant B as sse.Broker
    participant F

    C->>H: GET /task/{id}/output
    H->>E: GetExecutor(taskID)
    break when GetExecutor 返回为空
        E--xC: 找不到正在执行的 taskID 任务
    end
    H->>CL: NewClient(c)
    H->>CL: go Listen()
    H->>E: SubscribeOrFail(client)
    break when Broker 未初始化（小概率）
        E--xC: 无订阅目标，任务未准备完毕
    end
    E->>B: Register(client)

    loop 任务执行期间
        F->>E: tc.println/tc.status
        E->>B: Broadcast(task_status_update)/Broadcast(task_output)
        B-->>CL: client.eventChan <- event
        CL->>C: 写出 SSE 帧并 Flush
    end

    F->>E: tc.done()
    E->>B: Broadcast(task_done)
    B-->>CL: client.eventChan <- event(task_done)
    CL->>C: 写出 task_done 并 Flush
    CL->>CL: Listen 退出，defer Close()
    CL-->>H: client.Done() 结束
    H->>E: defer Unsubscribe(client)
    E->>B: Unregister(client)
    B->>B: 清理订阅
```

## 说明

- `HandleGetTaskOutput` 负责把 HTTP 请求升级为 SSE 连接，并把客户端注册到当前任务的执行器上。
- `tasks.Executor` 负责在任务执行期间广播状态和输出。
- `sse.Broker` 负责把事件分发给所有已订阅客户端。
- `sse.Client.Listen()` 负责把事件真正写到 HTTP 响应里；收到 `task_done` 后会退出，从而触发连接关闭。
- 路由层在连接结束后会执行退订，避免订阅残留。
