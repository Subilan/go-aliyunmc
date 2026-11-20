package realtime

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/sse"
)

func receive(t *testing.T, ch <-chan sse.Event, want string) sse.Event {
	t.Helper()
	select {
	case event, ok := <-ch:
		if !ok {
			t.Fatalf("channel closed while waiting for %q", want)
		}
		if event.Event != want {
			t.Fatalf("event = %q, want %q", event.Event, want)
		}
		return event
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %q", want)
		return sse.Event{}
	}
}

func expectNoEvent(t *testing.T, ch <-chan sse.Event) {
	t.Helper()
	select {
	case event, ok := <-ch:
		if ok {
			t.Fatalf("unexpected event %q", event.Event)
		}
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubTargetedAndBroadcast(t *testing.T) {
	h := NewHub()
	user1, unsub1, ok := h.Subscribe(1, 4, nil)
	if !ok {
		t.Fatal("subscribe user1 failed")
	}
	defer unsub1()
	user2, unsub2, ok := h.Subscribe(2, 4, nil)
	if !ok {
		t.Fatal("subscribe user2 failed")
	}
	defer unsub2()

	h.PublishUser(1, sse.Event{Event: "targeted"})
	receive(t, user1, "targeted")
	expectNoEvent(t, user2)

	h.PublishAll(sse.Event{Event: "broadcast"})
	receive(t, user1, "broadcast")
	receive(t, user2, "broadcast")
}

func TestHubInitialEventsAndMultipleTabs(t *testing.T) {
	h := NewHub()
	tab1, unsub1, _ := h.Subscribe(7, 2, nil, sse.Event{Event: "ready"})
	defer unsub1()
	tab2, unsub2, _ := h.Subscribe(7, 2, nil, sse.Event{Event: "ready"})
	defer unsub2()

	receive(t, tab1, "ready")
	receive(t, tab2, "ready")

	h.PublishUser(7, sse.Event{Event: "notification"})
	receive(t, tab1, "notification")
	receive(t, tab2, "notification")
}

func TestHubOverflowRemovesSubscriber(t *testing.T) {
	h := NewHub()
	var overflows atomic.Int32
	ch, _, ok := h.Subscribe(1, 1, func() {
		overflows.Add(1)
	})
	if !ok {
		t.Fatal("subscribe failed")
	}

	// buffer 只有 1，连续两次广播触发溢出并关闭连接。
	h.PublishAll(sse.Event{Event: "first"})
	h.PublishAll(sse.Event{Event: "second"})

	if got := overflows.Load(); got != 1 {
		t.Fatalf("overflow callbacks = %d, want 1", got)
	}
	if h.Len() != 0 {
		t.Fatalf("subscribers = %d, want 0", h.Len())
	}

	events := 0
	for range ch {
		events++
	}
	if events != 1 {
		t.Fatalf("buffered events = %d, want 1", events)
	}
}

func TestHubCloseAndUnsubscribeIdempotent(t *testing.T) {
	h := NewHub()
	ch, unsubscribe, _ := h.Subscribe(1, 2, nil)
	unsubscribe()
	unsubscribe()

	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed")
	}

	ch2, unsubscribe2, _ := h.Subscribe(2, 2, nil)
	h.Close()
	h.Close()
	unsubscribe2()

	if _, ok := <-ch2; ok {
		t.Fatal("channel should be closed after Close")
	}
	if h.Len() != 0 {
		t.Fatalf("subscribers = %d, want 0", h.Len())
	}

	closed, unsubscribe3, ok := h.Subscribe(3, 2, nil)
	if ok {
		t.Fatal("subscribe after Close should fail")
	}
	unsubscribe3()
	if _, open := <-closed; open {
		t.Fatal("channel from closed hub should be closed")
	}
}
