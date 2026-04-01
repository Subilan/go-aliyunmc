package tasks

func backupTask(tc *TaskContext) error {
	ip, err := getRemoteIPFromDBPlaceholder()
	if err != nil {
		return err
	}

	script, err := renderScriptTemplate(BackupC.TemplatePath, BackupC.Vars)
	if err != nil {
		return err
	}

	err = executeRemoteScript(tc.Context(), ip, BackupC.SSH, script, tc.println)
	if err != nil {
		return err
	}

	return nil
}
