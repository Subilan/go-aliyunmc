package monitors

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

type fakeIdleCountdownClock struct {
	now time.Time
}

func (c *fakeIdleCountdownClock) Now() time.Time {
	return c.now
}

func (c *fakeIdleCountdownClock) Advance(duration time.Duration) {
	c.now = c.now.Add(duration)
}

func newTestIdleCountdown(duration time.Duration) (*IdleCountdownController, *fakeIdleCountdownClock) {
	clock := &fakeIdleCountdownClock{now: time.Unix(1_700_000_000, 0)}
	controller := newIdleCountdownController(true, duration, 0)
	controller.now = clock.Now
	return controller, clock
}

func TestIdleCountdownPauseResumeResetAndManualSet(t *testing.T) {
	controller, clock := newTestIdleCountdown(20 * time.Minute)

	if !controller.StartIfInactive() {
		t.Fatal("StartIfInactive() = false, want true")
	}
	clock.Advance(10 * time.Minute)

	paused, err := controller.Pause()
	if err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if paused.Status != IdleCountdownPaused || paused.RemainingSeconds != 600 {
		t.Fatalf("Pause() = %+v, want paused with 600 seconds", paused)
	}

	clock.Advance(5 * time.Minute)
	if got := controller.State().RemainingSeconds; got != 600 {
		t.Fatalf("paused remaining = %d, want 600", got)
	}

	resumed, err := controller.Resume()
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if resumed.Status != IdleCountdownRunning || resumed.RemainingSeconds != 600 {
		t.Fatalf("Resume() = %+v, want running with 600 seconds", resumed)
	}

	clock.Advance(15 * time.Second)
	if got := controller.State().RemainingSeconds; got != 585 {
		t.Fatalf("resumed remaining = %d, want 585", got)
	}

	reset, err := controller.Reset()
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if reset.Status != IdleCountdownRunning || reset.RemainingSeconds != 1200 {
		t.Fatalf("Reset() = %+v, want running with configured 1200 seconds", reset)
	}

	manual, err := controller.SetRemaining(90)
	if err != nil {
		t.Fatalf("SetRemaining() error = %v", err)
	}
	if manual.Status != IdleCountdownRunning || manual.RemainingSeconds != 90 {
		t.Fatalf("SetRemaining() = %+v, want running with 90 seconds", manual)
	}
	if manual.ConfiguredSeconds != 1200 {
		t.Fatalf("configured seconds = %d, want 1200", manual.ConfiguredSeconds)
	}
}

func TestIdleCountdownCancelDoesNotOverridePause(t *testing.T) {
	controller, _ := newTestIdleCountdown(time.Minute)
	controller.StartIfInactive()
	if _, err := controller.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	if controller.StartIfInactive() {
		t.Fatal("StartIfInactive() restarted a paused countdown")
	}
	if got := controller.State().Status; got != IdleCountdownPaused {
		t.Fatalf("status = %s, want paused", got)
	}
	if !controller.Cancel() {
		t.Fatal("Cancel() = false, want true")
	}
	if got := controller.State(); got.Status != IdleCountdownInactive || got.RemainingSeconds != 0 {
		t.Fatalf("state after cancel = %+v, want inactive", got)
	}
}

func TestIdleCountdownCancelUnlessPausedKeepsPause(t *testing.T) {
	controller, clock := newTestIdleCountdown(20 * time.Minute)
	controller.StartIfInactive()
	clock.Advance(5 * time.Minute)

	if _, err := controller.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	// 玩家回到服务器时，暂停不应被取消。
	changed, status := controller.CancelUnlessPaused()
	if changed {
		t.Fatal("CancelUnlessPaused() = true, want false for paused countdown")
	}
	if status != IdleCountdownPaused {
		t.Fatalf("status after CancelUnlessPaused = %s, want paused", status)
	}
	if got := controller.State(); got.Status != IdleCountdownPaused || got.RemainingSeconds != 900 {
		t.Fatalf("state after CancelUnlessPaused = %+v, want paused with 900 seconds", got)
	}

	// 玩家离开后监控器会调用 StartIfInactive，暂停不能被覆盖。
	clock.Advance(10 * time.Minute)
	if controller.StartIfInactive() {
		t.Fatal("StartIfInactive() restarted a paused countdown")
	}

	resumed, err := controller.Resume()
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if resumed.Status != IdleCountdownRunning || resumed.RemainingSeconds != 900 {
		t.Fatalf("Resume() = %+v, want running with frozen 900 seconds", resumed)
	}
}

func TestIdleCountdownPersistenceRestoresPausedAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idle_countdown.json")
	clock := &fakeIdleCountdownClock{now: time.Unix(1_700_000_000, 0)}

	controller := newIdleCountdownController(true, 20*time.Minute, 0)
	controller.now = clock.Now
	controller.AttachPersistence(path, nil)
	controller.StartIfInactive()
	clock.Advance(5 * time.Minute)
	if _, err := controller.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	// 模拟进程重启：新控制器从同一文件恢复。
	restored := newIdleCountdownController(true, 20*time.Minute, 0)
	restored.now = clock.Now
	restored.AttachPersistence(path, nil)

	state := restored.State()
	if state.Status != IdleCountdownPaused || state.RemainingSeconds != 900 {
		t.Fatalf("restored state = %+v, want paused with 900 seconds", state)
	}
	if restored.StartIfInactive() {
		t.Fatal("StartIfInactive() overwrote restored paused state")
	}
}

func TestIdleCountdownPersistenceRestoresRunning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idle_countdown.json")
	clock := &fakeIdleCountdownClock{now: time.Unix(1_700_000_000, 0)}

	controller := newIdleCountdownController(true, 20*time.Minute, 0)
	controller.now = clock.Now
	controller.AttachPersistence(path, nil)
	controller.StartIfInactive()
	clock.Advance(5 * time.Minute)

	restored := newIdleCountdownController(true, 20*time.Minute, 0)
	restored.now = clock.Now
	restored.AttachPersistence(path, nil)

	state := restored.State()
	if state.Status != IdleCountdownRunning || state.RemainingSeconds != 900 {
		t.Fatalf("restored state = %+v, want running with 900 seconds", state)
	}
}

func TestIdleCountdownPersistenceDropsExpiredRunningAndClearedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idle_countdown.json")
	clock := &fakeIdleCountdownClock{now: time.Unix(1_700_000_000, 0)}

	controller := newIdleCountdownController(true, 20*time.Minute, 0)
	controller.now = clock.Now
	controller.AttachPersistence(path, nil)
	controller.StartIfInactive()

	// 停机时间超过截止时间：不恢复运行中的倒计时，交给快照逻辑重新判断。
	clock.Advance(30 * time.Minute)
	expired := newIdleCountdownController(true, 20*time.Minute, 0)
	expired.now = clock.Now
	expired.AttachPersistence(path, nil)
	if got := expired.State().Status; got != IdleCountdownInactive {
		t.Fatalf("status after expired restore = %s, want inactive", got)
	}

	// 显式取消会清空持久化文件。
	controller.Cancel()
	cleared := newIdleCountdownController(true, 20*time.Minute, 0)
	cleared.now = clock.Now
	cleared.AttachPersistence(path, nil)
	if got := cleared.State(); got.Status != IdleCountdownInactive {
		t.Fatalf("status after cancel restore = %s, want inactive", got.Status)
	}
}

func TestIdleCountdownExpiresOnce(t *testing.T) {
	controller, clock := newTestIdleCountdown(2 * time.Second)
	controller.StartIfInactive()
	clock.Advance(2 * time.Second)

	controller.mu.Lock()
	controller.expireLocked(clock.Now())
	controller.mu.Unlock()

	state := controller.State()
	if state.Status != IdleCountdownFinalizing || state.RemainingSeconds != 0 {
		t.Fatalf("state after expiry = %+v, want finalizing with 0 seconds", state)
	}
	select {
	case <-controller.Expired():
	default:
		t.Fatal("Expired() did not receive an event")
	}

	if _, err := controller.SetRemaining(10); !errors.Is(err, ErrIdleCountdownBusy) {
		t.Fatalf("SetRemaining() error = %v, want ErrIdleCountdownBusy", err)
	}
	if _, err := controller.Pause(); !errors.Is(err, ErrIdleCountdownBusy) {
		t.Fatalf("Pause() error = %v, want ErrIdleCountdownBusy", err)
	}
	if _, err := controller.Resume(); !errors.Is(err, ErrIdleCountdownBusy) {
		t.Fatalf("Resume() error = %v, want ErrIdleCountdownBusy", err)
	}
	if _, err := controller.Reset(); !errors.Is(err, ErrIdleCountdownBusy) {
		t.Fatalf("Reset() error = %v, want ErrIdleCountdownBusy", err)
	}
	if controller.Cancel() {
		t.Fatal("Cancel() = true while finalizing")
	}
	if controller.StartIfInactive() {
		t.Fatal("StartIfInactive() = true while finalizing")
	}
	controller.Complete()
	if got := controller.State().Status; got != IdleCountdownInactive {
		t.Fatalf("status after Complete() = %s, want inactive", got)
	}
	if !controller.StartIfInactive() {
		t.Fatal("StartIfInactive() = false after Complete()")
	}
}

func TestIdleCountdownBackgroundExpiry(t *testing.T) {
	controller := newIdleCountdownController(true, 30*time.Millisecond, 0)
	controller.tickInterval = 5 * time.Millisecond
	controller.StartIfInactive()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go controller.run(ctx)

	select {
	case <-controller.Expired():
	case <-time.After(time.Second):
		t.Fatal("background countdown did not expire")
	}
	if got := controller.State().Status; got != IdleCountdownFinalizing {
		t.Fatalf("status after background expiry = %s, want finalizing", got)
	}
}

func TestIdleCountdownValidationAndDisabledState(t *testing.T) {
	controller, _ := newTestIdleCountdown(time.Minute)
	if _, err := controller.SetRemaining(0); !errors.Is(err, ErrIdleCountdownDuration) {
		t.Fatalf("SetRemaining(0) error = %v, want ErrIdleCountdownDuration", err)
	}
	if _, err := controller.SetRemaining(maxIdleCountdownSeconds + 1); !errors.Is(err, ErrIdleCountdownDuration) {
		t.Fatalf("SetRemaining(too large) error = %v, want ErrIdleCountdownDuration", err)
	}

	disabled := newIdleCountdownController(false, time.Minute, 0)
	if disabled.StartIfInactive() {
		t.Fatal("disabled countdown started")
	}
	if _, err := disabled.Reset(); !errors.Is(err, ErrIdleCountdownDisabled) {
		t.Fatalf("Reset() error = %v, want ErrIdleCountdownDisabled", err)
	}
}

func TestIdleCountdownIdleSeconds(t *testing.T) {
	controller, clock := newTestIdleCountdown(20 * time.Minute)

	if got := controller.State().IdleSeconds; got != 0 {
		t.Fatalf("idle seconds before start = %d, want 0", got)
	}

	controller.StartIfInactive()
	clock.Advance(5 * time.Minute)
	if got := controller.State().IdleSeconds; got != 300 {
		t.Fatalf("idle seconds after 5 minutes = %d, want 300", got)
	}

	// 暂停期间玩家仍然没有上线，空闲时长继续按实际时间累计。
	if _, err := controller.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	clock.Advance(2 * time.Minute)
	if got := controller.State().IdleSeconds; got != 420 {
		t.Fatalf("idle seconds while paused = %d, want 420", got)
	}

	if _, err := controller.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	clock.Advance(time.Minute)
	if got := controller.State().IdleSeconds; got != 480 {
		t.Fatalf("idle seconds after resume = %d, want 480", got)
	}

	// 手动重置剩余时间不应把空服周期重新计时。
	if _, err := controller.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if got := controller.State().IdleSeconds; got != 480 {
		t.Fatalf("idle seconds after reset = %d, want 480", got)
	}

	controller.Cancel()
	if got := controller.State().IdleSeconds; got != 0 {
		t.Fatalf("idle seconds after cancel = %d, want 0", got)
	}
}

func TestIdleCountdownPersistenceRestoresIdleSeconds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idle_countdown.json")
	clock := &fakeIdleCountdownClock{now: time.Unix(1_700_000_000, 0)}

	controller := newIdleCountdownController(true, 20*time.Minute, 0)
	controller.now = clock.Now
	controller.AttachPersistence(path, nil)
	controller.StartIfInactive()
	clock.Advance(5 * time.Minute)

	restored := newIdleCountdownController(true, 20*time.Minute, 0)
	restored.now = clock.Now
	restored.AttachPersistence(path, nil)

	if got := restored.State().IdleSeconds; got != 300 {
		t.Fatalf("restored idle seconds = %d, want 300", got)
	}
}

func remindIdleCountdown(controller *IdleCountdownController, now time.Time) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.remindLocked(now)
}

func expectNoIdleReminder(t *testing.T, controller *IdleCountdownController) {
	t.Helper()
	select {
	case event := <-controller.Reminded():
		t.Fatalf("不应收到提醒: %+v", event)
	default:
	}
}

func TestIdleCountdownReminder(t *testing.T) {
	clock := &fakeIdleCountdownClock{now: time.Unix(1_700_000_000, 0)}
	controller := newIdleCountdownController(true, 20*time.Minute, 10*time.Minute)
	controller.now = clock.Now
	controller.StartIfInactive()

	remindIdleCountdown(controller, clock.Now())
	expectNoIdleReminder(t, controller)

	clock.Advance(10 * time.Minute)
	remindIdleCountdown(controller, clock.Now())

	select {
	case event := <-controller.Reminded():
		if event.IdleSeconds != 600 || event.RemainingSeconds != 600 {
			t.Fatalf("提醒事件不正确: %+v", event)
		}
	default:
		t.Fatal("到达阈值后应收到提醒")
	}

	// 同一轮倒计时只提醒一次。
	remindIdleCountdown(controller, clock.Now())
	expectNoIdleReminder(t, controller)

	// 暂停恢复不重复提醒。
	if _, err := controller.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	clock.Advance(time.Minute)
	if _, err := controller.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	remindIdleCountdown(controller, clock.Now())
	expectNoIdleReminder(t, controller)

	// 重置到阈值之上后重新允许提醒。
	if _, err := controller.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	clock.Advance(10 * time.Minute)
	remindIdleCountdown(controller, clock.Now())
	select {
	case <-controller.Reminded():
	default:
		t.Fatal("重置后应重新提醒")
	}
}

func TestIdleCountdownReminderRestartsAfterCancel(t *testing.T) {
	clock := &fakeIdleCountdownClock{now: time.Unix(1_700_000_000, 0)}
	controller := newIdleCountdownController(true, 20*time.Minute, 10*time.Minute)
	controller.now = clock.Now

	controller.StartIfInactive()
	clock.Advance(10 * time.Minute)
	remindIdleCountdown(controller, clock.Now())
	select {
	case <-controller.Reminded():
	default:
		t.Fatal("第一轮倒计时应收到提醒")
	}

	controller.Cancel()
	if !controller.StartIfInactive() {
		t.Fatal("取消后应能开始新一轮倒计时")
	}
	clock.Advance(10 * time.Minute)
	remindIdleCountdown(controller, clock.Now())
	select {
	case <-controller.Reminded():
	default:
		t.Fatal("新一轮倒计时应再次提醒")
	}
}

func TestIdleCountdownReminderDisabled(t *testing.T) {
	clock := &fakeIdleCountdownClock{now: time.Unix(1_700_000_000, 0)}
	controller := newIdleCountdownController(true, 20*time.Minute, 0)
	controller.now = clock.Now
	controller.StartIfInactive()

	clock.Advance(15 * time.Minute)
	remindIdleCountdown(controller, clock.Now())
	expectNoIdleReminder(t, controller)
}

func TestIdleCountdownBackgroundReminder(t *testing.T) {
	controller := newIdleCountdownController(true, 60*time.Millisecond, 20*time.Millisecond)
	controller.tickInterval = 5 * time.Millisecond
	controller.StartIfInactive()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go controller.run(ctx)

	select {
	case event := <-controller.Reminded():
		if event.RemainingSeconds <= 0 {
			t.Fatalf("提醒事件剩余时间不正确: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("后台倒计时没有发出提醒")
	}
}
