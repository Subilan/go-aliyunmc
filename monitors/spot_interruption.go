package monitors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/aliyun"
	"github.com/Subilan/go-aliyunmc/global_states"
	"github.com/Subilan/go-aliyunmc/log_util"
	"github.com/Subilan/go-aliyunmc/server"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

// SpotInterruptionMonitor 监控抢占式实例的回收通知。
//
// 阿里云会在抢占式实例被回收前约 5 分钟，通过实例的 OperationLocks 字段（LockReason 为
// Recycling）发出待回收通知。本监控器定期调用 DescribeInstances 感知该通知，并在收到
// 通知后立即保护性停机 Minecraft 服务器（触发全盘保存），避免实例被强制断电导致回档。
// 归档不在此流程中执行：数据依赖实例磁盘保留（spot_interruption_behavior = Stop），
// 由管理员在实例被回收后通过 /instance/start 重新开启实例来恢复服务。
//
// 停机后系统会进入回收保护状态，期间阻止 start_server 任务触发。当实例通过
// /instance/start 接口被管理员重新启动并恢复 Running 后，保护状态自动解除。
type SpotInterruptionMonitor struct {
	enabled              bool
	interval             time.Duration
	stopWaitTimeout      time.Duration
	offlineCheckInterval time.Duration
	logger               *log_util.NamedLogger

	handlingMu sync.Mutex
	handling   bool // 保护流水线是否正在执行，防止重复触发
}

func newSpotInterruptionMonitor() *SpotInterruptionMonitor {
	return &SpotInterruptionMonitor{
		enabled:              SpotInterruptionC.Enabled,
		interval:             time.Duration(SpotInterruptionC.PollIntervalSec) * time.Second,
		stopWaitTimeout:      time.Duration(SpotInterruptionC.StopWaitTimeoutSec) * time.Second,
		offlineCheckInterval: time.Duration(SpotInterruptionC.OfflineCheckIntervalSec) * time.Second,
		logger:               log_util.NewNamedLogger("[monitor/spot-interruption] ", "spot-interruption-monitor"),
	}
}

func (m *SpotInterruptionMonitor) run(ctx context.Context) {
	if !m.enabled {
		m.logger.Info("disabled")
		return
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	m.pollOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pollOnce(ctx)
		}
	}
}

// pollOnce 查询当前活跃实例的回收状态并据此更新全局保护状态。
func (m *SpotInterruptionMonitor) pollOnce(ctx context.Context) {
	instance, err := store.GetActiveInstanceDefaultNil()
	if err != nil {
		m.logger.Warn("查询活跃实例失败: %v", err)
		return
	}

	// 活跃实例不存在时，回收保护状态已无意义（无法启动服务器），自动解除，避免残留。
	if instance == nil {
		if global_states.IsSpotInterruptionActive() {
			m.logger.Info("活跃实例不存在，自动解除回收保护状态")
			global_states.ClearSpotInterruption()
		}
		return
	}

	if aliyun.EcsClient == nil {
		m.logger.Warn("EcsClient 未初始化，跳过回收状态检查")
		return
	}

	resp, err := aliyun.EcsClient.DescribeInstancesWithContext(ctx, &ecs20140526.DescribeInstancesRequest{
		RegionId:    tea.String(aliyun.C.RegionId),
		InstanceIds: tea.String(instance.InstanceId),
	}, &dara.RuntimeOptions{})

	if err != nil {
		m.logger.Warn("查询实例回收状态失败: %v", err)
		return
	}

	// 在响应中定位目标实例，获取其回收锁和状态。
	var (
		recycling = false
		status    string
		found     = false
	)

	if resp != nil && resp.Body != nil && resp.Body.Instances != nil {
		for _, item := range resp.Body.Instances.Instance {
			if item == nil || item.InstanceId == nil || tea.StringValue(item.InstanceId) != instance.InstanceId {
				continue
			}
			found = true
			if item.Status != nil {
				status = tea.StringValue(item.Status)
			}
			if item.OperationLocks != nil {
				for _, lock := range item.OperationLocks.LockReason {
					if lock != nil && tea.StringValue(lock.LockReason) == "Recycling" {
						recycling = true
						break
					}
				}
			}
		}
	}

	// 实例在云上已不存在（被释放或删除）。与活跃实例不存在一样，自动解除保护状态。
	if !found {
		if global_states.IsSpotInterruptionActive() {
			m.logger.Warn("活跃实例在云上已不存在，自动解除回收保护状态")
			global_states.ClearSpotInterruption()
		}
		return
	}

	// 实例已不再处于待回收状态：若已恢复运行则解除保护。
	// 注意：必须先于 recycling 判断执行，避免实例恢复 Running 后 Recycling lock 仍残留导致保护无法解除。
	if global_states.IsSpotInterruptionActive() && status == "Running" {
		m.logger.Info("实例 %s 已恢复运行，解除回收保护状态", instance.InstanceId)
		global_states.ClearSpotInterruption()
		return
	}

	if recycling {
		m.handleRecycling(instance)
	}
}

// handleRecycling 处理抢占式实例待回收通知：进入保护状态并执行保护性停机流水线。
func (m *SpotInterruptionMonitor) handleRecycling(instance *models.Instance) {
	if global_states.IsSpotInterruptionActive() {
		// 已处于保护状态，不重复处理。
		return
	}

	m.logger.Warn("检测到抢占式实例 %s 即将被回收（LockReason=Recycling），进入回收保护状态", instance.InstanceId)
	global_states.SetSpotInterruption(instance.InstanceId)

	m.handlingMu.Lock()
	if m.handling {
		m.handlingMu.Unlock()
		return
	}
	m.handling = true
	m.handlingMu.Unlock()

	go func() {
		defer func() {
			m.handlingMu.Lock()
			m.handling = false
			m.handlingMu.Unlock()
		}()
		m.executePipeline()
	}()
}

// executePipeline 执行保护性停机流水线：停止服务器（触发全盘保存）→ 等待离线。
// 归档不在本流程内执行，数据依赖实例磁盘保留（spot_interruption_behavior = Stop），
// 由管理员在实例被回收后通过 /instance/start 重新开启实例恢复服务。
func (m *SpotInterruptionMonitor) executePipeline() {
	instance, err := store.GetActiveInstanceDefaultNil()
	if err != nil || instance == nil {
		m.logger.Error("保护性停机失败：无法获取活跃实例")
		return
	}
	if instance.Ip == "" {
		m.logger.Error("保护性停机失败：活跃实例 IP 为空")
		return
	}

	ctx := context.Background()

	// 保护性停机。失败（服务器始终未离线）说明存档可能仍在写入。
	if err := m.gracefulShutdown(ctx, instance); err != nil {
		m.logger.Error("保护性停机未完成：%v", err)
		m.logger.Error("服务器可能仍在写入存档，实例被回收时存在回档风险，请关注实例磁盘数据")
		return
	}

	m.logger.Warn("保护性停机流水线结束。实例仍处于待回收状态，请等待实例被回收后通过 /instance/start 重新开启实例")
}

// gracefulShutdown 向服务器发送 stop 命令并等待其离线。
// 只有服务器确认离线（存档写入完成）才返回 nil；若在超时时间内始终在线，返回错误。
func (m *SpotInterruptionMonitor) gracefulShutdown(ctx context.Context, instance *models.Instance) error {
	snap := serverMonitor.Snapshot()
	if snap.IsValid() && !snap.Value.Online {
		m.logger.Info("服务器已离线，跳过关闭步骤")
		return nil
	}

	// RCON stop 可能因瞬时网络问题失败，重试数次。
	const maxStopAttempts = 3
	var lastStopErr error
	for i := 1; i <= maxStopAttempts; i++ {
		m.logger.Info("发送 stop 命令（第 %d/%d 次）", i, maxStopAttempts)
		if _, lastStopErr = server.RunSingleCommand(instance.Ip, "stop"); lastStopErr == nil {
			break
		}
		m.logger.Warn("发送 stop 命令失败: %v", lastStopErr)
		time.Sleep(2 * time.Second)
	}

	// 无论 stop 命令是否显式成功，都以服务器最终离线为准：
	// 离线即代表 Minecraft 已完成保存并退出，磁盘数据稳定。
	if err := waitServerOffline(ctx, m.stopWaitTimeout, m.offlineCheckInterval); err != nil {
		return fmt.Errorf("等待服务器离线超时: %w", err)
	}

	if lastStopErr != nil {
		m.logger.Warn("stop 命令未能显式成功，但服务器已离线，视为停机完成")
	}
	return nil
}
