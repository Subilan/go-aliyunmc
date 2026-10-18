package tasks

import (
	"strconv"

	"github.com/Subilan/go-aliyunmc/log_util"
	"github.com/Subilan/go-aliyunmc/sse"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
)

const (
	// taskReplayBuffer 是带历史回放的订阅者缓冲大小。回放期间订阅者可能
	// 暂时跟不上广播速度，因此比普通订阅的缓冲更大。
	taskReplayBuffer = 1024
	// taskOutputReplayChunk 是单次从数据库读取的回放行数。
	taskOutputReplayChunk = 500
)

// SubscribeFrom 将 client 注册为当前任务的事件订阅者，并先回放 seq > afterSeq
// 的持久化输出，再继续转发实时事件。
//
// 实现要点：先订阅再请求 monitor 刷盘。订阅之后产生的输出一定在 live 通道中；
// 订阅之前产生的输出由刷盘保证已经写入数据库，因此两者拼接不会丢失或重复。
func (e *Executor) SubscribeFrom(client *sse.Client, afterSeq uint64) (func(), bool) {
	if e.hub == nil {
		return func() {}, false
	}

	live, unsubscribe, ok := e.hub.Subscribe(taskReplayBuffer, client.Close)
	if !ok {
		return func() {}, false
	}

	persistedSeq := e.persistedSnapshot()

	replay := make(chan sse.Event)
	go func() {
		defer close(replay)
		streamPersistedOutput(replay, e.taskID, afterSeq, &persistedSeq, client.Done())
	}()

	client.Attach(relayEvents(client, replay, live, persistedSeq))
	return unsubscribe, true
}

// persistedSnapshot 请求 monitor 把当前待写入的输出全部落库，并返回已落库的最大序号。
// 若执行器已经结束，则所有输出都已在终止流程中落库，返回 lastSeq 即可。
func (e *Executor) persistedSnapshot() uint64 {
	reply := make(chan uint64, 1)

	select {
	case e.flushReq <- reply:
	case <-e.ctx.Done():
		return e.lastSeq.Load()
	}

	select {
	case seq := <-reply:
		return seq
	case <-e.ctx.Done():
		return e.lastSeq.Load()
	}
}

// TaskDoneEventID 是 task_done 的固定事件 ID。客户端重连时带回该 ID，
// 服务端返回 204 停止 EventSource 重连。
const TaskDoneEventID = "done"

// ReplayTaskOutput 回放一个已结束任务的持久化输出，并在最后发送 task_done。
// 调用方需要在调用后执行 client.Listen()。
func ReplayTaskOutput(client *sse.Client, task *models.Task, afterSeq uint64) {
	events := make(chan sse.Event, taskEventBuffer)
	client.Attach(events)

	go func() {
		defer close(events)

		streamPersistedOutput(events, task.ID, afterSeq, nil, client.Done())

		select {
		case events <- taskDoneEvent(task.Status == models.TaskStatusSuccess, task.Error):
		case <-client.Done():
		}
	}()
}

// streamPersistedOutput 按块读取 (afterSeq, maxSeq] 内的输出行并写入 out。
// out 由调用方关闭。
func streamPersistedOutput(out chan<- sse.Event, taskID uint, afterSeq uint64, maxSeq *uint64, done <-chan struct{}) {
	cursor := afterSeq

	for {
		lines, err := store.GetTaskOutputLinesRange(taskID, cursor, maxSeq, taskOutputReplayChunk)
		if err != nil {
			log_util.Error("读取任务输出失败: %v\n", err)
			return
		}

		for _, line := range lines {
			select {
			case out <- taskOutputEvent(line):
			case <-done:
				return
			}
		}

		if len(lines) < taskOutputReplayChunk {
			return
		}
		cursor = lines[len(lines)-1].Seq
	}
}

// relayEvents 把回放事件和实时事件合并为一条有序事件流。
// 回放完成前先缓存实时事件，避免历史与实时事件交错；回放结束后再按序转发。
// replayMax 是回放覆盖到的最大序号，序号不大于它的实时输出事件会被丢弃（已回放过）。
func relayEvents(client *sse.Client, replay, live <-chan sse.Event, replayMax uint64) <-chan sse.Event {
	out := make(chan sse.Event, taskEventBuffer)

	go func() {
		defer close(out)

		var deferred []sse.Event

		for replay != nil || live != nil {
			select {
			case event, ok := <-replay:
				if !ok {
					replay = nil
					for _, cached := range deferred {
						if !forwardEvent(out, client, cached, replayMax) {
							return
						}
					}
					deferred = nil
					continue
				}

				select {
				case out <- event:
				case <-client.Done():
					return
				}

			case event, ok := <-live:
				if !ok {
					live = nil
					continue
				}

				if replay == nil {
					if !forwardEvent(out, client, event, replayMax) {
						return
					}
				} else {
					deferred = append(deferred, event)
				}
			}
		}
	}()

	return out
}

func forwardEvent(out chan<- sse.Event, client *sse.Client, event sse.Event, replayMax uint64) bool {
	if seq, ok := outputEventSeq(event); ok && seq <= replayMax {
		return true
	}

	select {
	case out <- event:
		return true
	case <-client.Done():
		return false
	}
}

func outputEventSeq(event sse.Event) (uint64, bool) {
	output, ok := event.Data.(TaskOutput)
	if !ok {
		return 0, false
	}
	return output.ID, true
}

func taskOutputEvent(line models.TaskOutputLine) sse.Event {
	return sse.Event{
		ID:    strconv.FormatUint(line.Seq, 10),
		Event: "task_output",
		Data: TaskOutput{
			ID:     line.Seq,
			Step:   line.Step,
			Output: line.Output,
		},
	}
}

func taskDoneEvent(success bool, errorMessage string) sse.Event {
	return sse.Event{
		ID:    TaskDoneEventID,
		Event: "task_done",
		Data: TaskSummary{
			Success: success,
			Error:   errorMessage,
		},
	}
}
