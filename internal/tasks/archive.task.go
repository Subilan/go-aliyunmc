package tasks

import (
	"github.com/Subilan/go-aliyunmc/internal/config"
	"github.com/Subilan/go-aliyunmc/internal/remote_util"
	"github.com/Subilan/go-aliyunmc/internal/store"
)

func archiveTask(tc *TaskContext, args map[string]any) error {
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}

	script, err := remote_util.RenderScriptTemplate(scriptTemplates, archiveTemplateName, config.Get().Tasks.Archive.Vars)
	if err != nil {
		return err
	}

	err = remote_util.Root(ip).ExecScript(tc.Context(), script, tc.println)
	if err != nil {
		return err
	}

	return nil
}

func checkArchiveTask(args map[string]any) error {
	return checkMustHaveActiveDeployedRunningInstance(args)
}
