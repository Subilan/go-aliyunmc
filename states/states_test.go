package states

import (
	"errors"
	"testing"
	"time"
)

func mustReceiveState[T any](t *testing.T, ch <-chan State[T]) State[T] {
	t.Helper()
	select {
	case v, ok := <-ch:
		if !ok {
			t.Fatalf("channel closed unexpectedly")
		}
		return v
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("timeout waiting for state")
		return State[T]{}
	}
}

func TestStateStore_BroadcastWhenErrorClearedWithoutValueChange(t *testing.T) {
	st := &StateStore[string]{}
	hub := NewHub[State[string]]()
	ch, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	st.Store("running", hub, nil)
	first := mustReceiveState(t, ch)
	if first.Value != "running" || first.Error != nil {
		t.Fatalf("unexpected first snapshot: %+v", first)
	}

	errTarget := errors.New("temporary fetch failed")
	st.StoreError(errTarget, hub, nil)
	withErr := mustReceiveState(t, ch)
	if withErr.Error == nil {
		t.Fatalf("expected error snapshot, got: %+v", withErr)
	}

	st.Store("running", hub, nil)
	recovered := mustReceiveState(t, ch)
	if recovered.Error != nil {
		t.Fatalf("expected recovered snapshot without error, got: %+v", recovered)
	}
	if recovered.Value != "running" {
		t.Fatalf("expected value to remain running, got: %s", recovered.Value)
	}
}

func TestHubbedStore_WaitSnapshot(t *testing.T) {
	store := NewHubbedStore[string]()

	done := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Millisecond)
		store.Store("ready", nil)
		close(done)
	}()

	snap, err := store.WaitSnapshot(500 * time.Millisecond)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if snap.Value != "ready" {
		t.Fatalf("expected ready, got: %s", snap.Value)
	}
	if snap.UpdatedAt.IsZero() {
		t.Fatalf("expected updated snapshot time")
	}
	<-done
}

func TestHubbedStore_WaitSnapshotTimeout(t *testing.T) {
	store := NewHubbedStore[string]()
	_, err := store.WaitSnapshot(40 * time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}
