package tasks

import (
	"go-aliyunmc/remote_util"
	"go-aliyunmc/store"
)

func archiveTask(tc *TaskContext, args map[string]any) error {
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
	return checkMustHaveActiveDeployedRunningInstance(args)
}
