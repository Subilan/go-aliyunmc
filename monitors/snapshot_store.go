package monitors

import (
	"errors"
	"fmt"
	"go-aliyunmc/logs"
	"sync"
	"time"
)

type snapshotStore[T comparable] struct {
	snapshot[T]
	mu sync.RWMutex
}

type snapshot[T any] struct {
	// Value 是 snapshot 的实际值。
	Value T

	// UpdatedAt 是 snapshot 的更新时间。
	UpdatedAt time.Time

	// Error 表示在获取 Value 过程中发生的错误。如果 Error 不为 nil，则 Value 可能不可靠。
	Error error
}

// Store 将新的 snapshot 存储到 store 中，并在 snapshot 发生变化时通过 hub 广播更新。
func (s *snapshotStore[T]) Store(value T, hub *hub[snapshot[T]]) {
	s.mu.Lock()
	changed := s.Value != value
	s.Value = value
	s.UpdatedAt = time.Now()
	s.Error = nil
	snapshot := s.snapshot
	s.mu.Unlock()

	if changed {
		logs.Info(fmt.Sprintf("[snapshotStore] 推送值的更新 %v", value))
		hub.Broadcast(snapshot)
	}
}

// StoreError 将错误存储到 store 中
func (s *snapshotStore[T]) StoreError(err error, hub *hub[snapshot[T]]) {
	s.mu.Lock()
	changed := s.Error == nil || !errors.Is(s.Error, err)
	s.Error = err
	s.UpdatedAt = time.Now()
	snapshot := s.snapshot
	s.mu.Unlock()

	if changed {
		logs.Info(fmt.Sprintf("[snapshotStore] 推送错误：%s", err.Error()))
		hub.Broadcast(snapshot)
	}
}

// Snapshot 返回 snapshot 的副本。
func (s *snapshotStore[T]) Snapshot() snapshot[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}
