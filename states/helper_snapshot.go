package states

import "time"

// Snapshot 获取指定 key 的当前状态快照。如果 key 不存在，返回 false。
func Snapshot[T comparable](key string) (State[T], bool) {
	store, ok := GetRecordedHubbedStore[T](key)
	if !ok {
		return State[T]{}, false
	}
	return store.Snapshot(), true
}

// StableSnapshot 获取指定 key 的当前状态快照，并且会阻塞到有快照更新或者超时。如果 key 不存在，返回 false。
func StableSnapshot[T comparable](key string, timeout time.Duration) (State[T], bool) {
	store, ok := GetRecordedHubbedStore[T](key)

	if !ok {
		return State[T]{}, false
	}

	current := store.Snapshot()
	if !current.UpdatedAt.IsZero() {
		return current, true
	}

	if timeout <= 0 {
		return current, true
	}

	ch, unsubscribe := store.Subscribe()
	defer unsubscribe()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		current = store.Snapshot()
		if !current.UpdatedAt.IsZero() {
			return current, true
		}

		select {
		case <-timer.C:
			return store.Snapshot(), true
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

func StableSnapshotIsInstanceRunning(timeout time.Duration) bool {
	snapshot, ok := StableSnapshot[string](HSKeyInstanceStatus, timeout)
	return ok && snapshot.Error == nil && snapshot.Value == "Running"
}

func SnapshotIsServerOnline() bool {
	snapshot, ok := SnapshotServerStatus()
	return ok && snapshot.Error == nil && snapshot.Value.Online
}

func SnapshotIsServerOffline() bool {
	snapshot, ok := SnapshotServerStatus()
	return ok && snapshot.Error == nil && !snapshot.Value.Online
}