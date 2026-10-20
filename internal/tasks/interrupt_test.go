package tasks

import (
	"testing"
	"time"
)

// Interrupt 只负责发出中断请求，不等待任务函数退出；停止确认由 Done 提供。
func TestInterruptDoesNotWaitForTaskFunc(t *testing.T) {
	funcExited := make(chan struct{})

	def := &TaskDefinition{
		Type: "interrupt_wait_test",
		F: func(tc *TaskContext, _ map[string]any) error {
			<-tc.Context().Done()
			// 模拟任务在收到取消信号后仍需要一点时间收尾。
			time.Sleep(100 * time.Millisecond)
			close(funcExited)
			return tc.Context().Err()
		},
	}

	executor := NewExecutor(def)
	task, err := executor.RunTask(nil, nil)
	if err != nil {
		t.Fatalf("run task failed: %v", err)
	}

	executor.Interrupt()

	select {
	case <-funcExited:
		t.Fatalf("Interrupt must not wait for the task function to exit")
	default:
	}

	waitDone(t, executor)

	select {
	case <-funcExited:
	default:
		t.Fatalf("Done closed before the task function exited")
	}

	// monitor 收尾完成后执行器应从全局表中移除，独占 key 也已释放。
	if _, ok := GetExecutor(task.ID); ok {
		t.Fatalf("executor should be unregistered after Done")
	}
}

// 任务函数不响应上下文时，Interrupt 仍然立即返回；Done 会一直等到它退出。
func TestInterruptReturnsImmediatelyWhenTaskIgnoresContext(t *testing.T) {
	release := make(chan struct{})

	def := &TaskDefinition{
		Type: "interrupt_timeout_test",
		F: func(tc *TaskContext, _ map[string]any) error {
			<-release
			return nil
		},
	}

	executor := NewExecutor(def)
	if _, err := executor.RunTask(nil, nil); err != nil {
		t.Fatalf("run task failed: %v", err)
	}

	returned := make(chan struct{})
	go func() {
		executor.Interrupt()
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatalf("Interrupt blocked while the task function ignored the context")
	}

	select {
	case <-executor.Done():
		t.Fatalf("Done closed while the task function was still running")
	default:
	}

	close(release)
	waitDone(t, executor)
}

// monitor 必须等任务函数退出后才写终态并注销执行器。
// 否则任务函数仍在操作实例时，独占锁就可能被下一个任务抢到。
func TestExecutorReleasedOnlyAfterTaskFuncExits(t *testing.T) {
	release := make(chan struct{})

	def := &TaskDefinition{
		Type: "interrupt_release_test",
		F: func(tc *TaskContext, _ map[string]any) error {
			<-tc.Context().Done()
			<-release
			return tc.Context().Err()
		},
	}

	executor := NewExecutor(def)
	task, err := executor.RunTask(nil, nil)
	if err != nil {
		t.Fatalf("run task failed: %v", err)
	}

	executor.Interrupt()

	// 中断已生效，但 F 仍被 release 阻塞。
	<-executor.tc.Context().Done()

	if _, ok := GetExecutor(task.ID); !ok {
		t.Fatal("executor should stay registered while the task function is still running")
	}
	select {
	case <-executor.Done():
		t.Fatal("executor finished before the task function exited")
	default:
	}

	close(release)
	waitDone(t, executor)

	if _, ok := GetExecutor(task.ID); ok {
		t.Fatal("executor should be unregistered after the task function exited")
	}
}

func waitDone(t *testing.T, executor *Executor) {
	t.Helper()
	select {
	case <-executor.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not finish in time")
	}
}
