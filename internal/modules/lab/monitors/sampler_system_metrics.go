package monitors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/states"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/remote_util"
)

// systemMetricsCommand 读取远端主机的 CPU 累计计数、内存与运行时长。
// CPU 使用率由相邻两次采样的差值计算，脚本本身不需要等待。
const systemMetricsCommand = `awk '
/^cpu / {
  total = 0
  for (i = 2; i <= 9; i++) total += $i
  printf "cpu_total=%d\n", total
  printf "cpu_idle=%d\n", $5 + $6
}
/^MemTotal:/ { printf "mem_total_kb=%d\n", $2 }
/^MemAvailable:/ { printf "mem_available_kb=%d\n", $2 }
' /proc/stat /proc/meminfo
cores=$(getconf _NPROCESSORS_ONLN 2>/dev/null || grep -c '^processor' /proc/cpuinfo)
printf 'cpu_cores=%s\n' "${cores:-1}"
read -r uptime _ < /proc/uptime
printf 'uptime_seconds=%s\n' "${uptime%%.*}"
`

const systemMetricsTimeout = 20 * time.Second

// SystemMetricsState 是实例主机的最近一次资源采样结果。
// Available 为 false 时其余字段没有意义，表示当前无法连接实例或实例不可采样。
type SystemMetricsState struct {
	Available       bool
	UptimeSeconds   int64
	CPUCores        int
	CPUUsagePercent float64
	// CPUValid 表示采样器已经取得两次 CPU 计数，可以计算使用率。
	CPUValid         bool
	MemoryUsedBytes  uint64
	MemoryTotalBytes uint64
}

// SystemMetricsSampler 定期通过 SSH 读取实例主机的 CPU 与内存占用。
// 实例不存在、未部署或未运行时不做采样，以免在不可用的目标上反复重连。
type SystemMetricsSampler struct {
	interval time.Duration
	store    *states.HubbedStore[SystemMetricsState]
	logger   *log_util.NamedLogger

	// 以下字段只由采样 goroutine 读写。
	cpuIP        string
	hasCPUSample bool
	lastCPUTotal uint64
	lastCPUIdle  uint64
	wasAvailable bool
}

func newSystemMetricsSampler() *SystemMetricsSampler {
	return &SystemMetricsSampler{
		interval: time.Duration(config.Get().Samplers.SystemMetrics.SampleIntervalSec) * time.Second,
		store:    states.NewHubbedStore[SystemMetricsState](),
		logger:   log_util.NewNamedLogger("[sampler/system-metrics] ", "system-metrics-sampler"),
	}
}

func (s *SystemMetricsSampler) Snapshot() states.State[SystemMetricsState] {
	return s.store.Snapshot()
}

func (s *SystemMetricsSampler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.sampleAndStore(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sampleAndStore(ctx)
		}
	}
}

// sampleTarget 返回当前可采样的实例 IP；不满足条件时返回空字符串。
func (s *SystemMetricsSampler) sampleTarget() string {
	instance, err := dao.GetActiveInstanceDefaultNil()
	if err != nil || instance == nil || instance.Ip == "" || !instance.IsDeployed {
		return ""
	}

	// 状态快照无效时允许尝试，实例可能刚好进入 Running；已知未运行时直接跳过。
	if snap := instanceMonitor.Snapshot(); snap.IsValid() && snap.Value != "Running" {
		return ""
	}

	return instance.Ip
}

func (s *SystemMetricsSampler) sampleAndStore(ctx context.Context) {
	ip := s.sampleTarget()
	if ip == "" {
		s.resetCPUSample()
		s.markUnavailable()
		return
	}

	execCtx, cancel := context.WithTimeout(ctx, systemMetricsTimeout)
	defer cancel()

	var output strings.Builder
	err := remote_util.Mc(ip).Exec(execCtx, systemMetricsCommand, func(line string) {
		output.WriteString(line)
		output.WriteByte('\n')
	})
	if err != nil {
		s.resetCPUSample()
		s.markUnavailable()
		if s.wasAvailable {
			s.logger.Warn("系统资源采样失败: %v", err)
		}
		s.wasAvailable = false
		return
	}

	raw, err := parseSystemMetrics(output.String())
	if err != nil {
		s.resetCPUSample()
		s.markUnavailable()
		if s.wasAvailable {
			s.logger.Warn("解析系统资源采样结果失败: %v", err)
		}
		s.wasAvailable = false
		return
	}

	state := SystemMetricsState{
		Available:        true,
		UptimeSeconds:    int64(raw.uptimeSeconds),
		CPUCores:         int(raw.cpuCores),
		MemoryUsedBytes:  memoryUsedBytes(raw.memTotalKB, raw.memAvailableKB),
		MemoryTotalBytes: raw.memTotalKB * 1024,
	}

	if s.cpuIP == ip && s.hasCPUSample &&
		raw.cpuTotal >= s.lastCPUTotal && raw.cpuIdle >= s.lastCPUIdle {
		totalDelta := raw.cpuTotal - s.lastCPUTotal
		idleDelta := raw.cpuIdle - s.lastCPUIdle
		if totalDelta > 0 && idleDelta <= totalDelta {
			state.CPUUsagePercent = float64(totalDelta-idleDelta) * 100 / float64(totalDelta)
			state.CPUValid = true
		}
	}

	s.cpuIP = ip
	s.hasCPUSample = true
	s.lastCPUTotal = raw.cpuTotal
	s.lastCPUIdle = raw.cpuIdle
	s.wasAvailable = true

	s.store.Store(state, s.logger)
}

func (s *SystemMetricsSampler) resetCPUSample() {
	s.cpuIP = ""
	s.hasCPUSample = false
}

func (s *SystemMetricsSampler) markUnavailable() {
	// 已经处于不可用状态时不做任何事，避免每次失败都触发状态广播。
	if s.store.Snapshot().Value.Available {
		s.store.Store(SystemMetricsState{}, s.logger)
	}
}

type systemMetricsRaw struct {
	cpuTotal       uint64
	cpuIdle        uint64
	memTotalKB     uint64
	memAvailableKB uint64
	cpuCores       uint64
	uptimeSeconds  uint64
}

// parseSystemMetrics 解析远端脚本输出的 key=value 行。
func parseSystemMetrics(output string) (systemMetricsRaw, error) {
	values := make(map[string]uint64)
	for line := range strings.SplitSeq(output, "\n") {
		key, raw, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return systemMetricsRaw{}, fmt.Errorf("字段 %s 的值无效: %q", key, raw)
		}
		values[key] = value
	}

	for _, key := range []string{
		"cpu_total", "cpu_idle", "mem_total_kb", "mem_available_kb", "cpu_cores", "uptime_seconds",
	} {
		if _, ok := values[key]; !ok {
			return systemMetricsRaw{}, fmt.Errorf("缺少字段 %s", key)
		}
	}

	if values["mem_total_kb"] == 0 {
		return systemMetricsRaw{}, fmt.Errorf("内存总量为 0")
	}
	if values["cpu_cores"] == 0 {
		return systemMetricsRaw{}, fmt.Errorf("CPU 核心数为 0")
	}

	return systemMetricsRaw{
		cpuTotal:       values["cpu_total"],
		cpuIdle:        values["cpu_idle"],
		memTotalKB:     values["mem_total_kb"],
		memAvailableKB: values["mem_available_kb"],
		cpuCores:       values["cpu_cores"],
		uptimeSeconds:  values["uptime_seconds"],
	}, nil
}

func memoryUsedBytes(totalKB, availableKB uint64) uint64 {
	if availableKB >= totalKB {
		return 0
	}
	return (totalKB - availableKB) * 1024
}
