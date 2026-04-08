package monitors

// SnapshotMonitor 表示一个可以读取当前 snapshot 并订阅更新的 monitor。
type SnapshotMonitor[T any] interface {
	Snapshot() snapshot[T]
	Subscribe() (update <-chan snapshot[T], unsubscribe func())
}
