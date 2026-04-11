package states

import (
	"errors"
	"go-aliyunmc/logs"
	"sync"
	"time"
)

type StateStore[T comparable] struct {
	State[T]
	mu sync.RWMutex
}

type State[T any] struct {
	// Value 是最近一次成功获取到的实际值。
	Value T

	// UpdatedAt 是最近一次更新的时间。
	UpdatedAt time.Time

	// Error 表示在获取 Value 过程中发生的错误。如果 Error 不为 nil，则 Value 可能不可靠。
	Error error
}

// Store 将新的 snapshot 存储到 store 中，并在 snapshot 发生变化时通过 hub 广播更新。
func (s *StateStore[T]) Store(value T, hub *Hub[State[T]], logger *logs.PrefixedLogger) {
	s.mu.Lock()
	// 认为值发生更新的条件：
	// 1. 新值与旧值不同；或者
	// 2. 之前存在错误，现在没有错误了（不论值是否改变）
	changed := s.Value != value || s.Error != nil
	s.Value = value
	s.UpdatedAt = time.Now()
	s.Error = nil
	snapshot := s.State
	s.mu.Unlock()

	if changed {
		if logger != nil {
			logger.Info("[stateStore] 更新值：%v", value)
		}
		hub.Broadcast(snapshot)
	}
}

// StoreError 将错误存储到 store 中，并在错误发生变化时通过 hub 广播更新。
func (s *StateStore[T]) StoreError(err error, hub *Hub[State[T]], logger *logs.PrefixedLogger) {
	s.mu.Lock()
	// 认为错误发生更新的条件：
	// 1. 之前没有错误，现在有了错误；或者
	// 2. 之前有错误，现在的错误与之前的错误不同了
	changed := s.Error == nil || !errors.Is(s.Error, err)
	s.Error = err
	s.UpdatedAt = time.Now()
	snapshot := s.State
	s.mu.Unlock()

	if changed {
		if logger != nil {
			logger.Info("[stateStore] 更新错误：%s", err.Error())
		}
		hub.Broadcast(snapshot)
	}
}

// Snapshot 返回 state 当前的副本。
func (s *StateStore[T]) Snapshot() State[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}
