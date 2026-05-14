package monitors

import (
	"context"
	"errors"
	"time"

	"github.com/Subilan/go-aliyunmc/log_util"
	"github.com/Subilan/go-aliyunmc/store/models"
	"github.com/Subilan/go-aliyunmc/tasks"
)

// BackupMonitor 定时触发备份任务。触发前会检查实例是否运行，并复用 tasks 的校验与执行逻辑。
type BackupMonitor struct {
	enabled  bool
	interval time.Duration
	logger   *log_util.NamedLogger
}

func newBackupMonitor() *BackupMonitor {
	return &BackupMonitor{
		enabled:  BackupC.Enabled,
		interval: time.Duration(BackupC.PollIntervalSec) * time.Second,
		logger:   log_util.NewNamedLogger("[monitor/backup] ", "backup-monitor"),
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

	m.triggerOnce()

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
	m.logger.Info("开始执行备份")
	task, err := tasks.TriggerTaskSync(models.TaskTypeBackup, nil, nil)
	if err != nil {
		if errors.Is(err, tasks.ErrTaskTypeExecuting) {
			m.logger.Info("跳过备份，因为已有备份任务在执行")
			return
		}
		if errors.Is(err, tasks.ErrTaskTypeNotFound) {
			m.logger.Error("backup task definition not found")
			return
		}
		m.logger.Info("跳过备份，因为不符合执行条件或出错：%v", err)
		return
	}
	m.logger.Info("备份完成，task_id=%d", task.ID)
}
