package states

import (
	"go-aliyunmc/log_util"
	"sync"
)

// HubbedStore 是一个将 StateStore 和 Hub 结合在一起的结构，简化了状态存储和订阅的使用。
type HubbedStore[T comparable] struct {
	st  *StateStore[T]
	hub *Hub[State[T]]
}

var hubbedStores = make(map[string]any)
var hubbedStoresMu sync.RWMutex

// NewHubbedStore 创建一个新的 HubbedStore。
func NewHubbedStore[T comparable]() *HubbedStore[T] {
	return &HubbedStore[T]{
		st:  &StateStore[T]{},
		hub: NewHub[State[T]](),
	}
}

// NewRecordedHubbedStore 创建一个新的 HubbedStore，并将其注册到全局 hubbedStores 中以便后续读取。
func NewRecordedHubbedStore[T comparable](key string) *HubbedStore[T] {
	if store, ok := GetRecordedHubbedStore[T](key); ok {
		return store
	}

	store := NewHubbedStore[T]()
	hubbedStoresMu.Lock()
	defer hubbedStoresMu.Unlock()
	hubbedStores[key] = store
	return store
}

func GetRecordedHubbedStore[T comparable](key string) (*HubbedStore[T], bool) {
	hubbedStoresMu.RLock()
	defer hubbedStoresMu.RUnlock()
	store, ok := hubbedStores[key]
	if !ok {
		return nil, false
	}
	castedStore, ok := store.(*HubbedStore[T])
	return castedStore, ok
}

func DeleteRecordedHubbedStore(key string) {
	hubbedStoresMu.Lock()
	defer hubbedStoresMu.Unlock()
	delete(hubbedStores, key)
}

// Store 将新的 snapshot 存储到 store 中，并在 snapshot 发生变化时通过 hub 广播更新。返回值表示是否发生了变化。
func (s *HubbedStore[T]) Store(value T, logger *log_util.NamedLogger) bool {
	return s.st.Store(value, s.hub, logger)
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
