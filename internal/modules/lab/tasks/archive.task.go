package tasks

import (
	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/remote_util"
)

func archiveTask(tc *TaskContext, args map[string]any) error {
	ip, err := dao.GetActiveInstanceIpNonEmpty()
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
