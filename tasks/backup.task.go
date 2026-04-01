package tasks

import (
	"fmt"
)

func backupTask(tc *TaskContext) error {
	tc.println("[backup] 准备目标地址")
	ip, err := getRemoteIPFromDBPlaceholder()
	if err != nil {
		return err
	}

	tc.println("[backup] 渲染备份脚本模板")
	script, err := renderScriptTemplate(BackupC.TemplatePath, BackupC.Vars)
	if err != nil {
		return err
	}

	tc.println(fmt.Sprintf("[backup] 连接远程机器 %s", ip))
	tc.println("[backup] 执行备份脚本")
	err = executeRemoteScript(tc.Context(), ip, BackupC.SSH, script, tc.println)
	if err != nil {
		return err
	}

	tc.println("[backup] 备份脚本执行完成")
	return nil
}
