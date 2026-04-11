package states

import (
	"fmt"
	"time"
)

// Snapshot 获取指定 key 的当前状态快照。如果 key 不存在，返回 false。
func Snapshot[T comparable](key string) (State[T], bool) {
	store, ok := GetRecordedHubbedStore[T](key)
	if !ok {
		return State[T]{}, false
	}
	return store.Snapshot(), true
}

// StableSnapshot 获取指定 key 的当前状态快照。
//  - 如果已有数据则直接返回；若暂无数据则等待到有数据后返回。
//  - 若在获取到数据前超时，返回 error。
func StableSnapshot[T comparable](key string, timeout time.Duration) (State[T], error) {
	store, ok := GetRecordedHubbedStore[T](key)

	if !ok {
		return State[T]{}, fmt.Errorf("state store not found: %s", key)
	}

	current := store.Snapshot()
	if !current.UpdatedAt.IsZero() {
		return current, nil
	}

	if timeout <= 0 {
		return State[T]{}, fmt.Errorf("wait stable snapshot timeout: %s", timeout)
	}

	ch, unsubscribe := store.Subscribe()
	defer unsubscribe()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		current = store.Snapshot()
		if !current.UpdatedAt.IsZero() {
			return current, nil
		}

		select {
		case <-timer.C:
			return State[T]{}, fmt.Errorf("wait stable snapshot timeout: %s", timeout)
		case <-ch:
		}
	}
}

func SnapshotInstanceStatus() (State[string], bool) {
	return Snapshot[string](HSKeyInstanceStatus)
}

func SnapshotServerStatus() (State[ServerStatusState], bool) {
	return Snapshot[ServerStatusState](HSKeyServerStatus)
}

func SnapshotIsInstanceRunning() bool {
	snapshot, ok := SnapshotInstanceStatus()
	return ok && snapshot.Error == nil && snapshot.Value == "Running"
}

func StableSnapshotInstanceStatus(timeout time.Duration) (State[string], error) {
	return StableSnapshot[string](HSKeyInstanceStatus, timeout)
}

func StableSnapshotIsInstanceRunning(timeout time.Duration) bool {
	snapshot, err := StableSnapshot[string](HSKeyInstanceStatus, timeout)
	return err == nil && snapshot.Error == nil && snapshot.Value == "Running"
}

func SnapshotIsServerOnline() bool {
	snapshot, ok := SnapshotServerStatus()
	return ok && snapshot.Error == nil && snapshot.Value.Online
}

func SnapshotIsServerOffline() bool {
	snapshot, ok := SnapshotServerStatus()
	return ok && snapshot.Error == nil && !snapshot.Value.Online
}