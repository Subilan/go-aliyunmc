package states

import "sync"

type Hub[T any] struct {
	mu       sync.RWMutex
	channels map[chan T]struct{}
}

func NewHub[T any]() *Hub[T] {
	return &Hub[T]{channels: make(map[chan T]struct{})}
}

func (h *Hub[T]) Subscribe() (<-chan T, func()) {
	ch := make(chan T, 1)

	h.mu.Lock()
	h.channels[ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		if _, ok := h.channels[ch]; ok {
			delete(h.channels, ch)
			close(ch)
		}
		h.mu.Unlock()
	}

	return ch, unsubscribe
}

func (h *Hub[T]) Broadcast(value T) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.channels {
		select {
		case ch <- value:
		default:
		}
	}
}
