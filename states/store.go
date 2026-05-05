package states

import (
	"errors"
	"go-aliyunmc/log_util"
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

// IsValid 返回这个状态的值目前是否有效。一个状态被认为是有效的，当且仅当它没有错误，并且 UpdatedAt 不是零值。
// 注意，IsValid 只反映当前状态是否有效，与先前状态以及未来状态无关。
func (state *State[T]) IsValid() bool {
	return state.Error == nil && !state.UpdatedAt.IsZero()
}

// ForceStore 无条件地将值存储到 store 中，并更新时间。
func (s *StateStore[T]) ForceStore(value T, hub *Hub[State[T]]) {
	s.mu.Lock()
	s.Value = value
	s.UpdatedAt = time.Now()
	s.Error = nil
	snapshot := s.State
	s.mu.Unlock()

	if hub != nil {
		hub.Broadcast(snapshot)
	}
}

// Store 将新的 snapshot 存储到 store 中。如果值没有发生变化，则不做任何操作。返回值表示是否发生了变化。
func (s *StateStore[T]) Store(value T, hub *Hub[State[T]], logger *log_util.NamedLogger) bool {
	s.mu.Lock()
	// 认为值发生更新的条件：
	// 1. 新值与旧值不同；或者
	// 2. 之前存在错误，现在没有错误了（不论值是否改变）
	changed := s.Value != value || s.Error != nil
	if changed {
		s.Value = value
		s.UpdatedAt = time.Now()
		s.Error = nil
	}
	snapshot := s.State
	s.mu.Unlock()

	if changed && hub != nil {
		if logger != nil {
			logger.Info("[stateStore] broadcast value update: %v", value)
		}
		hub.Broadcast(snapshot)
	}

	return changed
}

// StoreError 将错误存储到 store 中。如果错误没有发生变化，则不做任何操作。返回值表示是否发生了变化。
func (s *StateStore[T]) StoreError(err error, hub *Hub[State[T]], logger *log_util.NamedLogger) bool {
	s.mu.Lock()
	// 认为错误发生更新的条件：
	// 1. 之前没有错误，现在有了错误；或者
	// 2. 之前有错误，现在的错误与之前的错误不同了
	changed := s.Error == nil || !errors.Is(s.Error, err)
	if changed {
		s.Error = err
		s.UpdatedAt = time.Now()
	}
	snapshot := s.State
	s.mu.Unlock()

	if changed && hub != nil {
		if logger != nil {
			logger.Info("[stateStore] broadcast error update: %s", err.Error())
		}
		hub.Broadcast(snapshot)
	}
	return changed
}

// Snapshot 返回 state 当前的副本。
func (s *StateStore[T]) Snapshot() State[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}
