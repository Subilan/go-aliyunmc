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

type deployStepType int

const (
	deployStepScript deployStepType = iota // 在远程服务器上执行的shell脚本
	deployStepLocal                        // 在当前后端执行的Go函数
)

// deployStep 表示部署任务的一个步骤
type deployStep struct {
	stepType   deployStepType
	script     string                                            // 脚本步骤：模板路径
	localFn    func(*TaskContext, string, context.Context) error // 本地步骤：执行函数
	timeoutSec int
}

const scriptBasePath = "scripts/"

// deploySteps 定义了部署任务的各个步骤，包括每个步骤对应的脚本模板路径和执行超时时间（秒）
var deploySteps = []deployStep{
	{script: "deploy.create-user.tmpl.sh", timeoutSec: 30},
	{script: "deploy.setup-ssh-authorized-keys.tmpl.sh", timeoutSec: 30},
	{script: "deploy.configure-apt-sources.tmpl.sh", timeoutSec: 120},
	{script: "deploy.setup-java-repo.tmpl.sh", timeoutSec: 120},
	{script: "deploy.install-system-packages.tmpl.sh", timeoutSec: 180},
	{script: "deploy.install-ossutil.tmpl.sh", timeoutSec: 60},
	{script: "deploy.format-and-mount-data-disk.tmpl.sh", timeoutSec: 60},
	{script: "deploy.restore-archive-data.tmpl.sh", timeoutSec: 420},
	{stepType: deployStepLocal, localFn: deployCopyWssCert, timeoutSec: 30},
}

func deployCopyWssCert(tc *TaskContext, ip string, ctx context.Context) error {
	if DeployC.Vars.WssCertSrc == "" || DeployC.Vars.WssCertDst == "" {
		tc.println("[deploy] WSS证书路径未配置，跳过证书上传步骤")
		return nil
	}

	tc.println(fmt.Sprintf("[deploy] 上传WSS证书: %s", DeployC.Vars.WssCertSrc))
	remotePath := DeployC.Vars.WssCertDst

	if err := remote_util.RsyncUpload(ctx, DeployC.Vars.WssCertSrc, remotePath, ip); err != nil {
		return fmt.Errorf("上传WSS证书失败: %w", err)
	}

	chownCmd := fmt.Sprintf("chown mc:mc %s && chmod 600 %s", remotePath, remotePath)
	if err := remote_util.ExecuteRemoteCommand(chownCmd, ip, ctx, tc.println, true); err != nil {
		return fmt.Errorf("设置WSS证书权限失败: %w", err)
	}

	tc.println("[deploy] WSS证书上传完成")
	return nil
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

		stepCtx, cancel := context.WithTimeout(tc.Context(), time.Duration(step.timeoutSec)*time.Second)
		var execErr error

		switch step.stepType {
		case deployStepScript:
			script, renderErr := remote_util.RenderScriptTemplate(scriptBasePath+step.script, expandedVars)
			if renderErr != nil {
				cancel()
				return fmt.Errorf("渲染部署步骤%d脚本失败: %w", tc.step, renderErr)
			}
			execErr = remote_util.ExecuteScriptRemote(script, ip, stepCtx, tc.println, true)
		case deployStepLocal:
			execErr = step.localFn(tc, ip, stepCtx)
		}
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
