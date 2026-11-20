package monitors

import (
	"slices"
	"testing"
	"time"
)

func testPlayerSet(names ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	return set
}

func TestPlayerSet(t *testing.T) {
	got := playerSet(" Alice, Bob ,, 张三 ")
	for _, name := range []string{"Alice", "Bob", "张三"} {
		if _, ok := got[name]; !ok {
			t.Errorf("playerSet 缺少玩家 %q: %v", name, got)
		}
	}
	if len(got) != 3 {
		t.Errorf("playerSet 数量不正确: %v", got)
	}
	if len(playerSet("")) != 0 {
		t.Errorf("空采样应返回空集合: %v", playerSet(""))
	}
}

func TestPlayerPresenceTracker(t *testing.T) {
	const gapThreshold = 15 * time.Second
	base := time.Unix(1_700_000_000, 0)

	newTracker := func() *playerPresenceTracker {
		return &playerPresenceTracker{gapThreshold: gapThreshold}
	}

	t.Run("首次采样只建立基线", func(t *testing.T) {
		tracker := newTracker()
		tracker.Observe(base, testPlayerSet("a"))

		joined, left := tracker.Flush()
		if len(joined) != 0 || len(left) != 0 {
			t.Fatalf("首次采样不应播报: joined=%v left=%v", joined, left)
		}
	})

	t.Run("合并窗口内的进与出", func(t *testing.T) {
		tracker := newTracker()
		tracker.Observe(base, testPlayerSet("a"))
		tracker.Flush()

		tracker.Observe(base.Add(5*time.Second), testPlayerSet("a", "b"))
		tracker.Observe(base.Add(10*time.Second), testPlayerSet("a", "b", "c"))

		joined, left := tracker.Flush()
		if !slices.Equal(joined, []string{"b", "c"}) {
			t.Fatalf("joined = %v, want [b c]", joined)
		}
		if len(left) != 0 {
			t.Fatalf("left = %v, want empty", left)
		}

		tracker.Observe(base.Add(15*time.Second), testPlayerSet("c"))
		joined, left = tracker.Flush()
		if len(joined) != 0 {
			t.Fatalf("joined = %v, want empty", joined)
		}
		if !slices.Equal(left, []string{"a", "b"}) {
			t.Fatalf("left = %v, want [a b]", left)
		}
	})

	t.Run("窗口内加入又离开不播报", func(t *testing.T) {
		tracker := newTracker()
		tracker.Observe(base, testPlayerSet("a"))
		tracker.Flush()

		tracker.Observe(base.Add(5*time.Second), testPlayerSet("a", "b"))
		tracker.Observe(base.Add(10*time.Second), testPlayerSet("a"))

		joined, left := tracker.Flush()
		if len(joined) != 0 || len(left) != 0 {
			t.Fatalf("净变化为零时不应播报: joined=%v left=%v", joined, left)
		}
	})

	t.Run("忽略过期的采样", func(t *testing.T) {
		tracker := newTracker()
		tracker.Observe(base, testPlayerSet("a"))
		tracker.Flush()

		tracker.Observe(base.Add(10*time.Second), testPlayerSet("a", "b"))
		tracker.Observe(base.Add(5*time.Second), testPlayerSet("a"))

		joined, left := tracker.Flush()
		if !slices.Equal(joined, []string{"b"}) {
			t.Fatalf("joined = %v, want [b]", joined)
		}
		if len(left) != 0 {
			t.Fatalf("left = %v, want empty", left)
		}
	})

	t.Run("采样断档后重建基线", func(t *testing.T) {
		tracker := newTracker()
		tracker.Observe(base, testPlayerSet("a"))
		tracker.Flush()

		tracker.Observe(base.Add(gapThreshold+time.Second), testPlayerSet("a", "b"))

		joined, left := tracker.Flush()
		if len(joined) != 0 || len(left) != 0 {
			t.Fatalf("断档后的首个采样不应播报: joined=%v left=%v", joined, left)
		}
	})

	t.Run("Rebaseline 丢弃未播报变化", func(t *testing.T) {
		tracker := newTracker()
		tracker.Observe(base, testPlayerSet("a"))
		tracker.Flush()

		tracker.Observe(base.Add(5*time.Second), testPlayerSet("a", "b"))
		tracker.Rebaseline()

		joined, left := tracker.Flush()
		if len(joined) != 0 || len(left) != 0 {
			t.Fatalf("Rebaseline 后不应播报: joined=%v left=%v", joined, left)
		}

		tracker.Observe(base.Add(10*time.Second), testPlayerSet("a"))
		joined, left = tracker.Flush()
		if len(joined) != 0 || len(left) != 0 {
			t.Fatalf("重建基线后的首个采样不应播报: joined=%v left=%v", joined, left)
		}
	})

	t.Run("玩家全部离线", func(t *testing.T) {
		tracker := newTracker()
		tracker.Observe(base, testPlayerSet("a", "b"))
		tracker.Flush()

		tracker.Observe(base.Add(5*time.Second), testPlayerSet())

		joined, left := tracker.Flush()
		if len(joined) != 0 {
			t.Fatalf("joined = %v, want empty", joined)
		}
		if !slices.Equal(left, []string{"a", "b"}) {
			t.Fatalf("left = %v, want [a b]", left)
		}
	})
}
