package monitors

import (
	"context"
	"fmt"
	"sync"
)

var emptyValueError = fmt.Errorf("值为空")
var missingTargetError = fmt.Errorf("目标不存在")

var (
	serverMonitor   *ServerStatusMonitor
	instanceMonitor *InstanceStatusMonitor
	initializeOnce  sync.Once
)

// MustInitialize 启动所有常驻 monitor。
func MustInitialize(ctx context.Context) {
	initializeOnce.Do(func() {
		serverMonitor = newServerStatusMonitor(ServerStatusC)
		instanceMonitor = newInstanceStatusMonitor(InstanceStatusC)

		go serverMonitor.run(ctx)
		go instanceMonitor.run(ctx)
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

// ServerStatus 返回 server status monitor 的接口视图。
func ServerStatus() SnapshotMonitor[ServerStatusState] {
	return serverMonitor
}

// InstanceStatus 返回 instance status monitor 的接口视图。
func InstanceStatus() SnapshotMonitor[string] {
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
