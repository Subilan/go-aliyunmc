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

func TestSnapshotAndTypeMismatch(t *testing.T) {
	key := "test_snapshot_type_mismatch"
	DeleteRecordedHubbedStore(key)
	t.Cleanup(func() { DeleteRecordedHubbedStore(key) })

	store := NewRecordedHubbedStore[string](key)
	store.Store("ok", nil)

	s, ok := Snapshot[string](key)
	if !ok {
		t.Fatalf("expected snapshot to exist")
	}
	if s.Value != "ok" {
		t.Fatalf("unexpected snapshot value: %s", s.Value)
	}

	_, ok = Snapshot[int](key)
	if ok {
		t.Fatalf("expected type mismatch snapshot to fail")
	}
}

func TestStableSnapshotWaitsForFirstUpdate(t *testing.T) {
	key := "test_stable_snapshot_waits"
	DeleteRecordedHubbedStore(key)
	t.Cleanup(func() { DeleteRecordedHubbedStore(key) })

	store := NewRecordedHubbedStore[string](key)

	go func() {
		time.Sleep(30 * time.Millisecond)
		store.Store("ready", nil)
	}()

	snapshot, err := StableSnapshot[string](key, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("expected stable snapshot success, got error: %v", err)
	}
	if snapshot.UpdatedAt.IsZero() {
		t.Fatalf("expected updated snapshot time")
	}
	if snapshot.Value != "ready" {
		t.Fatalf("expected ready, got %s", snapshot.Value)
	}
}

func TestStableSnapshotTimeoutReturnsError(t *testing.T) {
	key := "test_stable_snapshot_timeout"
	DeleteRecordedHubbedStore(key)
	t.Cleanup(func() { DeleteRecordedHubbedStore(key) })

	_ = NewRecordedHubbedStore[string](key)

	_, err := StableSnapshot[string](key, 40*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}
