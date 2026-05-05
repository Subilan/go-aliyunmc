package states

import (
	"fmt"
	"go-aliyunmc/log_util"
	"time"
)

// HubbedStore 是一个将 StateStore 和 Hub 结合在一起的结构，简化了状态存储和订阅的使用。
type HubbedStore[T comparable] struct {
	st  *StateStore[T]
	hub *Hub[State[T]]
}

// NewHubbedStore 创建一个新的 HubbedStore。
func NewHubbedStore[T comparable]() *HubbedStore[T] {
	return &HubbedStore[T]{
		st:  &StateStore[T]{},
		hub: NewHub[State[T]](),
	}
}

// Store 将新的 snapshot 存储到 store 中，并在 snapshot 发生变化时通过 hub 广播更新。返回值表示是否发生了变化。
func (s *HubbedStore[T]) Store(value T, logger *log_util.NamedLogger) bool {
	return s.st.Store(value, s.hub, logger)
}

func (s *HubbedStore[T]) ForceStore(value T, logger *log_util.NamedLogger) {
	s.st.ForceStore(value, s.hub)
}

// StoreError 将错误存储到 store 中，并在错误发生变化时通过 hub 广播更新。
func (s *HubbedStore[T]) StoreError(err error, logger *log_util.NamedLogger) bool {
	return s.st.StoreError(err, s.hub, logger)
}

// Snapshot 返回当前状态的副本。
func (s *HubbedStore[T]) Snapshot() State[T] {
	return s.st.Snapshot()
}

// Subscribe 订阅状态更新，返回一个接收状态更新的 channel 和一个取消订阅的函数。
func (s *HubbedStore[T]) Subscribe() (<-chan State[T], func()) {
	return s.hub.Subscribe()
}

// WaitSnapshot 等待首次数据到达后返回快照。若已有数据则直接返回，否则等待更新或超时。
func (s *HubbedStore[T]) WaitSnapshot(timeout time.Duration) (State[T], error) {
	current := s.Snapshot()
	if !current.UpdatedAt.IsZero() {
		return current, nil
	}

	if timeout <= 0 {
		return State[T]{}, fmt.Errorf("wait stable snapshot timeout: %s", timeout)
	}

	ch, unsubscribe := s.Subscribe()
	defer unsubscribe()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		current = s.Snapshot()
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
