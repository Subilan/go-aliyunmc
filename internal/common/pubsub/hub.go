// Package pubsub 提供一个泛型的订阅/广播内核，供状态分发与 SSE 事件流共用。
package pubsub

import "sync"

// Hub 将值广播给多个订阅者，每个订阅者拥有独立的缓冲 channel。
type Hub[T any] struct {
	mu     sync.Mutex
	subs   map[*subscriber[T]]struct{}
	closed bool
}

type subscriber[T any] struct {
	ch         chan T
	onOverflow func()
}

// NewHub 创建一个未关闭的 Hub。
func NewHub[T any]() *Hub[T] {
	return &Hub[T]{subs: make(map[*subscriber[T]]struct{})}
}

// Subscribe 创建一个带缓冲的订阅。buffer 小于 0 时按 0 处理。
// Hub 已关闭时 ok 为 false，返回的 channel 已经关闭。
func (h *Hub[T]) Subscribe(buffer int, onOverflow func()) (<-chan T, func(), bool) {
	if buffer < 0 {
		buffer = 0
	}

	ch := make(chan T, buffer)
	s := &subscriber[T]{ch: ch, onOverflow: onOverflow}

	h.mu.Lock()
	if h.closed {
		close(ch)
		h.mu.Unlock()
		return ch, func() {}, false
	}
	h.subs[s] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			if _, ok := h.subs[s]; ok {
				delete(h.subs, s)
				close(ch)
			}
			h.mu.Unlock()
		})
	}

	return ch, unsubscribe, true
}

// Broadcast 非阻塞地向所有订阅者投递 value。
// 缓冲区已满且配置了 onOverflow 的订阅者会被移除。
func (h *Hub[T]) Broadcast(value T) {
	var overflowed []func()

	h.mu.Lock()
	for s := range h.subs {
		select {
		case s.ch <- value:
		default:
			if s.onOverflow == nil {
				continue
			}
			delete(h.subs, s)
			close(s.ch)
			overflowed = append(overflowed, s.onOverflow)
		}
	}
	h.mu.Unlock()

	// 回调放在锁外执行，允许回调再次访问 Hub 或做取消操作。
	for _, fn := range overflowed {
		fn()
	}
}

// Close 关闭 Hub 及其所有订阅 channel。可重复调用。
func (h *Hub[T]) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}
	h.closed = true

	for s := range h.subs {
		delete(h.subs, s)
		close(s.ch)
	}
}

// Len 返回当前订阅者数量，主要用于观测与测试。
func (h *Hub[T]) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}
