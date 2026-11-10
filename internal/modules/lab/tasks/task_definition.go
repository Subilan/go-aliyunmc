package tasks

import (
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
)

type TaskDefinition struct {
	// Timeout 是任务的超时时间，0 表示不超时。
	Timeout time.Duration `json:"timeout"`

	// Type 是任务类型，与执行函数 F 一一对应。
	Type labmodels.TaskType `json:"type"`

	// Exclusive 为 true 时，同一 ExclusiveGroup 的任务互斥。
	// 互斥由 Executor.RunTask 统一执行。
	Exclusive bool `json:"exclusive"`

	// ExclusiveGroup 让多个类型共享同一资源的任务互斥，为空时等于 Type。
	// 例如创建实例、部署、启动服务器和启动游戏共用 [TaskExclusiveGroupDeployChain]。
	ExclusiveGroup labmodels.TaskType `json:"exclusiveGroup"`

	// Role 是执行该任务所需的最低 lab 权限，未填时等于 perm_util.RoleBasic。
	Role perm_util.Role `json:"role"`

	// F 是任务的执行函数，在独立 goroutine 中运行，返回 error 或 panic 表示失败。
	F TaskFunc `json:"-"`

	// C 在任务执行前检查输入参数与环境。
	C TaskCheckFunc `json:"-"`

	// E 在任务执行前检查执行者是否具有相应权限。
	E TaskEnforcerFunc `json:"-"`
}

type TaskEnforcerFunc func(perm_util.Role, map[string]any) error

type TaskFunc func(*TaskContext, map[string]any) error

// TaskExclusiveGroupDeployChain 是部署链路的互斥分组。
const TaskExclusiveGroupDeployChain labmodels.TaskType = "deploy_chain"

var TaskDefinitions = make(map[labmodels.TaskType]*TaskDefinition)

// exclusiveKey 返回该任务在独占执行器表中的键。
// 没有任何互斥要求时返回空字符串。
func (d *TaskDefinition) exclusiveKey() labmodels.TaskType {
	if !d.Exclusive {
		return ""
	}
	if d.ExclusiveGroup != "" {
		return d.ExclusiveGroup
	}
	return d.Type
}

func GetTaskDefinition(taskType labmodels.TaskType) *TaskDefinition {
	def, ok := TaskDefinitions[taskType]
	if !ok {
		return nil
	}
	return def
}

func TryGetTaskDefinition(taskType string) (*TaskDefinition, bool) {
	def, ok := TaskDefinitions[labmodels.TaskType(taskType)]
	return def, ok
}

// RegisterTaskDefinitions 注册内置任务定义，需要在 tasks.MustLoadConfig() 之后调用。
func RegisterTaskDefinitions() {
	TaskDefinitions[labmodels.TaskTypeTest] = &TaskDefinition{
		Exclusive: true,
		Type:      labmodels.TaskTypeTest,
		Timeout:   1 * time.Hour,
		F:         testTask,
	}

	TaskDefinitions[labmodels.TaskTypeDeploy] = &TaskDefinition{
		Exclusive:      true,
		ExclusiveGroup: TaskExclusiveGroupDeployChain,
		Type:           labmodels.TaskTypeDeploy,
		Timeout:        0, // 不设置超时，由任务内部逻辑控制
		F:              deployTask,
		C:              checkDeployTask,
	}

	TaskDefinitions[labmodels.TaskTypeBackup] = &TaskDefinition{
		Exclusive: config.Get().Tasks.Backup.Exclusive,
		Type:      labmodels.TaskTypeBackup,
		Timeout:   time.Duration(config.Get().Tasks.Backup.TimeoutSec) * time.Second,
		F:         backupTask,
		C:         checkBackupTask,
		Role:      perm_util.RoleOperator,
	}

	TaskDefinitions[labmodels.TaskTypeArchive] = &TaskDefinition{
		Exclusive: true,
		Type:      labmodels.TaskTypeArchive,
		Timeout:   time.Duration(config.Get().Tasks.Archive.TimeoutSec) * time.Second,
		F:         archiveTask,
		C:         checkArchiveTask,
		Role:      perm_util.RoleOperator,
	}

	TaskDefinitions[labmodels.TaskTypeCreateInstance] = &TaskDefinition{
		Exclusive:      true,
		ExclusiveGroup: TaskExclusiveGroupDeployChain,
		Type:           labmodels.TaskTypeCreateInstance,
		Timeout:        10 * time.Minute,
		F:              createInstanceTask,
		C:              checkCreateInstanceTask,
		E:              enforceCreateInstanceTask,
	}

	TaskDefinitions[labmodels.TaskTypeOneClickStartGame] = &TaskDefinition{
		Exclusive:      true,
		ExclusiveGroup: TaskExclusiveGroupDeployChain,
		Type:           labmodels.TaskTypeOneClickStartGame,
		Timeout:        0, // 与 deploy 任务一致：各步骤自带超时，整体不设上限
		F:              oneClickStartGameTask,
		E:              enforceOneClickStartGameTask,
	}

	TaskDefinitions[labmodels.TaskTypeStartServer] = &TaskDefinition{
		Exclusive:      true,
		ExclusiveGroup: TaskExclusiveGroupDeployChain,
		Type:           labmodels.TaskTypeStartServer,
		Timeout:        3 * time.Minute,
		F:              startServerTask,
		C:              checkStartServerTask,
	}
}
