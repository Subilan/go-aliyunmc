package tasks

import "go-aliyunmc/store"

func backupTask(tc *TaskContext, _ map[string]any) error {
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}

	script, err := renderScriptTemplate(BackupC.TemplatePath, BackupC.Vars)
	if err != nil {
		return err
	}

	err = executeRemoteScript(tc.Context(), ip, BackupC.SSH, script, tc.println, true)
	if err != nil {
		return err
	}

	return nil
}

func checkBackupTask(args map[string]any) error {
	return checkMustHaveActiveDeployedInstance(args)
}
