package tasks

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/remote_util"
	"github.com/Subilan/go-aliyunmc/store"
)

func backupTask(tc *TaskContext, _ map[string]any) error {
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}

	script, err := remote_util.RenderScriptTemplate(config.Get().Tasks.Backup.TemplatePath, config.Get().Tasks.Backup.Vars)
	if err != nil {
		return err
	}

	err = remote_util.Root(ip).ExecScript(tc.Context(), script, tc.println)
	if err != nil {
		return err
	}

	return nil
}

func checkBackupTask(args map[string]any) error {
	if IsArchiveTaskRunning() {
		return h.HttpError(http.StatusConflict, "自动回收流程正在执行，不允许触发backup任务")
	}

	if err := checkMustHaveActiveDeployedRunningInstance(args); err != nil {
		return err
	}

	snap := SnapshotServerStatus()

	if !snap.IsValid() {
		return h.HttpError(http.StatusServiceUnavailable, "无法获取最新的服务器状态")
	}

	if !snap.Value.Online {
		return h.HttpError(http.StatusServiceUnavailable, "服务器未运行，跳过备份")
	}

	return nil
}
