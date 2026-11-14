package monitors

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/log_util"
)

// IdleCountdownStatus 描述关服倒计时当前所处的阶段。
type IdleCountdownStatus string

const (
	IdleCountdownInactive   IdleCountdownStatus = "inactive"
	IdleCountdownRunning    IdleCountdownStatus = "running"
	IdleCountdownPaused     IdleCountdownStatus = "paused"
	IdleCountdownFinalizing IdleCountdownStatus = "finalizing"

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

// idleCountdownPersistedState 是写入 state/idle_countdown.json 的内容。
// 运行中的倒计时保存绝对截止时间，暂停中的倒计时保存冻结的剩余时长。
type idleCountdownPersistedState struct {
	Status      IdleCountdownStatus `json:"status"`
	RemainingMs int64               `json:"remaining_ms,omitempty"`
	DeadlineMs  int64               `json:"deadline_ms,omitempty"`
}

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
	// statePath 为空表示不持久化；非空时每次状态迁移都会写入该文件。
	statePath string
	logger    *log_util.NamedLogger
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

// AttachPersistence 启用状态持久化，并尝试恢复上次保存的状态。
// path 为空时不启用持久化；文件不存在或内容无效时忽略，保持当前状态。
func (c *IdleCountdownController) AttachPersistence(path string, logger *log_util.NamedLogger) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statePath = path
	c.logger = logger
	c.restoreLocked()
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
	return c.cancelLocked(false)
}

// CancelUnlessPaused 与 Cancel 相同，但会保留管理员手动暂停的状态。
// 空服倒计时因玩家回到服务器而结束时使用：运行中的倒计时应取消，
// 而暂停是管理员显式表达的意图，不应被玩家进出静默重置。
// 返回值是是否发生了取消，以及调用后的状态，便于调用方区分“取消”与“保留暂停”。
func (c *IdleCountdownController) CancelUnlessPaused() (bool, IdleCountdownStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	changed := c.cancelLocked(true)
	return changed, c.status
}

func (c *IdleCountdownController) cancelLocked(preservePause bool) bool {
	if c.status == IdleCountdownFinalizing {
		return false
	}
	if preservePause && c.status == IdleCountdownPaused {
		return false
	}
	changed := c.status != IdleCountdownInactive
	c.status = IdleCountdownInactive
	c.remaining = 0
	c.deadline = time.Time{}
	if changed {
		c.persistLocked()
	}
	return changed
}

// Pause 冻结当前剩余时间。
func (c *IdleCountdownController) Pause() (IdleCountdownState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if c.status == IdleCountdownFinalizing || (c.status == IdleCountdownRunning && !c.deadline.After(now)) {
		c.expireLocked(now)
		return c.stateLocked(now), ErrIdleCountdownBusy
	}
	if c.status != IdleCountdownRunning {
		return c.stateLocked(now), ErrIdleCountdownNotRun
	}

	c.remaining = c.deadline.Sub(now)
	c.deadline = time.Time{}
	c.status = IdleCountdownPaused
	c.persistLocked()
	return c.stateLocked(now), nil
}

// Resume 从暂停时的剩余时间继续倒计时。
func (c *IdleCountdownController) Resume() (IdleCountdownState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if c.status == IdleCountdownFinalizing {
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
	if c.status != IdleCountdownFinalizing {
		return
	}
	c.status = IdleCountdownInactive
	c.remaining = 0
	c.deadline = time.Time{}
	c.persistLocked()
}

func (c *IdleCountdownController) startAt(duration time.Duration, now time.Time) (IdleCountdownState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		return c.stateLocked(now), ErrIdleCountdownDisabled
	}
	if c.status == IdleCountdownFinalizing {
		return c.stateLocked(now), ErrIdleCountdownBusy
	}
	c.startLocked(duration, now)
	return c.stateLocked(now), nil
}

func (c *IdleCountdownController) startLocked(duration time.Duration, now time.Time) {
	c.status = IdleCountdownRunning
	c.remaining = duration
	c.deadline = now.Add(duration)
	c.persistLocked()
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

// restoreLocked 读取持久化状态并恢复。超过截止时间的运行中倒计时不会恢复，
// 而是交回快照逻辑重新判断，避免进程停机后一启动就无条件触发归档流程。
func (c *IdleCountdownController) restoreLocked() {
	if c.statePath == "" {
		return
	}

	data, err := os.ReadFile(c.statePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			c.logError("读取关服倒计时状态失败: %v", err)
		}
		return
	}

	var saved idleCountdownPersistedState
	if err := json.Unmarshal(data, &saved); err != nil {
		c.logError("解析关服倒计时状态失败: %v", err)
		return
	}

	now := c.now()
	maxDuration := time.Duration(maxIdleCountdownSeconds) * time.Second
	switch saved.Status {
	case IdleCountdownPaused:
		if saved.RemainingMs <= 0 {
			return
		}
		if saved.RemainingMs > maxDuration.Milliseconds() {
			saved.RemainingMs = maxDuration.Milliseconds()
		}
		c.status = IdleCountdownPaused
		c.remaining = time.Duration(saved.RemainingMs) * time.Millisecond
		c.deadline = time.Time{}
		c.logInfo("已恢复暂停的倒计时：剩余 %d 秒", int64(math.Ceil(c.remaining.Seconds())))
	case IdleCountdownRunning:
		remaining := time.UnixMilli(saved.DeadlineMs).Sub(now)
		if remaining <= 0 {
			return
		}
		if remaining > maxDuration {
			remaining = maxDuration
		}
		c.status = IdleCountdownRunning
		c.remaining = remaining
		c.deadline = now.Add(remaining)
		c.logInfo("已恢复运行中的倒计时：剩余 %d 秒", int64(math.Ceil(c.remaining.Seconds())))
	}
}

// persistLocked 把当前状态写入 statePath。未启用持久化时不做任何事。
// 未启动与关服中不保留状态：前者没有需要恢复的内容，后者恢复后无人消费
// Expired 事件，重新判断更安全。
func (c *IdleCountdownController) persistLocked() {
	if c.statePath == "" {
		return
	}

	if c.status != IdleCountdownPaused && c.status != IdleCountdownRunning {
		if err := os.Remove(c.statePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			c.logError("删除关服倒计时状态失败: %v", err)
		}
		return
	}

	saved := idleCountdownPersistedState{Status: c.status}
	if c.status == IdleCountdownPaused {
		saved.RemainingMs = c.remaining.Milliseconds()
	} else {
		saved.DeadlineMs = c.deadline.UnixMilli()
	}

	data, err := json.Marshal(saved)
	if err != nil {
		c.logError("序列化关服倒计时状态失败: %v", err)
		return
	}

	if err := os.MkdirAll(filepath.Dir(c.statePath), 0755); err != nil {
		c.logError("创建关服倒计时状态目录失败: %v", err)
		return
	}

	tmpPath := c.statePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		c.logError("写入关服倒计时状态失败: %v", err)
		return
	}
	if err := os.Rename(tmpPath, c.statePath); err != nil {
		c.logError("替换关服倒计时状态失败: %v", err)
	}
}

func (c *IdleCountdownController) logInfo(format string, args ...any) {
	if c.logger != nil {
		c.logger.Info(format, args...)
	}
}

func (c *IdleCountdownController) logError(format string, args ...any) {
	if c.logger != nil {
		c.logger.Error(format, args...)
	}
}

func (c *IdleCountdownController) expireLocked(now time.Time) {
	if c.status != IdleCountdownRunning || c.deadline.After(now) {
		return
	}
	c.status = IdleCountdownFinalizing
	c.remaining = 0
	c.deadline = time.Time{}
	c.persistLocked()
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
			// 程序退出时保留已持久化的状态，重启后可恢复暂停/剩余时间。
			return
		case now := <-ticker.C:
			c.mu.Lock()
			c.expireLocked(now)
			c.mu.Unlock()
		}
	}
}
