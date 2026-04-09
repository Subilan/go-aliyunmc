package monitors

import (
	"context"
	"fmt"
	"sync"
)

// StateMonitor 表示一个针对某种状态的监控器，提供获取当前状态快照和订阅状态更新的功能。
type StateMonitor[T any] interface {
	Snapshot() snapshot[T]
	Subscribe() (update <-chan snapshot[T], unsubscribe func())
}

var emptyValueError = fmt.Errorf("值为空")
var missingTargetError = fmt.Errorf("目标不存在")

var (
	serverMonitor   *ServerStatusMonitor
	instanceMonitor *InstanceStatusMonitor
	fileSyncPoller  *FileSyncPoller
	autoArchiveIdle *AutoArchiveIdleMonitor
	initializeOnce  sync.Once
)

// MustInitialize 启动所有常驻 monitor。
func MustInitialize(ctx context.Context) {
	initializeOnce.Do(func() {
		serverMonitor = newServerStatusMonitor()
		instanceMonitor = newInstanceStatusMonitor()
		fileSyncPoller = newFileSyncPoller()
		autoArchiveIdle = newAutoArchiveIdleMonitor()

		go serverMonitor.run(ctx)
		go instanceMonitor.run(ctx)
		go fileSyncPoller.run(ctx)
		go autoArchiveIdle.run(ctx)
	})
}

// SnapshotServerStatus 返回当前 server status snapshot 的副本。
func SnapshotServerStatus() snapshot[ServerStatusState] {
	if serverMonitor == nil {
		return snapshot[ServerStatusState]{}
	}
	return serverMonitor.Snapshot()
}

// ServerStatus 返回 server status monitor 的接口视图。
func ServerStatus() StateMonitor[ServerStatusState] {
	return serverMonitor
}

// InstanceStatus 返回 instance status monitor 的接口视图。
func InstanceStatus() StateMonitor[string] {
	return instanceMonitor
}

// SubscribeServerSnapshot 订阅 server status 的更新。
func SubscribeServerSnapshot() (<-chan snapshot[ServerStatusState], func()) {
	if serverMonitor == nil {
		ch := make(chan snapshot[ServerStatusState])
		close(ch)
		return ch, func() {}
	}
	return serverMonitor.Subscribe()
}

// SubscribeInstanceSnapshot 订阅 instance status 的更新。
func SubscribeInstanceSnapshot() (<-chan snapshot[string], func()) {
	if instanceMonitor == nil {
		ch := make(chan snapshot[string])
		close(ch)
		return ch, func() {}
	}
	return instanceMonitor.Subscribe()
}
