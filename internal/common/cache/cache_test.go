package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetOrLoadCaches(t *testing.T) {
	store := New[int](time.Minute, 0)

	var loads atomic.Int64
	load := func() (int, error) {
		return int(loads.Add(1)), nil
	}

	first, err := store.GetOrLoad("key", load)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	second, err := store.GetOrLoad("key", load)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	if first != 1 || second != 1 {
		t.Fatalf("expected cached value 1, got %d and %d", first, second)
	}
	if loads.Load() != 1 {
		t.Fatalf("expected a single load, got %d", loads.Load())
	}
}

func TestGetOrLoadDoesNotCacheError(t *testing.T) {
	store := New[int](time.Minute, 0)

	var loads atomic.Int64
	load := func() (int, error) {
		if loads.Add(1) == 1 {
			return 0, errors.New("boom")
		}
		return 7, nil
	}

	if _, err := store.GetOrLoad("key", load); err == nil {
		t.Fatal("expected an error on the first load")
	}
	value, err := store.GetOrLoad("key", load)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if value != 7 {
		t.Fatalf("expected 7, got %d", value)
	}
	if loads.Load() != 2 {
		t.Fatalf("expected a retry after the error, got %d loads", loads.Load())
	}
}

func TestGetOrLoadSuppressesConcurrentLoads(t *testing.T) {
	store := New[string](time.Minute, 0)

	var loads atomic.Int64
	load := func() (string, error) {
		loads.Add(1)
		time.Sleep(50 * time.Millisecond)
		return "value", nil
	}

	var wg sync.WaitGroup
	results := make([]string, 32)
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			value, err := store.GetOrLoad("key", load)
			if err != nil {
				t.Errorf("load failed: %v", err)
				return
			}
			results[index] = value
		}(i)
	}
	wg.Wait()

	if loads.Load() != 1 {
		t.Fatalf("expected a single shared load, got %d", loads.Load())
	}
	for i, value := range results {
		if value != "value" {
			t.Fatalf("unexpected value at %d: %q", i, value)
		}
	}
}

func TestGetOrLoadExpires(t *testing.T) {
	store := New[int](20*time.Millisecond, 0)

	var loads atomic.Int64
	load := func() (int, error) {
		return int(loads.Add(1)), nil
	}

	if _, err := store.GetOrLoad("key", load); err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	time.Sleep(40 * time.Millisecond)

	value, err := store.GetOrLoad("key", load)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if value != 2 {
		t.Fatalf("expected a reload after expiry, got %d", value)
	}
}

func TestGetOrLoadRespectsCapacity(t *testing.T) {
	store := New[int](time.Minute, 2)

	for _, key := range []string{"a", "b", "c"} {
		if _, err := store.GetOrLoad(key, func() (int, error) { return len(key), nil }); err != nil {
			t.Fatalf("load %s failed: %v", key, err)
		}
	}

	var loads atomic.Int64
	if _, err := store.GetOrLoad("a", func() (int, error) {
		loads.Add(1)
		return 0, nil
	}); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loads.Load() != 1 {
		t.Fatalf("expected the oldest entry to be evicted, got %d loads", loads.Load())
	}
}
