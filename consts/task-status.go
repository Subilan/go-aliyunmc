package consts

// TaskStatus 表示系统中的一个任务的运行状态
type TaskStatus string

const (
	// TaskStatusRunning 表示任务正在进行中
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusSuccess 表示任务已经成功结束。这是表示任务成功的一个唯一状态。
	TaskStatusSuccess TaskStatus = "success"
	// TaskStatusFailed 表示任务因为某种原因失败了。
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled 表示任务被主动取消
	TaskStatusCancelled TaskStatus = "cancelled"
	// TaskStatusTimedOut 表示任务超出了上下文的超时显示，被取消
	TaskStatusTimedOut TaskStatus = "timed_out"
)
