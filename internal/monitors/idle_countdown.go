package monitors

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"
)

// IdleCountdownStatus 描述关服倒计时当前所处的阶段。
type IdleCountdownStatus string

const (
	IdleCountdownInactive   IdleCountdownStatus = "inactive"
	IdleCountdownRunning    IdleCountdownStatus = "running"
	IdleCountdownPaused     IdleCountdownStatus = "paused"
	IdleCountdownTriggering IdleCountdownStatus = "triggering"

	maxIdleCountdownSeconds int64 = 7 * 24 * 60 * 60
)

var (
	ErrIdleCountdownDisabled = errors.New("关服倒计时未启用")
	ErrIdleCountdownNotRun   = errors.New("当前没有运行中的倒计时")
	ErrIdleCountdownNotPause = errors.New("当前没有已暂停的倒计时")
	ErrIdleCountdownBusy     = errors.New("倒计时正在执行关服流程")
	ErrIdleCountdownDuration = errors.New("倒计时时长必须在 1 秒到 7 天之间")
)

// IdleCountdownState 是提供给前端展示的关服倒计时状态。
type IdleCountdownState struct {
	Enabled           bool                `json:"enabled"`
	Status            IdleCountdownStatus `json:"status"`
	RemainingSeconds  int64               `json:"remaining_seconds"`
	ConfiguredSeconds int64               `json:"configured_seconds"`
}

type idleCountdownClock func() time.Time

// IdleCountdownController 管理关服倒计时。它把剩余时间与动态截止时间分开保存，
// 从而避免每秒修改全局状态，并让暂停/恢复保持精确。
type IdleCountdownController struct {
	mu           sync.RWMutex
	enabled      bool
	configured   time.Duration
	status       IdleCountdownStatus
	remaining    time.Duration
	deadline     time.Time
	now          idleCountdownClock
	expired      chan struct{}
	tickInterval time.Duration
}

func newIdleCountdownController(enabled bool, configured time.Duration) *IdleCountdownController {
	return &IdleCountdownController{
		enabled:      enabled,
		configured:   configured,
		status:       IdleCountdownInactive,
		now:          time.Now,
		expired:      make(chan struct{}, 1),
		tickInterval: time.Second,
	}
}

// State 返回倒计时的当前快照。运行中的剩余秒数按截止时间即时计算。
func (c *IdleCountdownController) State() IdleCountdownState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stateLocked(c.now())
}

// Expired 在倒计时归零时收到一次通知。
func (c *IdleCountdownController) Expired() <-chan struct{} {
	return c.expired
}

// StartIfInactive 由空服监控触发。运行中、暂停中或正在关服时不会覆盖现有状态。
func (c *IdleCountdownController) StartIfInactive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled || c.status != IdleCountdownInactive {
		return false
	}
	c.startLocked(c.configured, c.now())
	return true
}

// Cancel 停止倒计时并回到未启动状态。关服流程已经开始时不能取消。
func (c *IdleCountdownController) Cancel() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == IdleCountdownTriggering {
		return false
	}
	changed := c.status != IdleCountdownInactive
	c.status = IdleCountdownInactive
	c.remaining = 0
	c.deadline = time.Time{}
	return changed
}

// Pause 冻结当前剩余时间。
func (c *IdleCountdownController) Pause() (IdleCountdownState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if c.status == IdleCountdownTriggering || (c.status == IdleCountdownRunning && !c.deadline.After(now)) {
		c.expireLocked(now)
		return c.stateLocked(now), ErrIdleCountdownBusy
	}
	if c.status != IdleCountdownRunning {
		return c.stateLocked(now), ErrIdleCountdownNotRun
	}

	c.remaining = c.deadline.Sub(now)
	c.deadline = time.Time{}
	c.status = IdleCountdownPaused
	return c.stateLocked(now), nil
}

// Resume 从暂停时的剩余时间继续倒计时。
func (c *IdleCountdownController) Resume() (IdleCountdownState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if c.status == IdleCountdownTriggering {
		return c.stateLocked(now), ErrIdleCountdownBusy
	}
	if c.status != IdleCountdownPaused {
		return c.stateLocked(now), ErrIdleCountdownNotPause
	}
	c.startLocked(c.remaining, now)
	return c.stateLocked(now), nil
}

// Reset 使用配置的默认时长重新开始倒计时。
func (c *IdleCountdownController) Reset() (IdleCountdownState, error) {
	return c.startAt(c.configured, c.now())
}

// SetRemaining 使用手动指定的时长重新开始倒计时。
func (c *IdleCountdownController) SetRemaining(seconds int64) (IdleCountdownState, error) {
	if seconds < 1 || seconds > maxIdleCountdownSeconds {
		return c.State(), ErrIdleCountdownDuration
	}
	return c.startAt(time.Duration(seconds)*time.Second, c.now())
}

// Complete 在关服流程结束后清除倒计时状态，等待监控器根据最新服务器状态重新判断。
func (c *IdleCountdownController) Complete() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.status != IdleCountdownTriggering {
		return
	}
	c.status = IdleCountdownInactive
	c.remaining = 0
	c.deadline = time.Time{}
}

func (c *IdleCountdownController) startAt(duration time.Duration, now time.Time) (IdleCountdownState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		return c.stateLocked(now), ErrIdleCountdownDisabled
	}
	if c.status == IdleCountdownTriggering {
		return c.stateLocked(now), ErrIdleCountdownBusy
	}
	c.startLocked(duration, now)
	return c.stateLocked(now), nil
}

func (c *IdleCountdownController) startLocked(duration time.Duration, now time.Time) {
	c.status = IdleCountdownRunning
	c.remaining = duration
	c.deadline = now.Add(duration)
}

func (c *IdleCountdownController) stateLocked(now time.Time) IdleCountdownState {
	remaining := c.remaining
	if c.status == IdleCountdownRunning {
		remaining = c.deadline.Sub(now)
		if remaining < 0 {
			remaining = 0
		}
	}
	remainingSeconds := int64(math.Ceil(remaining.Seconds()))
	if c.status == IdleCountdownInactive {
		remainingSeconds = 0
	}
	return IdleCountdownState{
		Enabled:           c.enabled,
		Status:            c.status,
		RemainingSeconds:  remainingSeconds,
		ConfiguredSeconds: int64(c.configured / time.Second),
	}
}

func (c *IdleCountdownController) expireLocked(now time.Time) {
	if c.status != IdleCountdownRunning || c.deadline.After(now) {
		return
	}
	c.status = IdleCountdownTriggering
	c.remaining = 0
	c.deadline = time.Time{}
	select {
	case c.expired <- struct{}{}:
	default:
	}
}

func (c *IdleCountdownController) run(ctx context.Context) {
	ticker := time.NewTicker(c.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.Cancel()
			return
		case now := <-ticker.C:
			c.mu.Lock()
			c.expireLocked(now)
			c.mu.Unlock()
		}
	}
}
