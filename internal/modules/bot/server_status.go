package bot

import (
	"math"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/states"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/monitors"

	"github.com/gin-gonic/gin"
)

// botServerStatusAbsent 是活跃实例不存在时返回的最小响应。
// 机器人只据此判断当前没有可用的服务器，不涉及实例层面的细节。
type botServerStatusAbsent struct {
	Available bool `json:"available"`
}

// botServerStatus 把实例与服务器状态聚合为一个「服务器」视图。
type botServerStatus struct {
	Available     bool                    `json:"available"`
	Online        bool                    `json:"online"`
	IP            string                  `json:"ip"`
	Players       botServerStatusPlayers  `json:"players"`
	UptimeSeconds *int64                  `json:"uptimeSeconds"`
	Release       *botServerStatusRelease `json:"release,omitempty"`
	Memory        *botServerStatusMemory  `json:"memory"`
	CPU           *botServerStatusCPU     `json:"cpu"`
}

type botServerStatusPlayers struct {
	Online int64 `json:"online"`
	// Max 为 nil 表示服务器离线或尚未取得状态，无法得知最大玩家数。
	Max *int64 `json:"max"`
}

// botServerStatusRelease 是空服后的预计释放时间。At 为 Unix 秒时间戳。
type botServerStatusRelease struct {
	InSeconds int64 `json:"inSeconds"`
	At        int64 `json:"at"`
}

type botServerStatusMemory struct {
	UsedBytes    uint64  `json:"usedBytes"`
	TotalBytes   uint64  `json:"totalBytes"`
	UsagePercent float64 `json:"usagePercent"`
}

type botServerStatusCPU struct {
	UsedCores    float64 `json:"usedCores"`
	Cores        int     `json:"cores"`
	UsagePercent float64 `json:"usagePercent"`
}

func handleServerStatus(_ *gin.Context) (any, error) {
	instance, err := dao.GetActiveInstanceDefaultNil()
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return botServerStatusAbsent{Available: false}, nil
	}

	response := botServerStatus{
		Available: true,
		IP:        instance.Ip,
		Players:   botServerStatusPlayers{},
	}

	statusValid := false
	if monitor := monitors.GetServerStatusMonitor(); monitor != nil {
		snap := monitor.Snapshot()
		if snap.IsValid() {
			statusValid = true
			response.Online = snap.Value.Online
			response.Players.Online = snap.Value.PlayerCount
			if snap.Value.Online && snap.Value.MaxPlayerCount > 0 {
				maxPlayers := snap.Value.MaxPlayerCount
				response.Players.Max = &maxPlayers
			}
		}
	}

	if metrics := systemMetricsSnapshot(); metrics.IsValid() && metrics.Value.Available {
		uptime := metrics.Value.UptimeSeconds
		response.UptimeSeconds = &uptime

		if metrics.Value.MemoryTotalBytes > 0 {
			usedPercent := float64(metrics.Value.MemoryUsedBytes) * 100 / float64(metrics.Value.MemoryTotalBytes)
			response.Memory = &botServerStatusMemory{
				UsedBytes:    metrics.Value.MemoryUsedBytes,
				TotalBytes:   metrics.Value.MemoryTotalBytes,
				UsagePercent: roundPercent(usedPercent),
			}
		}

		if metrics.Value.CPUValid && metrics.Value.CPUCores > 0 {
			response.CPU = &botServerStatusCPU{
				UsedCores:    math.Round(metrics.Value.CPUUsagePercent/100*float64(metrics.Value.CPUCores)*100) / 100,
				Cores:        metrics.Value.CPUCores,
				UsagePercent: roundPercent(metrics.Value.CPUUsagePercent),
			}
		}
	}

	response.Release = idleRelease(statusValid, response.Players.Online)
	return response, nil
}

func systemMetricsSnapshot() (snap states.State[monitors.SystemMetricsState]) {
	sampler := monitors.GetSystemMetricsSampler()
	if sampler == nil {
		return snap
	}
	return sampler.Snapshot()
}

// idleRelease 返回空服倒计时对应的释放时间。
// 只有玩家数为 0 且关服倒计时正在运行或收尾时才存在确定的释放时间；
// 倒计时暂停、未启用或尚未开始时返回 nil。
func idleRelease(statusValid bool, playerCount int64) *botServerStatusRelease {
	if !statusValid || playerCount != 0 {
		return nil
	}

	monitor := monitors.GetAutoArchiveIdleMonitor()
	if monitor == nil {
		return nil
	}

	now := time.Now()
	state := monitor.Countdown().State()
	switch state.Status {
	case monitors.IdleCountdownRunning:
		return &botServerStatusRelease{
			InSeconds: state.RemainingSeconds,
			At:        now.Add(time.Duration(state.RemainingSeconds) * time.Second).Unix(),
		}
	case monitors.IdleCountdownFinalizing:
		return &botServerStatusRelease{
			InSeconds: 0,
			At:        now.Unix(),
		}
	default:
		return nil
	}
}

func roundPercent(value float64) float64 {
	return math.Round(value*10) / 10
}
