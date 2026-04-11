package monitors

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go-aliyunmc/aliyun"
	"go-aliyunmc/logs"
	"go-aliyunmc/remote_util"
	"go-aliyunmc/server"
	"go-aliyunmc/states"
	"go-aliyunmc/store"
	"go-aliyunmc/utils"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"gorm.io/gorm"
)

var autoArchiveFlowRunning atomic.Bool

type archiveTaskRuntimeConfig struct {
	TemplatePath string `toml:"template_path" validate:"required"`
	TimeoutSec   int    `toml:"timeout_sec" validate:"required,min=1"`
	Vars         struct {
		OSSRoot string `toml:"oss_root" validate:"required"`
	} `toml:"vars" validate:"required"`
}

type AutoArchiveIdleMonitor struct {
	enabled               bool
	idleCountdown         time.Duration
	stopWaitTimeout       time.Duration
	offlineCheckInterval  time.Duration
	ignoreMissingOnDelete bool
	archiveCfg            archiveTaskRuntimeConfig
	logger                *logs.PrefixedLogger
}

func newAutoArchiveIdleMonitor() *AutoArchiveIdleMonitor {
	archiveCfg := archiveTaskRuntimeConfig{}
	utils.MustBindConfig(&archiveCfg, "task-archive")
	logger := logs.NewPrefixedLogger("[monitor/auto-archive-idle] ")

	return &AutoArchiveIdleMonitor{
		enabled:               AutoArchiveIdleC.Enabled,
		idleCountdown:         time.Duration(AutoArchiveIdleC.IdleCountdownSec) * time.Second,
		stopWaitTimeout:       time.Duration(AutoArchiveIdleC.StopWaitTimeoutSec) * time.Second,
		offlineCheckInterval:  time.Duration(AutoArchiveIdleC.OfflineCheckIntervalSec) * time.Second,
		ignoreMissingOnDelete: AutoArchiveIdleC.DeleteIgnoreNonExistent,
		archiveCfg:            archiveCfg,
		logger:                logger,
	}
}

// IsAutoArchiveFlowRunning 返回空服自动回收流程是否正在执行中。
func IsAutoArchiveFlowRunning() bool {
	return autoArchiveFlowRunning.Load()
}

// run 是AutoArchiveIdleMonitor的主循环，监听服务器状态快照更新并根据配置自动执行归档回收流程。
func (m *AutoArchiveIdleMonitor) run(ctx context.Context) {
	if !m.enabled {
		m.logger.Info("disabled")
		return
	}

	var (
		countdownTimer *time.Timer
		countdownC     <-chan time.Time
		running        bool
	)

	doneCh := make(chan error, 1)

	cancelCountdown := func(reason string) {
		if countdownTimer == nil {
			return
		}
		if !countdownTimer.Stop() {
			select {
			// 消费
			case <-countdownTimer.C:
			default:
			}
		}
		countdownTimer = nil
		countdownC = nil
		m.logger.Info("倒计时取消，原因：%s", reason)
	}

	startCountdown := func() {
		if countdownTimer != nil || running {
			return
		}
		if !store.HasActiveInstance() {
			return
		}
		countdownTimer = time.NewTimer(m.idleCountdown)
		countdownC = countdownTimer.C
		m.logger.Info("倒计时开始：%d", int(m.idleCountdown.Seconds()))
	}

	// 根据当前快照状态决定是否取消倒计时或者开启倒计时
	handleSnapshot := func(s states.State[states.ServerStatusState]) {
		if s.Error != nil {
			return
		}

		if !store.HasActiveInstance() {
			cancelCountdown("active_instance_missing")
			return
		}

		if s.Value.PlayerCount > 0 {
			// 服务器一定在线
			cancelCountdown("players_back")
			return
		}

		// 此时，要么服务器不在线，要么服务器在线但没有玩家。
		startCountdown()
	}

	updates, unsubscribe := serverMonitor.Subscribe()
	defer unsubscribe()

	current, ok := states.SnapshotServerStatus()
	if ok {
		handleSnapshot(current)
	} else {
		m.logger.Info("无法获取服务器状态快照，直接进入监听循环")
	}

	for {
		select {
		case <-ctx.Done():
			cancelCountdown("context_done")
			return
		case s, ok := <-updates:
			if !ok {
				cancelCountdown("updates_channel_closed")
				return
			}
			current = s
			handleSnapshot(s)
		case <-countdownC:
			countdownTimer = nil
			countdownC = nil
			if running {
				continue
			}
			running = true
			autoArchiveFlowRunning.Store(true)
			m.logger.Info("倒计时结束，开始执行归档流程")
			go func() {
				defer autoArchiveFlowRunning.Store(false)
				err := m.executeArchivePipeline(ctx)
				select {
				case doneCh <- err:
				default:
				}
			}()
		case err := <-doneCh:
			running = false
			if err != nil {
				m.logger.Error("归档流程失败：%v", err)
			} else {
				m.logger.Info("归档流程完成")
			}
			handleSnapshot(current)
		}
	}
}

// executeArchivePipeline 执行服务器离线、数据归档和实例记录删除的流程。
func (m *AutoArchiveIdleMonitor) executeArchivePipeline(ctx context.Context) error {
	instance, err := store.GetActiveInstanceDefaultNil()

	if err != nil {
		return err
	}

	if instance == nil {
		m.logger.Info("skip pipeline: active instance not found")
		return nil
	}

	if instance.Ip == "" {
		return fmt.Errorf("active instance ip is empty")
	}

	if states.SnapshotIsServerOnline() {
		m.logger.Info("stop_sent")
		if _, err := server.RunSingleCommand(instance.Ip, "stop"); err != nil {
			return fmt.Errorf("send stop command failed: %w", err)
		}
		if err := m.waitServerOffline(ctx); err != nil {
			return err
		}
		m.logger.Info("offline_confirmed")
	} else {
		m.logger.Info("skip stop: already offline or status unavailable")
	}

	if err := m.runArchiveTask(ctx); err != nil {
		return err
	}

	m.logger.Info("archive_done")

	if err := m.deleteActiveInstance(); err != nil {
		return err
	}

	m.logger.Info("delete_done")
	return nil
}

// waitServerOffline 等待服务器离线，期间会定期检查服务器状态快照并监听服务器状态变化，以尽早发现服务器已离线的情况。
func (m *AutoArchiveIdleMonitor) waitServerOffline(ctx context.Context) error {
	waitCtx, cancel := context.WithTimeout(ctx, m.stopWaitTimeout)
	defer cancel()

	updates, unsubscribe := states.SubscribeServerSnapshot()
	defer unsubscribe()

	ticker := time.NewTicker(m.offlineCheckInterval)
	defer ticker.Stop()

	if states.SnapshotIsServerOffline() {
		return nil
	}

	for {
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("wait server offline timeout: %w", waitCtx.Err())
		case <-ctx.Done():
			return ctx.Err()
		case s, ok := <-updates:
			if ok && s.Error == nil && !s.Value.Online {
				return nil
			}
		case <-ticker.C:
			if states.SnapshotIsServerOffline() {
				return nil
			}
		}
	}
}

// runArchiveTask 执行归档任务。
func (m *AutoArchiveIdleMonitor) runArchiveTask(ctx context.Context) error {
	instance, err := store.GetActiveInstanceDefaultNil()
	if err != nil {
		return err
	}
	if instance == nil {
		return fmt.Errorf("active instance not found before archive")
	}
	if instance.Ip == "" {
		return fmt.Errorf("active instance ip is empty before archive")
	}

	script, err := remote_util.RenderScriptTemplate(m.archiveCfg.TemplatePath, m.archiveCfg.Vars)

	if err != nil {
		return err
	}

	m.logger.Info("archive_started")
	archiveCtx, cancel := context.WithTimeout(ctx, time.Duration(m.archiveCfg.TimeoutSec)*time.Second)
	defer cancel()

	if err := remote_util.ExecuteScriptRemote(script, instance.Ip, archiveCtx, nil, true); err != nil {
		return err
	}

	return nil
}

// deleteActiveInstance 删除当前的活跃实例记录，并尝试删除对应的云服务器实例。如果配置了 ignoreMissingOnDelete，则在数据库记录不存在时不返回错误。
func (m *AutoArchiveIdleMonitor) deleteActiveInstance() error {
	instance, err := store.GetActiveInstanceDefaultNil()
	if err != nil {
		return err
	}
	if instance == nil {
		return nil
	}

	if aliyun.EcsClient == nil {
		return fmt.Errorf("ecs client is nil")
	}

	deleteReq := &ecs20140526.DeleteInstanceRequest{
		InstanceId: tea.String(instance.InstanceId),
		Force:      tea.Bool(true),
		ForceStop:  tea.Bool(true),
	}

	if _, err := aliyun.EcsClient.DeleteInstance(deleteReq); err != nil {
		return fmt.Errorf("delete ecs instance failed: %w", err)
	}

	if err := store.DeleteActiveInstance(); err != nil {
		if m.ignoreMissingOnDelete && errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("delete active instance record failed: %w", err)
	}

	return nil
}
