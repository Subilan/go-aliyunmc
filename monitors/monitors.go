package monitors

import (
	"context"
	"fmt"
	"sync"
	"time"
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
	initializeOnce  sync.Once
)

// MustInitialize 启动所有常驻 monitor。
func MustInitialize(ctx context.Context) {
	initializeOnce.Do(func() {
		serverMonitor = newServerStatusMonitor()
		instanceMonitor = newInstanceStatusMonitor()
		fileSyncPoller = newFileSyncPoller()

		go serverMonitor.run(ctx)
		go instanceMonitor.run(ctx)
		go fileSyncPoller.run(ctx)
	})
}

// SnapshotServerStatus 返回当前 server status snapshot 的副本。
func SnapshotServerStatus() snapshot[ServerStatusState] {
	if serverMonitor == nil {
		return snapshot[ServerStatusState]{}
	}
	return serverMonitor.Snapshot()
}

// SnapshotInstanceStatus 返回当前 instance status snapshot 的副本。
func SnapshotInstanceStatus() snapshot[string] {
	if instanceMonitor == nil {
		return snapshot[string]{}
	}
	return instanceMonitor.Snapshot()
}

func SnapshotIsInstanceRunning() bool {
	snapshot := SnapshotInstanceStatus()
	return snapshot.Error == nil && snapshot.Value == "Running"
}

// StableSnapshotIsInstanceRunning 在 instance monitor 尚未产出首个快照时等待至多 timeout。
// 一旦有快照（无论值是 Running 还是其他状态），行为与 SnapshotIsInstanceRunning 保持一致。
func StableSnapshotIsInstanceRunning(timeout time.Duration) bool {
	if instanceMonitor == nil {
		return SnapshotIsInstanceRunning()
	}

	current := SnapshotInstanceStatus()
	if !current.UpdatedAt.IsZero() {
		return current.Error == nil && current.Value == "Running"
	}

	if timeout <= 0 {
		return SnapshotIsInstanceRunning()
	}

	ch, unsubscribe := SubscribeInstanceSnapshot()
	defer unsubscribe()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		current = SnapshotInstanceStatus()
		if !current.UpdatedAt.IsZero() {
			return current.Error == nil && current.Value == "Running"
		}

		select {
		case <-timer.C:
			return SnapshotIsInstanceRunning()
		case <-ch:
		}
	}
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
