package tasks

import "go-aliyunmc/store"

func archiveTask(tc *TaskContext, _ map[string]any) error {
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}

	script, err := renderScriptTemplate(ArchiveC.TemplatePath, ArchiveC.Vars)
	if err != nil {
		return err
	}

	err = executeRemoteScript(tc.Context(), ip, ArchiveC.SSH, script, tc.println, true)
	if err != nil {
		return err
	}

	return nil
}
