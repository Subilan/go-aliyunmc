package tasks

import (
	"context"
	"go-aliyunmc/store/models"
	"sync"
	"time"
)

type TaskContext struct {
	ctx        context.Context
	cancel     context.CancelFunc
	task       *models.Task
	taskLock   sync.Mutex
	step       uint
	outputChan chan TaskOutput
	statusChan chan models.TaskStatus
}

func NewTaskContext(task *models.Task, timeout time.Duration) *TaskContext {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))

	return &TaskContext{
		ctx:        ctx,
		cancel:     cancel,
		task:       task,
		step:       0,
		outputChan: make(chan TaskOutput),
		statusChan: make(chan models.TaskStatus),
	}
}

func (tc *TaskContext) Context() context.Context {
	return tc.ctx
}

func (tc *TaskContext) println(msg string) {
	tc.outputChan <- TaskOutput{
		Step:   tc.step,
		Output: msg,
	}
}

func (tc *TaskContext) nextStep() {
	tc.step++
}

func (tc *TaskContext) status(status models.TaskStatus) {
	tc.statusChan <- status
}

func (tc *TaskContext) done() {
	tc.status(models.TaskStatusSuccess)
	tc.cancel()
}

func (tc *TaskContext) setError(err string) {
	tc.taskLock.Lock()
	defer tc.taskLock.Unlock()
	tc.task.Error = err
}

func (tc *TaskContext) throw(err error) {
	tc.setError(err.Error())
	tc.status(models.TaskStatusFailed)
	tc.cancel()
}
