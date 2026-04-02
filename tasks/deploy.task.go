package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-aliyunmc/aliyun"
)

func deployTask(tc *TaskContext, args map[string]any) error {
	// 由于这是一个分步任务，先将步骤推进到第一步，以便在日志中正确显示当前步骤信息
	tc.nextStep()

	ip, err := getRemoteIPFromDBPlaceholder()
	if err != nil {
		return err
	}

	// 构建扩展的模板变量，包含从aliyun.C读取的值
	expandedVars := struct {
		DeployTemplateVars
		RegionId        string
		AccessKeyId     string
		AccessKeySecret string
		DataDiskSize    int
	}{
		DeployTemplateVars: DeployC.Vars,
		RegionId:           aliyun.C.RegionId,
		AccessKeyId:        aliyun.C.AccessKeyId,
		AccessKeySecret:    aliyun.C.AccessKeySecret,
		DataDiskSize:       aliyun.C.Ecs.DataDisk.Size,
	}

	for _, step := range DeployC.Steps {
		tc.println(fmt.Sprintf("[deploy] 执行步骤 %d/%d: %s", tc.step, len(DeployC.Steps), step.Name))

		script, renderErr := renderScriptTemplate(step.ScriptPath, expandedVars)
		if renderErr != nil {
			return fmt.Errorf("渲染部署步骤%d脚本失败: %w", tc.step, renderErr)
		}

		stepCtx, cancel := context.WithTimeout(tc.Context(), time.Duration(step.TimeoutSec)*time.Second)
		execErr := executeRemoteScript(stepCtx, ip, DeployC.SSH, script, tc.println)
		cancel()
		if execErr != nil {
			if errors.Is(execErr, context.DeadlineExceeded) {
				return fmt.Errorf("执行部署步骤%d超时(%ds)", tc.step, step.TimeoutSec)
			}
			if errors.Is(execErr, context.Canceled) {
				return execErr
			}
			return fmt.Errorf("执行部署步骤%d失败: %w", tc.step, execErr)
		}

		tc.nextStep()
	}

	tc.println("[deploy] 部署任务完成")
	return nil
}
