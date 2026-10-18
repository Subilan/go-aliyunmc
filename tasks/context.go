package tasks

import (
	"context"
	"github.com/Subilan/go-aliyunmc/store/models"
	"time"
)

// taskOutputChanBuffer 是任务输出进入 executor 前的缓冲行数。
// 使用有缓冲 channel 可以在输出突发时保留每一行，而不是依赖调用方恰好及时接收。
const taskOutputChanBuffer = 1024

type TaskContext struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	// step 记录了当前任务所处的步骤序号。
	step uint
	// outputChan 用于向 executor 传递一个任务输出。该输出将立即反馈到数据库并同步到客户端。
	outputChan chan TaskOutput
	// statusChan 用于向 executor 传递一个任务状态的更新。该更新将立即反馈到数据库并同步到客户端。
	statusChan chan models.TaskStatus
}

func NewTaskContext() *TaskContext {
	return NewTaskContextWithTimeout(0)
}

// NewTaskContextWithTimeout 创建一个带有超时设置的 TaskContext。
// 当 timeout 大于 0 时，如果任务执行时间超过 timeout，TaskContext 将自动调用 throw 方法将任务标记为失败，并且 cause 将被设置为 context.DeadlineExceeded。
func NewTaskContextWithTimeout(timeout time.Duration) *TaskContext {
	ctx, cancelCause := context.WithCancelCause(context.Background())
	var timeoutCancel context.CancelFunc
	if timeout > 0 {
		ctx, timeoutCancel = context.WithTimeoutCause(ctx, timeout, context.DeadlineExceeded)
	}

	cancel := context.CancelCauseFunc(func(cause error) {
		cancelCause(cause)
		if timeoutCancel != nil {
			timeoutCancel()
		}
	})

	return &TaskContext{
		ctx:        ctx,
		cancel:     cancel,
		step:       0,
		outputChan: make(chan TaskOutput, taskOutputChanBuffer),
		statusChan: make(chan models.TaskStatus),
	}
}

func (tc *TaskContext) Context() context.Context {
	return tc.ctx
}

func (tc *TaskContext) println(msg string) {
	select {
	case <-tc.ctx.Done():
		return
	default:
	}

	// 阻塞直到 exec 接收到这一行或任务结束。缓冲区满时形成背压，
	// 而不是静默丢弃输出。
	select {
	case tc.outputChan <- TaskOutput{
		Step:   tc.step,
		Output: msg,
	}:
	case <-tc.ctx.Done():
	}
}

func (tc *TaskContext) nextStep() {
	tc.step++
}

// status 将任务的状态更新为指定的状态。
func (tc *TaskContext) status(status models.TaskStatus) {
	select {
	case <-tc.ctx.Done():
		return
	case tc.statusChan <- status:
	}
}

// done 结束 monitor 的运行并将标为已完成。该函数只能在任务返回之前调用。
func (tc *TaskContext) done() {
	tc.cancel(nil)
}

// throw 结束 monitor 的运行并标为失败。以首次 cause 为准，重复调用是空操作。
func (tc *TaskContext) throw(err error) {
	tc.cancel(err)
}
