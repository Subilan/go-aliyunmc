package tasks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go-aliyunmc/aliyun"
	"go-aliyunmc/h"
	"go-aliyunmc/remote_util"
	"go-aliyunmc/store"
)

// deployTaskVars 表示部署任务实际所需要的模板变量，包含了DeployTemplateVars中的字段以及从aliyun.C读取的相关字段。
type deployTaskVars struct {
	DeployTemplateVars
	Username        string
	Password        string
	RegionId        string
	AccessKeyId     string
	AccessKeySecret string
	DataDiskSize    int
}

type deployStep struct {
	scriptPath string
	timeoutSec int
}

// deploySteps 定义了部署任务的各个步骤，包括每个步骤对应的脚本模板路径和执行超时时间（秒）
var deploySteps = []deployStep{
	{scriptPath: "scripts/deploy.create-user.tmpl.sh", timeoutSec: 30},
	{scriptPath: "scripts/deploy.setup-ssh-authorized-keys.tmpl.sh", timeoutSec: 30},
	{scriptPath: "scripts/deploy.configure-apt-sources.tmpl.sh", timeoutSec: 120},
	{scriptPath: "scripts/deploy.setup-java-repo.tmpl.sh", timeoutSec: 120},
	{scriptPath: "scripts/deploy.install-system-packages.tmpl.sh", timeoutSec: 180},
	{scriptPath: "scripts/deploy.install-ossutil.tmpl.sh", timeoutSec: 60},
	{scriptPath: "scripts/deploy.format-and-mount-data-disk.tmpl.sh", timeoutSec: 60},
	{scriptPath: "scripts/deploy.restore-archive-data.tmpl.sh", timeoutSec: 420},
}

func deployTask(tc *TaskContext, args map[string]any) error {
	// 由于这是一个分步任务，先将步骤推进到第一步，以便在日志中正确显示当前步骤信息
	tc.nextStep()

	ip, err := store.GetActiveInstanceIpNonEmpty()

	if err != nil {
		return err
	}

	// 构建扩展的模板变量，包含从aliyun.C读取的值
	expandedVars := deployTaskVars{
		DeployTemplateVars: DeployC.Vars,
		Username:           "mc",
		Password:           aliyun.C.Ecs.ProdPassword,
		RegionId:           aliyun.C.RegionId,
		AccessKeyId:        aliyun.C.AccessKeyId,
		AccessKeySecret:    aliyun.C.AccessKeySecret,
		DataDiskSize:       aliyun.C.Ecs.DataDisk.Size,
	}

	for _, step := range deploySteps {
		tc.println(fmt.Sprintf("[deploy] 执行步骤 %d/%d", tc.step, len(deploySteps)))

		script, renderErr := remote_util.RenderScriptTemplate(step.scriptPath, expandedVars)
		if renderErr != nil {
			return fmt.Errorf("渲染部署步骤%d脚本失败: %w", tc.step, renderErr)
		}

		stepCtx, cancel := context.WithTimeout(tc.Context(), time.Duration(step.timeoutSec)*time.Second)
		execErr := remote_util.ExecuteScriptRemote(script, ip, stepCtx, tc.println, true)
		cancel()
		if execErr != nil {
			// 如果stepCtx超时或被取消，返回特定的错误信息；否则返回一般的执行错误
			if errors.Is(execErr, context.DeadlineExceeded) {
				tc.println(fmt.Sprintf("[deploy] 步骤%d执行超时", tc.step))
			}
			if errors.Is(execErr, context.Canceled) {
				tc.println(fmt.Sprintf("[deploy] 执行步骤%d时被取消", tc.step))
			}
			return execErr
		}

		tc.nextStep()
	}

	err = store.SetActiveDeployed()

	if err != nil {
		tc.println("[deploy] 警告：无法更新实例的部署状态，请联系管理员以进行进一步操作")
		tc.println(err.Error())
	}

	tc.println("[deploy] 部署任务完成")
	return nil
}

func checkDeployTask(args map[string]any) error {
	if err := checkMustHaveActiveInstance(args); err != nil {
		return err
	}

	activeInstance, err := store.GetActiveInstance()

	if err != nil {
		return err
	}

	if activeInstance.IsDeployed {
		return h.HttpError(http.StatusConflict, "当前实例已部署完成，无需重复部署")
	}

	return nil
}
