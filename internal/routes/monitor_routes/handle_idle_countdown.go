package monitor_routes

import (
	"errors"
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/monitors"
	"github.com/Subilan/go-aliyunmc/internal/store"

	"github.com/gin-gonic/gin"
)

type setIdleCountdownRequest struct {
	RemainingSeconds int64 `json:"remaining_seconds"`
}

type idleCountdownResponse struct {
	monitors.IdleCountdownState
	HasActiveInstance bool `json:"has_active_instance"`
}

// HandleGetIdleCountdown 返回关服倒计时的完整状态。
func HandleGetIdleCountdown(c *gin.Context) (any, error) {
	countdown, err := getIdleCountdown()
	if err != nil {
		return nil, err
	}
	return newIdleCountdownResponse(countdown.State())
}

// HandleGetIdleCountdownRemainingSecs 保留首页使用的旧接口；未启动时返回 -1。
func HandleGetIdleCountdownRemainingSecs() int64 {
	countdown, err := getIdleCountdown()
	if err != nil {
		return -1
	}
	state := countdown.State()
	if state.Status == monitors.IdleCountdownInactive {
		return -1
	}
	return state.RemainingSeconds
}

// HandlePauseIdleCountdown 暂停当前关服倒计时。
func HandlePauseIdleCountdown(c *gin.Context) (any, error) {
	return runCountdownControl(false, func(countdown *monitors.IdleCountdownController) (monitors.IdleCountdownState, error) {
		return countdown.Pause()
	})
}

// HandleResumeIdleCountdown 从暂停状态继续关服倒计时。
func HandleResumeIdleCountdown(c *gin.Context) (any, error) {
	return runCountdownControl(true, func(countdown *monitors.IdleCountdownController) (monitors.IdleCountdownState, error) {
		return countdown.Resume()
	})
}

// HandleResetIdleCountdown 使用配置的默认时长重新开始关服倒计时。
func HandleResetIdleCountdown(c *gin.Context) (any, error) {
	return runCountdownControl(true, func(countdown *monitors.IdleCountdownController) (monitors.IdleCountdownState, error) {
		return countdown.Reset()
	})
}

// HandleSetIdleCountdown 使用手动指定的剩余秒数重新开始关服倒计时。
func HandleSetIdleCountdown(req setIdleCountdownRequest, c *gin.Context) (any, error) {
	return runCountdownControl(true, func(countdown *monitors.IdleCountdownController) (monitors.IdleCountdownState, error) {
		return countdown.SetRemaining(req.RemainingSeconds)
	})
}

func runCountdownControl(
	checkCanStart bool,
	action func(*monitors.IdleCountdownController) (monitors.IdleCountdownState, error),
) (any, error) {
	countdown, err := getIdleCountdown()
	if err != nil {
		return nil, err
	}
	if checkCanStart {
		if err := ensureCountdownCanStart(countdown); err != nil {
			return nil, err
		}
	} else if !countdown.State().Enabled {
		return nil, h.HttpError(http.StatusConflict, monitors.ErrIdleCountdownDisabled.Error())
	}
	state, err := action(countdown)
	if err != nil {
		return nil, countdownControlError(err)
	}
	return newIdleCountdownResponse(state)
}

func newIdleCountdownResponse(state monitors.IdleCountdownState) (idleCountdownResponse, error) {
	hasActiveInstance, err := store.HasActiveInstance()
	if err != nil {
		return idleCountdownResponse{}, h.HttpError(http.StatusServiceUnavailable, "无法确认活跃实例状态")
	}
	return idleCountdownResponse{
		IdleCountdownState: state,
		HasActiveInstance:  hasActiveInstance,
	}, nil
}

func getIdleCountdown() (*monitors.IdleCountdownController, error) {
	monitor := monitors.GetAutoArchiveIdleMonitor()
	if monitor == nil || monitor.Countdown() == nil {
		return nil, h.HttpError(http.StatusServiceUnavailable, "关服倒计时服务尚未就绪")
	}
	return monitor.Countdown(), nil
}

func ensureCountdownCanStart(countdown *monitors.IdleCountdownController) error {
	state := countdown.State()
	if !state.Enabled {
		return h.HttpError(http.StatusConflict, "关服倒计时未启用")
	}
	hasActiveInstance, err := store.HasActiveInstance()
	if err != nil {
		return h.HttpError(http.StatusServiceUnavailable, "无法确认活跃实例状态")
	}
	if !hasActiveInstance {
		return h.HttpError(http.StatusConflict, "暂无活跃实例，无法设置关服倒计时")
	}

	serverMonitor := monitors.GetServerStatusMonitor()
	if serverMonitor == nil {
		return h.HttpError(http.StatusServiceUnavailable, "服务器状态监控尚未就绪")
	}
	snapshot := serverMonitor.Snapshot()
	if snapshot.IsValid() && snapshot.Value.PlayerCount > 0 {
		return h.HttpError(http.StatusConflict, "服务器有玩家在线，无法设置关服倒计时")
	}
	return nil
}

func countdownControlError(err error) error {
	switch {
	case errors.Is(err, monitors.ErrIdleCountdownDuration):
		return h.HttpError(http.StatusBadRequest, err.Error())
	case errors.Is(err, monitors.ErrIdleCountdownDisabled),
		errors.Is(err, monitors.ErrIdleCountdownNotRun),
		errors.Is(err, monitors.ErrIdleCountdownNotPause),
		errors.Is(err, monitors.ErrIdleCountdownBusy):
		return h.HttpError(http.StatusConflict, err.Error())
	default:
		return err
	}
}
