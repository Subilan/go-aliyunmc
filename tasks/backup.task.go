package tasks

import (
	"go-aliyunmc/h"
	"go-aliyunmc/remote_util"
	"go-aliyunmc/states"
	"go-aliyunmc/store"
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
	if states.IsArchiveTaskRunning() {
		return h.HttpError(http.StatusConflict, "自动回收流程正在执行，不允许触发backup任务")
	}
	
	return checkMustHaveActiveDeployedRunningInstance(args)
}
