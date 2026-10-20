// Package pubsub 提供一个泛型的订阅/广播内核，供状态分发与 SSE 事件流共用。
package pubsub

import "sync"

// Hub 将值广播给多个订阅者。每个订阅者拥有独立的缓冲 channel。
//
// 当订阅者的缓冲已满时，行为由订阅时传入的 onOverflow 决定：
//   - onOverflow 为 nil：丢弃本次广播的值，保留订阅者（适合状态类数据，后续值会覆盖当前值）；
//   - onOverflow 不为 nil：移除该订阅者并关闭其 channel，然后在锁外调用 onOverflow
//     （适合日志流，慢消费者应被断开）。
//
// Close 关闭所有订阅 channel，并拒绝后续订阅。Hub 关闭后 Broadcast 不再产生任何效果。
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
//
// 返回值：
//   - 只读 channel：订阅事件流；
//   - unsubscribe：幂等，释放订阅并关闭 channel；
//   - ok：Hub 已关闭时为 false，此时 channel 已关闭，unsubscribe 为空操作。
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
