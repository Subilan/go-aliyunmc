package tasks

func archiveTask(tc *TaskContext, _ map[string]any) error {
	ip, err := getRemoteIPFromDBPlaceholder()
	if err != nil {
		return err
	}

	script, err := renderScriptTemplate(ArchiveC.TemplatePath, ArchiveC.Vars)
	if err != nil {
		return err
	}

	err = executeRemoteScript(tc.Context(), ip, ArchiveC.SSH, script, tc.println)
	if err != nil {
		return err
	}

	return nil
}
