package tasks

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Subilan/go-aliyunmc/log_util"
	"github.com/Subilan/go-aliyunmc/pubsub"
	"github.com/Subilan/go-aliyunmc/sse"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
)

var ErrTaskInterrupted = errors.New("任务中断")
var ErrTaskTypeExecuting = errors.New("同类任务正在执行中")

type Executor struct {
	// hub 用于管理当前任务的 SSE 订阅者，并向连接中的客户端广播任务状态更新和输出。
	hub *pubsub.Hub[sse.Event]
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
	// taskID 是任务创建后固定的 ID，供 monitor 之外的 goroutine 读取。
	taskID uint
	// lastSeq 是当前任务已分配的最后一条输出序号。monitor 是唯一写入者。
	lastSeq atomic.Uint64
	// flushReq 请求 monitor 立即把待写入的输出行落库，并回传已落库的最大序号。
	flushReq chan chan uint64
}

// NewExecutor 创建一个新的 Executor 实例，并为其分配一个 context.Backgound() 上下文。传入的 def 不得为 nil。
func NewExecutor(def *TaskDefinition) *Executor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Executor{
		taskDefinition: def,
		ctx:            ctx,
		cancel:         cancel,
		flushReq:       make(chan chan uint64),
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

// Subscribe 将 client 注册为当前任务的事件订阅者，并把订阅 channel 接入客户端。
// 返回的 unsubscribe 用于在连接结束后释放订阅；如果执行器尚未初始化广播器，
// 或广播器已经关闭，则返回 false。
func (e *Executor) Subscribe(client *sse.Client) (func(), bool) {
	if e.hub == nil {
		return func() {}, false
	}

	events, unsubscribe, ok := e.hub.Subscribe(taskEventBuffer, client.Close)
	if !ok {
		return func() {}, false
	}

	client.Attach(events)
	return unsubscribe, true
}

// RunTask 创建并开始执行一个任务，并将任务的触发者设置为 by（可以为 nil，表示由系统触发）。
//   - 如果返回的 error 不为 nil，则说明任务未能成功启动，调用者可以认为这个任务没有被执行；
//   - 如果 error 为 nil，则说明任务已成功启动，调用者可以通过返回的 *models.Task 获取到这个任务的 ID 以及其他相关信息。
func (e *Executor) RunTask(by *uint, args map[string]any) (*models.Task, error) {
	if e.taskDefinition.Exclusive {
		if _, ok := GetExclusiveExecutor(e.taskDefinition.Type); ok {
			return nil, ErrTaskTypeExecuting
		}
		e.exclusiveType = true
	}

	// 创建任务记录
	task, err := store.CreateTask(e.taskDefinition.Type, by)

	if err != nil {
		return nil, err
	}

	// 将任务标记为正在执行，并记录开始时间
	e.task = task
	e.taskID = task.ID
	now := time.Now()
	task.StartAt = &now
	task.Status = models.TaskStatusRunning

	if err := store.UpdateTask(task); err != nil {
		return nil, err
	}

	// 初始化广播器和 TaskContext
	e.hub = pubsub.NewHub[sse.Event]()
	tc := NewTaskContextWithTimeout(e.taskDefinition.Timeout)
	e.stateLock.Lock()
	e.tc = tc
	interrupted := e.interrupted
	e.stateLock.Unlock()

	/** 运行阶段，从这里开始需要正确处理 goroutine 的退出 **/

	// 将当前的执行器注册到全局执行器映射中，
	// 以便其他 goroutine 可以通过任务 ID 获取到它。
	SetExecutor(task.ID, e)

	if e.exclusiveType {
		SetExclusiveExecutor(task.Type, e)
	}

	// 在启动 monitor 之前复制一份任务快照返回给调用方。
	// e.task 归 monitor goroutine 所有，调用方（例如 HTTP handler 在 JSON 序列化时）
	// 不得直接读取它，否则会与 monitor 的写入产生数据竞争。
	snapshot := *task

	// 启动 monitor goroutine。
	go e.monitor()

	// 如果在启动任务的过程中，任务被触发过中断逻辑，则立即将任务标记为失败并退出。
	if interrupted {
		tc.throw(ErrTaskInterrupted)
		return &snapshot, nil
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

	return &snapshot, nil
}

// RunTaskAndWait 启动任务并阻塞等待任务完成。
// 返回值中的 error 表示任务最终执行结果或者启动错误。
func (e *Executor) RunTaskAndWait(by *uint, args map[string]any) (*models.Task, error) {
	task, err := e.RunTask(by, args)
	if err != nil {
		return nil, err
	}

	waitErr := <-e.Wait()

	// monitor 已退出，e.task 不会再被修改，返回最终状态。
	if e.task != nil {
		final := *e.task
		task = &final
	}

	return task, waitErr
}

type TaskSummary struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

const (
	// taskEventBuffer 是每个 SSE 订阅者的事件缓冲大小。
	taskEventBuffer = 100
	// taskOutputFlushInterval 是输出写库的最大延迟。
	taskOutputFlushInterval = 500 * time.Millisecond
	// taskOutputFlushLines 是触发提前写库的未落库行数。
	taskOutputFlushLines = 256
	// taskOutputInsertBatch 是单次 INSERT 写入的行数上限。
	taskOutputInsertBatch = 200
)

func (e *Executor) Done() <-chan struct{} {
	return e.ctx.Done()
}

// Wait 返回一个只读错误通道。任务结束时会向通道发送最终错误（成功时为 nil）并关闭通道。
func (e *Executor) Wait() <-chan error {
	ch := make(chan error, 1)
	go func() {
		<-e.Done()
		ch <- e.finalError()
		close(ch)
	}()
	return ch
}

func (e *Executor) finalError() error {
	if e.task == nil {
		return nil
	}

	if e.task.Status != models.TaskStatusFailed {
		return nil
	}

	return fmt.Errorf("%s", e.task.Error)
}

func (e *Executor) monitor() {
	defer e.cancel()
	defer DeleteExecutor(e.task.ID)
	if e.exclusiveType {
		defer DeleteExclusiveExecutor(e.task.Type)
	}
	// 无论任务如何结束，都保证广播器被关闭，订阅者不会残留。
	defer e.hub.Close()

	flushTicker := time.NewTicker(taskOutputFlushInterval)
	defer flushTicker.Stop()

	// needsPersist 表示内存中的任务状态还没有写入数据库。
	needsPersist := false

	// 输出按行缓冲，批量写入 task_output_lines。flushPos 之前的行已经落库。
	var (
		pendingLines []models.TaskOutputLine
		flushPos     int
		persistedSeq uint64
	)

	// flushTask 将内存中的任务状态同步到数据库。失败时保留 needsPersist，
	// 周期 flush 会在下一个 tick 再次尝试。
	flushTask := func() {
		if !needsPersist {
			return
		}
		if err := store.UpdateTask(e.task); err != nil {
			log_util.Error("更新任务状态时出错: %v\n", err)
			return
		}
		needsPersist = false
	}

	// flushOutput 将尚未落库的输出行分批写入数据库。失败时保留未写入的部分，
	// 下一个 tick 或下一次显式 flush 会继续重试。
	flushOutput := func() {
		for flushPos < len(pendingLines) {
			end := min(flushPos+taskOutputInsertBatch, len(pendingLines))
			if err := store.AppendTaskOutputLines(pendingLines[flushPos:end]); err != nil {
				log_util.Error("写入任务输出时出错: %v\n", err)
				return
			}
			persistedSeq = pendingLines[end-1].Seq
			flushPos = end
		}

		if flushPos == len(pendingLines) {
			pendingLines = pendingLines[:0]
			flushPos = 0
		}
	}

	handleOutput := func(output TaskOutput) {
		seq := e.lastSeq.Add(1)
		output.ID = seq

		// 只有步骤变化时才需要更新任务行。
		if e.task.Step != output.Step {
			e.task.Step = output.Step
			needsPersist = true
		}

		pendingLines = append(pendingLines, models.TaskOutputLine{
			TaskID: e.task.ID,
			Seq:    seq,
			Step:   output.Step,
			Output: output.Output,
		})

		// 先广播；持久化失败不应影响实时输出。
		e.hub.Broadcast(sse.Event{
			ID:    strconv.FormatUint(seq, 10),
			Event: "task_output",
			Data:  output,
		})

		if len(pendingLines)-flushPos >= taskOutputFlushLines {
			flushOutput()
		}
	}

	for {
		select {
		case <-e.tc.ctx.Done():
			// 任务结束时 outputChan 中可能仍有已入队的输出，先全部消费再收尾。
		drain:
			for {
				select {
				case output := <-e.tc.outputChan:
					handleOutput(output)
				default:
					break drain
				}
			}

			// 终止前必须把剩余输出落库，回放才能看到完整历史。
			flushOutput()

			cause := context.Cause(e.tc.ctx)

			if errors.Is(cause, context.DeadlineExceeded) {
				e.task.Status = models.TaskStatusFailed
				e.task.Error = "__TIMEOUT__"
			} else if errors.Is(cause, ErrTaskInterrupted) {
				e.task.Status = models.TaskStatusFailed
				e.task.Error = "__INTERRUPTED__"
				log_util.Info("任务%d被中断", e.task.ID)
			} else if errors.Is(cause, context.Canceled) {
				e.task.Status = models.TaskStatusSuccess
			} else {
				e.task.Status = models.TaskStatusFailed
				e.task.Error = cause.Error()
			}
			now := time.Now()
			e.task.EndAt = &now
			needsPersist = true
			// 终态只尝试落库一次。失败时记日志，不阻塞 task_done 的广播；
			// 持久化重试应由统一的写库策略处理，而不是在这个分支里单独实现。
			flushTask()

			// task_done 使用 lastSeq+1 作为事件 ID：客户端凭它判断历史已经消费完毕，
			// 重连时服务端可以直接返回 204 停止重连。
			e.hub.Broadcast(taskDoneEvent(
				e.task.Status == models.TaskStatusSuccess,
				e.task.Error,
				e.lastSeq.Load()+1,
			))
			return

		case status := <-e.tc.statusChan:
			e.task.Status = status
			// 先广播，再写库；写库失败不回退已发出的状态。
			e.hub.Broadcast(sse.Event{
				Event: "task_status_update",
				Data:  status,
			})
			needsPersist = true
			flushTask()

		case output := <-e.tc.outputChan:
			handleOutput(output)

		case reply := <-e.flushReq:
			// 订阅者需要精确的回放边界：先把当前缓冲全部落库，再回传序号。
			flushOutput()
			flushTask()
			reply <- persistedSeq

		case <-flushTicker.C:
			flushOutput()
			flushTask()
		}
	}
}
