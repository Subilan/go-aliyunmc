package monitors

import (
	"context"
	"errors"
	"time"

	"go-aliyunmc/logs"
	"go-aliyunmc/store/models"
	"go-aliyunmc/tasks"
)

// BackupMonitor 定时触发备份任务。触发前会检查实例是否运行，并复用 tasks 的校验与执行逻辑。
type BackupMonitor struct {
	enabled  bool
	interval time.Duration
	logger   *logs.PrefixedLogger
}

func newBackupMonitor() *BackupMonitor {
	return &BackupMonitor{
		enabled:  BackupC.Enabled,
		interval: time.Duration(BackupC.PollIntervalSec) * time.Second,
		logger:   logs.NewPrefixedLogger("[monitor/backup] "),
	}
}

func (m *BackupMonitor) run(ctx context.Context) {
	if !m.enabled {
		m.logger.Info("disabled, interrupting monitor")
		return
	}

	if m.interval <= 0 {
		m.logger.Error("invalid poll interval, interrupting monitor")
		return
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.triggerOnce()
		}
	}
}

func (m *BackupMonitor) triggerOnce() {
	m.logger.Info("执行backup任务")
	task, err := tasks.TriggerTaskSync(models.TaskTypeBackup, nil, nil)
	if err != nil {
		if errors.Is(err, tasks.ErrTaskTypeExecuting) {
			m.logger.Info("backup任务已在执行，跳过本次触发")
			return
		}
		if errors.Is(err, tasks.ErrTaskTypeNotFound) {
			m.logger.Error("backup task definition not found")
			return
		}
		m.logger.Info("backup校验未通过或执行失败，跳过触发: %v", err)
		return
	}
	m.logger.Info("backup任务完成，task_id=%d", task.ID)
}
