package pubsub

import (
	"sync/atomic"
	"testing"
	"time"
)

func receive[T any](t *testing.T, ch <-chan T) (T, bool) {
	t.Helper()
	select {
	case v, ok := <-ch:
		return v, ok
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("timeout waiting for value")
		var zero T
		return zero, false
	}
}

func TestHubBroadcastToAllSubscribers(t *testing.T) {
	h := NewHub[int]()

	ch1, unsub1, ok1 := h.Subscribe(1, nil)
	ch2, unsub2, ok2 := h.Subscribe(1, nil)
	defer unsub1()
	defer unsub2()

	if !ok1 || !ok2 {
		t.Fatalf("expected subscribe to succeed")
	}

	h.Broadcast(42)

	if v, ok := receive(t, ch1); !ok || v != 42 {
		t.Fatalf("subscriber 1: got %v, %v", v, ok)
	}
	if v, ok := receive(t, ch2); !ok || v != 42 {
		t.Fatalf("subscriber 2: got %v, %v", v, ok)
	}
}

func TestHubDropsValueWhenBufferFullWithoutOverflowCallback(t *testing.T) {
	h := NewHub[int]()
	ch, unsubscribe, _ := h.Subscribe(1, nil)
	defer unsubscribe()

	h.Broadcast(1)
	h.Broadcast(2) // dropped: buffer full, subscriber retained

	if v, ok := receive(t, ch); !ok || v != 1 {
		t.Fatalf("expected first value, got %v, %v", v, ok)
	}
	if h.Len() != 1 {
		t.Fatalf("subscriber should be retained, got %d", h.Len())
	}

	h.Broadcast(3)
	if v, ok := receive(t, ch); !ok || v != 3 {
		t.Fatalf("expected third value after drain, got %v, %v", v, ok)
	}
}

func TestHubRemovesSlowSubscriberWhenOverflowCallbackConfigured(t *testing.T) {
	h := NewHub[int]()

	var called atomic.Int32
	ch, _, _ := h.Subscribe(1, func() { called.Add(1) })

	h.Broadcast(1)
	h.Broadcast(2) // buffer full -> subscriber removed and closed

	if v, ok := receive(t, ch); !ok || v != 1 {
		t.Fatalf("expected buffered value, got %v, %v", v, ok)
	}
	if _, ok := receive(t, ch); ok {
		t.Fatalf("expected channel to be closed")
	}
	if called.Load() != 1 {
		t.Fatalf("expected overflow callback once, got %d", called.Load())
	}
	if h.Len() != 0 {
		t.Fatalf("expected no subscribers, got %d", h.Len())
	}
}

func TestHubCloseClosesSubscribersAndRejectsNewOnes(t *testing.T) {
	h := NewHub[int]()
	ch, unsubscribe, _ := h.Subscribe(2, nil)

	h.Broadcast(1)
	h.Close()
	h.Close() // idempotent

	if v, ok := receive(t, ch); !ok || v != 1 {
		t.Fatalf("expected buffered value before close, got %v, %v", v, ok)
	}
	if _, ok := receive(t, ch); ok {
		t.Fatalf("expected channel to be closed")
	}

	unsubscribe() // must not panic after Close

	ch2, unsubscribe2, ok := h.Subscribe(1, nil)
	if ok {
		t.Fatalf("expected subscribe on closed hub to fail")
	}
	if _, open := <-ch2; open {
		t.Fatalf("expected returned channel to be closed")
	}
	unsubscribe2()

	h.Broadcast(2) // no-op
}

func TestHubCloseWaitsForNoOne(t *testing.T) {
	h := NewHub[int]()
	_, unsubscribe, _ := h.Subscribe(0, nil)
	defer unsubscribe()

	// Close 只关闭 channel，不会阻塞在尚未被消费的订阅者上。
	done := make(chan struct{})
	go func() {
		h.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("Close blocked on a subscriber")
	}
}
