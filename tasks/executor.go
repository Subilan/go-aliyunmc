package tasks

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/logs"
	"go-aliyunmc/sse"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
	"sync"
	"time"
)

var ErrTaskInterrupted = errors.New("任务中断")
var ErrTaskTypeExecuting = errors.New("同类任务正在执行中")

type Executor struct {
	// broker 用于管理当前任务的 SSE 连接，并向连接中的客户端广播任务状态更新和输出。
	broker *sse.Broker
	// task 指向当前正在执行的任务信息。
	//  - invariant: 只能在 monitor goroutine 中访问和修改 task，其余 goroutine 访问将导致竞态条件。
	task *models.Task
	// taskDefinition 用于存储当前执行器对应的任务定义，它在执行器的生命周期内保持不变。
	taskDefinition *TaskDefinition
	// stateLock 保护 interrupted 字段和 tc 字段的访问。
	stateLock sync.Mutex
	// tc 是任务的执行上下文，用于记录任务的执行进度、输出以及状态更新等信息。
	tc *TaskContext
	// interruptOnce 确保 Interrupt 方法的幂等性。
	interruptOnce sync.Once
	// interrupted 记录任务是否被触发过中断逻辑。
	interrupted bool
	// ctx 用于控制当前执行器的生命周期。
	ctx context.Context
	// cancel 用于取消当前执行器的执行。
	cancel context.CancelFunc
	// exlusiveType 记录当前任务是否属于独占类型，如果是独占类型，在任务结束前同类任务将无法执行。
	// 它的值相当于 taskDefinition.TypeExlusive。
	exclusiveType bool
}

func NewExecutor(def *TaskDefinition) *Executor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Executor{
		taskDefinition: def,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Interrupt 触发任务执行器的中断逻辑，monitor 将立即退出并将任务标记为失败。
func (e *Executor) Interrupt() {
	e.interruptOnce.Do(func() {
		e.stateLock.Lock()
		e.interrupted = true
		tc := e.tc
		e.stateLock.Unlock()

		if tc != nil {
			tc.throw(ErrTaskInterrupted)
		}
	})
}

// SubscribeOrFail 将 client 添加到当前任务执行器的 broker 订阅列表中，用于接收任务执行状态更新以及输出。
// 如果当前任务执行器的 broker 未初始化（常常发生于任务开始执行之前），则返回 false。
func (e *Executor) SubscribeOrFail(client *sse.Client) bool {
	if e.broker == nil {
		return false
	}
	e.broker.Register(client)
	return true
}

// RunTask 创建并开始执行一个任务，并将任务的触发者设置为 by（可以为 nil，表示由系统触发）。
//   - 如果返回的 error 不为 nil，则说明任务未能成功启动，调用者可以认为这个任务没有被执行；
//   - 如果 error 为 nil，则说明任务已成功启动，调用者可以通过返回的 *models.Task 获取到这个任务的 ID 以及其他相关信息。
func (e *Executor) RunTask(by *uint, args map[string]any) (*models.Task, error) {
	/** 初始化阶段，此阶段发生的错误被认为是 earlyExit **/
	if e.taskDefinition.Exclusive {
		if !TrySetExecutingType(e.taskDefinition.Type) {
			return nil, ErrTaskTypeExecuting
		}
		e.exclusiveType = true
	}

	earlyExitCleanup := func() {
		if e.exclusiveType {
			DeleteExecutingType(e.taskDefinition.Type)
			e.exclusiveType = false
		}
	}

	// 创建任务记录
	task, err := store.CreateTask(e.taskDefinition.Type, by)

	if err != nil {
		earlyExitCleanup()
		return nil, err
	}

	// 将任务标记为正在执行，并记录开始时间
	e.task = task
	now := time.Now()
	task.StartAt = &now
	task.Status = models.TaskStatusRunning

	if err := store.UpdateTask(task); err != nil {
		earlyExitCleanup()
		return nil, err
	}

	// 初始化 broker 和 TaskContext
	e.broker = sse.NewBroker()
	tc := NewTaskContextWithTimeout(e.taskDefinition.Timeout)
	e.stateLock.Lock()
	e.tc = tc
	interrupted := e.interrupted
	e.stateLock.Unlock()

	/** 运行阶段，从这里开始需要正确处理 goroutine 的退出 **/

	// 将当前的执行器注册到全局执行器映射中，
	// 以便其他 goroutine 可以通过任务 ID 获取到它。
	SetExecutor(task.ID, e)

	// 启动 broker 和 monitor goroutine。
	go e.broker.Run()
	go e.monitor()

	// 如果在启动任务的过程中，任务被触发过中断逻辑，则立即将任务标记为失败并退出。
	if interrupted {
		tc.throw(ErrTaskInterrupted)
		return task, nil
	}

	// 否则，在新的 goroutine 中执行任务定义中的函数 F，并在其 panic 后将任务标记为失败。
	go func() {
		defer func() {
			if r := recover(); r != nil {
				tc.throw(fmt.Errorf("任务执行panic: %v", r))
			}
		}()

		if err := e.taskDefinition.F(tc, args); err != nil {
			tc.throw(err)
			return
		}

		tc.done()
	}()

	return task, nil
}

type TaskSummary struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// updateAndBroadcast 是一个方便的封装，它尝试将 e.task 同步到数据库，并在没有出现差错的情况下广播 event。
func (e *Executor) updateAndBroadcast(event sse.Event) {
	if err := store.UpdateTask(e.task); err != nil {
		logs.Error("更新任务状态时出错: %v\n", err)
	} else {
		e.broker.Broadcast(event)
	}
}

// updateAndSummary 是一个方便的封装，它尝试将 e.task 同步到数据库，并在没有出现差错的情况下广播任务结束事件。
func (e *Executor) updateAndSummary() {
	if err := store.UpdateTask(e.task); err != nil {
		logs.Error("更新任务状态时出错: %v\n", err)
	} else {
		e.broker.Broadcast(sse.Event{
			Event: "task_done",
			Data: TaskSummary{
				Success: e.task.Status == models.TaskStatusSuccess,
				Error:   e.task.Error,
			},
		})
	}
}

func (e *Executor) Done() <-chan struct{} {
	return e.ctx.Done()
}

func (e *Executor) monitor() {
	defer e.broker.Stop()
	defer e.cancel()
	defer DeleteExecutor(e.task.ID)
	if e.exclusiveType {
		defer DeleteExecutingType(e.task.Type)
	}

	for {
		select {
		case <-e.tc.ctx.Done():
			cause := context.Cause(e.tc.ctx)

			if errors.Is(cause, context.DeadlineExceeded) {
				e.task.Status = models.TaskStatusFailed
				e.task.Error = "__TIMEOUT__"
			} else if errors.Is(cause, ErrTaskInterrupted) {
				e.task.Status = models.TaskStatusFailed
				e.task.Error = "__INTERRUPTED__"
				logs.Info("任务%d被中断", e.task.ID)
			} else if errors.Is(cause, context.Canceled) {
				e.task.Status = models.TaskStatusSuccess
			} else {
				e.task.Status = models.TaskStatusFailed
				e.task.Error = cause.Error()
			}
			now := time.Now()
			e.task.EndAt = &now
			e.updateAndSummary()
			return

		case status := <-e.tc.statusChan:
			e.task.Status = status
			e.updateAndBroadcast(sse.Event{
				Event: "task_status_update",
				Data:  status,
			})

		case output := <-e.tc.outputChan:
			e.task.Output += output.Output + "\n"
			e.task.Step = output.Step
			e.updateAndBroadcast(sse.Event{
				Event: "task_output",
				Data:  output,
			})
		}
	}
}
