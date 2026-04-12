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
//   - 如果已有数据则直接返回；若暂无数据则等待到有数据后返回。
//   - 若在获取到数据前超时，返回 error。
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

func SnapshotIsInstanceRunning() (bool, bool) {
	snapshot, ok := SnapshotInstanceStatus()
	return snapshot.Value == "Running", ok && snapshot.IsValid()
}

func StableSnapshotInstanceStatus(timeout time.Duration) (State[string], error) {
	return StableSnapshot[string](HSKeyInstanceStatus, timeout)
}

func StableSnapshotIsInstanceRunning(timeout time.Duration) (bool, error) {
	snapshot, err := StableSnapshot[string](HSKeyInstanceStatus, timeout)
	if err != nil {
		return false, err
	}
	return snapshot.IsValid() && snapshot.Value == "Running", nil
}

func SnapshotIsServerOnline() (bool, bool) {
	snapshot, ok := SnapshotServerStatus()
	return snapshot.Value.Online, ok && snapshot.IsValid()
}

func SnapshotIsServerOffline() (bool, bool) {
	snapshot, ok := SnapshotServerStatus()
	return !snapshot.Value.Online, ok && snapshot.IsValid()
}

func SnapshotBestEcsCandidate() (EcsCandidate, bool) {
	snapshot, ok := Snapshot[EcsCandidate](HSKeyBestEcsCandidate)
	return snapshot.Value, ok && snapshot.IsValid()
}
