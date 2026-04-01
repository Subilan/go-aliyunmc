package tasks

import (
	"fmt"
)

func archiveTask(tc *TaskContext) error {
	tc.println("[archive] 准备目标地址")
	ip, err := getRemoteIPFromDBPlaceholder()
	if err != nil {
		return err
	}

	tc.println("[archive] 渲染归档脚本模板")
	script, err := renderScriptTemplate(ArchiveC.TemplatePath, ArchiveC.Vars)
	if err != nil {
		return err
	}

	tc.println(fmt.Sprintf("[archive] 连接远程机器 %s", ip))
	tc.println("[archive] 执行归档脚本")
	err = executeRemoteScript(tc.Context(), ip, ArchiveC.SSH, script, tc.println)
	if err != nil {
		return err
	}

	tc.println("[archive] 归档脚本执行完成")
	return nil
}
