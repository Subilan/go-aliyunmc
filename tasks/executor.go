package tasks

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/logs"
	"go-aliyunmc/sse"
	"go-aliyunmc/store/models"
	"time"
)

type Executor struct {
	broker         *sse.Broker
	tc             *TaskContext
	taskDefinition *TaskDefinition
	interrupt      chan struct{}
}

func NewExecutor(def *TaskDefinition) *Executor {
	return &Executor{
		taskDefinition: def,
		interrupt:      make(chan struct{}),
	}
}

// Interrupt 触发任务执行器的中断逻辑，monitor 将立即退出并将任务标记为失败。
func (e *Executor) Interrupt() {
	e.interrupt <- struct{}{}
	close(e.interrupt)
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

// RunTask 创建并开始执行这个任务
func (e *Executor) RunTask(by *uint) (*models.Task, error) {
	if e.taskDefinition.TypeExclusive {
		if IsTypeExecuting(e.taskDefinition.Type) {
			return nil, fmt.Errorf("同类任务正在执行中")
		}

		SetExecutingType(e.taskDefinition.Type)
	}

	task, err := CreateTask(e.taskDefinition.Type, by)

	if err != nil {
		return nil, err
	}

	now := time.Now()
	task.StartAt = &now
	task.Status = models.TaskStatusRunning
	if err := UpdateTask(task); err != nil {
		return nil, err
	}

	e.broker = sse.NewBroker()
	e.tc = NewTaskContext(task, e.taskDefinition.Timeout)
	SetExecutor(task.ID, e)

	go e.broker.Run()
	go e.monitor()
	go e.taskDefinition.F(e.tc)

	return task, nil
}

type TaskSummary struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func (e *Executor) updateAndBroadcast(event sse.Event) {
	e.tc.taskLock.Lock()
	defer e.tc.taskLock.Unlock()

	if err := UpdateTask(e.tc.task); err != nil {
		logs.Error("更新任务状态时出错: %v\n", err)
	} else {
		e.broker.Broadcast(event)
	}
}

func (e *Executor) updateAndSummary() {
	e.tc.taskLock.Lock()
	defer e.tc.taskLock.Unlock()

	if err := UpdateTask(e.tc.task); err != nil {
		logs.Error("更新任务状态时出错: %v\n", err)
	} else {
		e.broker.Broadcast(sse.Event{
			Event: "task_done",
			Data: TaskSummary{
				Success: e.tc.task.Status == models.TaskStatusSuccess,
				Error:   e.tc.task.Error,
			},
		})
	}
}

func (e *Executor) Done() <-chan struct{} {
	return e.tc.ctx.Done()
}

func (e *Executor) monitor() {
	defer close(e.tc.outputChan)
	defer close(e.tc.statusChan)
	defer e.broker.Stop()
	defer e.tc.cancel()
	defer DeleteExecutor(e.tc.task.ID)
	defer DeleteExecutingType(e.tc.task.Type)

	tc := e.tc

	for {
		select {
		case <-e.interrupt:
			tc.taskLock.Lock()
			tc.task.Status = models.TaskStatusFailed
			tc.task.Error = "中断"
			now := time.Now()
			tc.task.EndAt = &now
			logs.Info("任务%d被中断", tc.task.ID)
			tc.taskLock.Unlock()
			e.updateAndSummary()
			return

		case <-e.tc.ctx.Done():
			tc.taskLock.Lock()
			if errors.Is(e.tc.ctx.Err(), context.DeadlineExceeded) {
				tc.task.Status = models.TaskStatusFailed
				tc.task.Error = "超时"
			}
			now := time.Now()
			tc.task.EndAt = &now
			tc.taskLock.Unlock()
			e.updateAndSummary()
			return

		case status := <-e.tc.statusChan:
			tc.taskLock.Lock()
			tc.task.Status = status
			tc.taskLock.Unlock()
			e.updateAndBroadcast(sse.Event{
				Event: "task_status_update",
				Data:  status,
			})

		case output := <-e.tc.outputChan:
			tc.taskLock.Lock()
			tc.task.Output += output.Output + "\n"
			tc.task.Step = output.Step
			tc.taskLock.Unlock()
			e.updateAndBroadcast(sse.Event{
				Event: "task_output",
				Data:  output,
			})
		}
	}
}
