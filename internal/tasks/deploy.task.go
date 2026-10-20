package tasks

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/config"
	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/log_util"
	"github.com/Subilan/go-aliyunmc/internal/remote_util"
	"github.com/Subilan/go-aliyunmc/internal/store"
)

// deployTaskVars 表示部署任务实际所需要的模板变量，包含了 config.TasksDeployVars 中的字段，
// 以及从配置其他节读取或推导出来的字段。
type deployTaskVars struct {
	config.TasksDeployVars
	// SSHGomcPublicKey 由 remote_util 从 mc 私钥推导，不来自配置。
	SSHGomcPublicKey string
	Username         string
	Password         string
	RegionId         string
	AccessKeyId      string
	AccessKeySecret  string
	DataDiskSize     int
}

type deployStepType int

const (
	deployStepScript deployStepType = iota // 在远程服务器上执行的shell脚本
	deployStepLocal                        // 在当前后端执行的Go函数
)

// deployStep 表示部署任务的一个步骤
type deployStep struct {
	name       string // 步骤名称，用于日志与失败点记录
	stepType   deployStepType
	script     string                                                       // 脚本步骤：模板路径
	localFn    func(*TaskContext, *remote_util.Conn, context.Context) error // 本地步骤：执行函数
	timeoutSec int
}

// 部署状态文件位于实例上，记录最近一次部署的结果与失败步骤。
const (
	deployStateDir  = "/var/lib/aliyunmc"
	deployStateFile = deployStateDir + "/deploy-state"
)

// deploySteps 定义了部署任务的各个步骤，包括每个步骤对应的脚本模板路径和执行超时时间（秒）。
// 所有脚本都必须保持幂等：部署失败后可以直接重新触发，已完成的步骤会跳过或安全覆盖。
var deploySteps = []deployStep{
	{name: "创建部署用户", script: "deploy.create-user.tmpl.sh", timeoutSec: 30},
	{name: "配置SSH公钥", script: "deploy.setup-ssh-authorized-keys.tmpl.sh", timeoutSec: 30},
	{name: "配置APT源", script: "deploy.configure-apt-sources.tmpl.sh", timeoutSec: 120},
	{name: "配置Java源", script: "deploy.setup-java-repo.tmpl.sh", timeoutSec: 120},
	{name: "安装系统软件包", script: "deploy.install-system-packages.tmpl.sh", timeoutSec: 180},
	{name: "安装ossutil", script: "deploy.install-ossutil.tmpl.sh", timeoutSec: 60},
	{name: "格式化并挂载数据盘", script: "deploy.format-and-mount-data-disk.tmpl.sh", timeoutSec: 60},
	{name: "恢复归档数据", script: "deploy.restore-archive-data.tmpl.sh", timeoutSec: 420},
	{name: "上传WSS证书", stepType: deployStepLocal, localFn: deployCopyWssCert, timeoutSec: 30},
}

// recordDeployState 把部署结果写入实例上的状态文件，便于在实例上直接查看失败点。
// 这是 best-effort 操作：连接已断开或任务已取消时只记录服务端日志。
func recordDeployState(conn *remote_util.Conn, content string) {
	if conn == nil {
		return
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	cmd := fmt.Sprintf(
		"mkdir -p %s && printf '%%s' '%s' | base64 -d > %s",
		deployStateDir,
		encoded,
		deployStateFile,
	)

	// 不复用任务 context：步骤超时或任务被取消时，任务 context 已经不可用，
	// 但失败点仍然需要尽量写入。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.Exec(ctx, cmd, func(string) {}); err != nil {
		log_util.Warn("写入部署状态文件失败: %v", err)
	}
}

// recordDeployFailure 记录部署失败点：步骤序号、步骤名称、错误与时间。
func recordDeployFailure(conn *remote_util.Conn, stepNo int, stepName, stepType string, cause error) {
	content := fmt.Sprintf(
		"status=failed\nstep=%d/%d\nname=%s\ntype=%s\nerror=%s\ntime=%s\n",
		stepNo,
		len(deploySteps),
		stepName,
		stepType,
		cause.Error(),
		time.Now().Format(time.RFC3339),
	)
	recordDeployState(conn, content)
}

func deployCopyWssCert(tc *TaskContext, conn *remote_util.Conn, ctx context.Context) error {
	if config.Get().Tasks.Deploy.Vars.WssCertSrc == "" || config.Get().Tasks.Deploy.Vars.WssCertDst == "" {
		tc.println("[deploy] WSS证书路径未配置，跳过证书上传步骤")
		return nil
	}

	tc.println(fmt.Sprintf("[deploy] 上传WSS证书: %s", config.Get().Tasks.Deploy.Vars.WssCertSrc))
	remotePath := config.Get().Tasks.Deploy.Vars.WssCertDst

	// 证书上传走 rsync，身份固定为 mc，与 root 连接无关。
	if err := remote_util.Mc(conn.Target().IP).Upload(ctx, config.Get().Tasks.Deploy.Vars.WssCertSrc, remotePath, tc.println); err != nil {
		return fmt.Errorf("上传WSS证书失败: %w", err)
	}

	chownCmd := fmt.Sprintf("chown mc:mc %s && chmod 600 %s", remotePath, remotePath)
	if err := conn.Exec(ctx, chownCmd, tc.println); err != nil {
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

	// 授权公钥从 mc 私钥推导。
	authorizedKey, err := remote_util.AuthorizedKeyLine()
	if err != nil {
		return err
	}

	// 构建扩展的模板变量：包含配置里的部署变量，以及从其他配置节读取或推导出来的值。
	expandedVars := deployTaskVars{
		TasksDeployVars:  config.Get().Tasks.Deploy.Vars,
		SSHGomcPublicKey: authorizedKey,
		Username:         "mc",
		Password:         config.Get().Aliyun.Ecs.ProdPassword,
		RegionId:         config.Get().Aliyun.RegionId,
		AccessKeyId:      config.Get().Aliyun.AccessKeyId,
		AccessKeySecret:  config.Get().Aliyun.AccessKeySecret,
		DataDiskSize:     config.Get().Aliyun.Ecs.DataDisk.Size,
	}

	// 整个部署过程复用同一条 root 连接。
	conn, err := remote_util.Root(ip).Connect(tc.Context())
	if err != nil {
		return err
	}
	defer conn.Close()

	for i, step := range deploySteps {
		stepNo := i + 1
		tc.println(fmt.Sprintf("[deploy] 执行步骤 %d/%d「%s」", stepNo, len(deploySteps), step.name))

		stepCtx, cancel := context.WithTimeout(tc.Context(), time.Duration(step.timeoutSec)*time.Second)
		var execErr error

		switch step.stepType {
		case deployStepScript:
			script, renderErr := remote_util.RenderScriptTemplate(scriptTemplates, scriptDir+step.script, expandedVars)
			if renderErr != nil {
				cancel()
				wrapped := fmt.Errorf("部署步骤 %d/%d「%s」渲染脚本失败: %w", stepNo, len(deploySteps), step.name, renderErr)
				recordDeployFailure(conn, stepNo, step.name, "render", wrapped)
				return wrapped
			}
			execErr = conn.ExecScript(stepCtx, script, tc.println)
		case deployStepLocal:
			execErr = step.localFn(tc, conn, stepCtx)
		}
		cancel()
		if execErr != nil {
			stepType := "script"
			if step.stepType == deployStepLocal {
				stepType = "local"
			}

			// 如果stepCtx超时或被取消，返回特定的错误信息；否则返回一般的执行错误
			if errors.Is(execErr, context.DeadlineExceeded) {
				tc.println(fmt.Sprintf("[deploy] 步骤%d执行超时", stepNo))
			}
			if errors.Is(execErr, context.Canceled) {
				tc.println(fmt.Sprintf("[deploy] 执行步骤%d时被取消", stepNo))
			}

			wrapped := fmt.Errorf("部署步骤 %d/%d「%s」失败: %w", stepNo, len(deploySteps), step.name, execErr)
			recordDeployFailure(conn, stepNo, step.name, stepType, wrapped)
			tc.println("[deploy] " + wrapped.Error())
			return wrapped
		}

		tc.nextStep()
	}

	recordDeployState(conn, fmt.Sprintf("status=success\ntime=%s\n", time.Now().Format(time.RFC3339)))

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
