package tasks

import (
	"go-aliyunmc/h"
	"go-aliyunmc/monitors"
	"go-aliyunmc/remote_util"
	"go-aliyunmc/store"
	"net/http"
)

func archiveTask(tc *TaskContext, _ map[string]any) error {
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}

	script, err := remote_util.RenderScriptTemplate(ArchiveC.TemplatePath, ArchiveC.Vars)
	if err != nil {
		return err
	}

	err = remote_util.ExecuteScriptRemote(script, ip, tc.Context(), tc.println, true)
	if err != nil {
		return err
	}

	return nil
}

func checkArchiveTask(args map[string]any) error {
	if monitors.IsAutoArchiveFlowRunning() {
		return h.HttpError(http.StatusConflict, "自动回收流程正在执行，暂不允许触发archive任务")
	}
	return checkMustHaveActiveDeployedRunningInstance(args)
}
