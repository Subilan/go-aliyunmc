package tasks

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/remote_util"
	"github.com/Subilan/go-aliyunmc/store"
	"net/http"
)

func backupTask(tc *TaskContext, _ map[string]any) error {
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}

	script, err := remote_util.RenderScriptTemplate(BackupC.TemplatePath, BackupC.Vars)
	if err != nil {
		return err
	}

	err = remote_util.ExecuteScriptRemote(script, ip, tc.Context(), tc.println, true)
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
