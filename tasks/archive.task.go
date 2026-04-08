package tasks

import (
	"go-aliyunmc/store"
)

func archiveTask(tc *TaskContext, _ map[string]any) error {
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}

	script, err := renderScriptTemplate(ArchiveC.TemplatePath, ArchiveC.Vars)
	if err != nil {
		return err
	}

	err = executeRemoteScript(script, scriptExecParams{
		Ctx:    tc.Context(),
		IP:     ip,
		Cfg:    ArchiveC.SSH,
		OnLine: tc.println,
		Root:   true,
	})
	if err != nil {
		return err
	}

	return nil
}

func checkArchiveTask(args map[string]any) error {
	return checkMustHaveActiveDeployedRunningInstance(args)
}
